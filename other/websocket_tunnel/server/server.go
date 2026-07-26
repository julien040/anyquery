package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/hostrouter"
	"github.com/olahol/melody"
	"github.com/puzpuzpuz/xsync/v3"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var dbSchema string

//go:embed redirect_oauth.html
var oauthHTML string

// gptTunnelIDPath matches the gpt-facing routes that carry the tunnel ID
// (the bearer-equivalent capability token for the gpt HTTP API) as their
// first path segment: /{id}/list-tables, /{id}/describe-table and
// /{id}/execute-query. Matching is done by path shape rather than by Host
// header so the redaction still applies if the router is ever mounted
// under another hostname.
//
// accessLogger runs before middleware.StripSlashes (see start()), so it
// sees the raw, un-normalized path - a trailing slash
// (e.g. "/xyz/execute-query/") must still match, otherwise the ID would be
// logged verbatim for exactly the requests StripSlashes exists to
// normalize. The action name is matched case-insensitively for the same
// reason: chi's routing is case-sensitive and would 404 "/xyz/Execute-Query",
// but the ID must not leak into the log on the way to that 404.
var gptTunnelIDPath = regexp.MustCompile(`(?i)^/[^/]+/(?:list-tables|describe-table|execute-query)/?$`)

// redactAccessLogPath returns path with its leading /{id} segment replaced
// by a placeholder if path matches one of the gpt-facing tunnel routes,
// and returns path unchanged otherwise.
func redactAccessLogPath(path string) string {
	if !gptTunnelIDPath.MatchString(path) {
		return path
	}
	_, rest, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return "/[REDACTED]/" + rest
}

// accessLogger is a stand-in for chi's middleware.Logger. It logs in the
// same shape (method, scheme, host, path, proto, remote addr, status,
// bytes written, duration) to stdout, but with redactAccessLogPath applied
// to the request path so the tunnel ID never lands in the access log -
// server.log otherwise captures a live capability token in plaintext on
// every request/response pair forwarded to a victim's machine. It logs
// r.URL.Path rather than chi's default r.RequestURI, which as a side
// effect also keeps the tunnel_id query parameter of the websocket-upgrade
// request (/websocket-anyquery?tunnel_id=...) out of the log - the same
// capability token, leaked through the same mechanism, just not the path
// this fix was asked to cover.
func accessLogger(next http.Handler) http.Handler {
	out := log.New(os.Stdout, "", log.LstdFlags)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		defer func() {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			out.Printf("%q from %s - %d %dB in %s",
				fmt.Sprintf("%s %s://%s%s %s", r.Method, scheme, r.Host, redactAccessLogPath(r.URL.Path), r.Proto),
				r.RemoteAddr, ww.Status(), ww.BytesWritten(), time.Since(start))
		}()

		next.ServeHTTP(ww, r)
	})
}

type server struct {
	dbWriteable bool
	addr        string
	dbPath      string
	logger      *slog.Logger
	db          *sql.DB
	melody      *melody.Melody

	sessions *xsync.MapOf[string, *melody.Session]
}

func newServer(dbWriteable bool, addr string, dbPath string,
	logger *slog.Logger) *server {
	return &server{
		dbWriteable: dbWriteable,
		addr:        addr,
		dbPath:      dbPath,
		logger:      logger,
	}
}

func (s *server) start() error {
	// Open the database
	databaseURI := strings.Builder{}
	databaseURI.WriteString("file:")
	databaseURI.WriteString(s.dbPath)
	if s.dbWriteable {
		databaseURI.WriteString("?mode=rwc")
	} else {
		// Check if the file exists
		_, err := os.Stat(s.dbPath)
		if err == os.ErrNotExist {
			return fmt.Errorf("database file does not exist. Open the server in writeable mode (-w) to create a new database")
		}
		databaseURI.WriteString("?mode=ro")
	}

	// Open the database
	s.logger.Info("Opening database", "uri", databaseURI.String())
	var err error
	s.db, err = sql.Open("sqlite", databaseURI.String())
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}
	defer s.db.Close()

	// Create the table if it doesn't exist
	_, err = s.db.Exec(dbSchema)
	if err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}

	r := chi.NewRouter()
	//r.Use(cors.AllowAll().Handler)
	r.Use(middleware.Recoverer)
	// accessLogger replaces middleware.Logger: the gpt-facing routes carry
	// the tunnel ID (a bearer-equivalent capability token) as a URL path
	// segment, and the default request logger would otherwise write it in
	// plaintext to stdout/server.log on every single request/response pair
	// forwarded to a victim's machine - the most realistic leak vector for
	// this token. All other routes are logged exactly as before.
	r.Use(accessLogger)
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(middleware.GetHead)
	r.Use(middleware.StripSlashes)

	hr := hostrouter.New()
	hr.Map("tunnel.anyquery.xyz", s.tunnelRouter())
	hr.Map("gpt.anyquery.xyz", s.gptRouter())
	hr.Map("gpt-actions.anyquery.xyz", s.gptRouter())

	r.Mount("/", hr)

	// Setup the websocket handler
	s.melody = melody.New()
	s.melody.Config.ConcurrentMessageHandling = true
	s.melody.Config.MessageBufferSize = 1024
	s.melody.Config.MaxMessageSize = 1024 * 1024 // 1MB
	s.melody.Config.PongWait = time.Second * 90  // Wait 60 seconds for pong, otherwise close the connection
	s.melody.Config.PingPeriod = time.Minute     // Send ping every 30 seconds

	r.Get("/websocket-anyquery", s.upgradeWS)
	s.melody.HandleConnect(s.handleConnectWS)
	s.melody.HandleDisconnect(s.handleDisconnectWS)
	s.melody.HandleMessage(s.handleMessage)

	// Create the sessions map
	s.sessions = xsync.NewMapOf[string, *melody.Session]()

	// Catch SIGINT and SIGTERM and close the database
	osChan := make(chan os.Signal, 1)
	signal.Notify(osChan, os.Interrupt)
	go func() {
		<-osChan
		s.logger.Info("Shutting down server")
		s.db.Close()

		// Disconnect all sessions
		s.sessions.Range(func(key string, value *melody.Session) bool {
			value.Close()
			return true
		})

		os.Exit(0)
	}()

	// Start the server
	s.logger.Info("Starting server", "addr", s.addr)

	return http.ListenAndServe(s.addr, r)
}

func (s *server) tunnelRouter() chi.Router {
	r := chi.NewRouter()
	r.Post("/tunnel/new", s.newTunnel)
	r.Post("/tunnel/oauth2/token", s.tunnelOauth2Token)
	r.Get("/tunnel/oauth2/redirect", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(oauthHTML))
	})
	return r
}

func (s *server) gptRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/{id}/list-tables", s.listTablesAPI)
	r.Post("/{id}/describe-table", s.describeTableAPI)
	r.Post("/{id}/execute-query", s.executeQueryAPI)
	return r
}
