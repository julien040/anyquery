package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
	r.Use(middleware.Logger)
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
