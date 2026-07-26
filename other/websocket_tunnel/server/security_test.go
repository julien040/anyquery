package main

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/olahol/melody"
	"github.com/puzpuzpuz/xsync/v3"
)

// TestGenerateRandomIDLength ensures generated IDs have exactly the
// requested length.
func TestGenerateRandomIDLength(t *testing.T) {
	for _, n := range []int{1, 8, 16, 32} {
		id, err := generateRandomID(n)
		if err != nil {
			t.Fatalf("generateRandomID(%d) returned error: %v", n, err)
		}
		if len(id) != n {
			t.Fatalf("generateRandomID(%d) returned id of length %d: %q", n, len(id), id)
		}
	}
}

// TestGenerateRandomIDCharset ensures every character in a generated ID
// belongs to the intended alphabet.
func TestGenerateRandomIDCharset(t *testing.T) {
	id, err := generateRandomID(256)
	if err != nil {
		t.Fatalf("generateRandomID returned error: %v", err)
	}
	for _, c := range id {
		if !strings.ContainsRune(alphabet, c) {
			t.Fatalf("character %q in generated id is not part of the alphabet %q", c, alphabet)
		}
	}
}

// TestGenerateRandomIDUsesFullAlphabet is a regression test for the
// off-by-one bug where rand.IntN(len(alphabet)-1) meant the last letter of
// the alphabet ('z') could never be produced (the real keyspace was 25^n
// instead of the intended 26^n). It draws a large, statistically
// significant sample and asserts every letter of the alphabet - including
// 'z' - appears at least once.
func TestGenerateRandomIDUsesFullAlphabet(t *testing.T) {
	seen := make(map[rune]bool)

	// Draw enough characters that the absence of any single letter is not
	// plausibly due to chance: with a uniform 26-letter alphabet, the
	// probability that a specific letter is missing from 10,000 draws is
	// (25/26)^10000, which is astronomically small (~1e-166). If the
	// off-by-one bug were reintroduced, 'z' would be missing every time.
	const totalChars = 10_000
	const idLen = 32

	drawn := 0
	for drawn < totalChars {
		id, err := generateRandomID(idLen)
		if err != nil {
			t.Fatalf("generateRandomID returned error: %v", err)
		}
		for _, c := range id {
			seen[c] = true
		}
		drawn += idLen
	}

	for _, c := range alphabet {
		if !seen[c] {
			t.Errorf("letter %q never appeared across %d draws; full alphabet is not being used (off-by-one regression?)", c, drawn)
		}
	}
}

// TestGenerateRandomIDNotAllSame is a sanity check that consecutive calls
// don't produce identical IDs (i.e. the generator isn't stuck returning a
// constant value).
func TestGenerateRandomIDNotAllSame(t *testing.T) {
	first, err := generateRandomID(16)
	if err != nil {
		t.Fatalf("generateRandomID returned error: %v", err)
	}
	same := true
	for i := 0; i < 20; i++ {
		id, err := generateRandomID(16)
		if err != nil {
			t.Fatalf("generateRandomID returned error: %v", err)
		}
		if id != first {
			same = false
			break
		}
	}
	if same {
		t.Fatalf("generateRandomID produced the same value %q across 21 calls", first)
	}
}

// TestRedactAccessLogPathRedactsTunnelRoutes ensures the tunnel ID path
// segment is stripped from the three gpt-facing routes that carry it.
func TestRedactAccessLogPathRedactsTunnelRoutes(t *testing.T) {
	const tunnelID = "vrcuqmkfxyz12345"

	cases := []string{
		"/" + tunnelID + "/list-tables",
		"/" + tunnelID + "/describe-table",
		"/" + tunnelID + "/execute-query",
		// Trailing slash: accessLogger runs before middleware.StripSlashes
		// (see start() in server.go), so it sees the raw path a client
		// sends. Without the trailing "/?" in gptTunnelIDPath, this case
		// would fall through unredacted right up until StripSlashes
		// normalizes it downstream, and the ID would already be logged.
		"/" + tunnelID + "/execute-query/",
		// chi's routing is case-sensitive (this 404s), but the log line is
		// written regardless of the eventual route outcome.
		"/" + tunnelID + "/Execute-Query",
	}

	for _, path := range cases {
		redacted := redactAccessLogPath(path)
		if strings.Contains(redacted, tunnelID) {
			t.Errorf("redactAccessLogPath(%q) = %q still contains the tunnel id", path, redacted)
		}
		if !strings.Contains(redacted, "[REDACTED]") {
			t.Errorf("redactAccessLogPath(%q) = %q does not contain the redaction placeholder", path, redacted)
		}
	}
}

// TestRedactAccessLogPathLeavesOtherRoutesAlone ensures routes that don't
// carry the tunnel ID (or don't match the gpt-facing shape) are logged
// unchanged, so operational visibility for the rest of the server is
// preserved.
func TestRedactAccessLogPathLeavesOtherRoutesAlone(t *testing.T) {
	cases := []string{
		"/ping",
		"/tunnel/new",
		"/tunnel/oauth2/token",
		"/tunnel/oauth2/redirect",
		"/websocket-anyquery",
		"/list-tables", // missing the id segment entirely
	}

	for _, path := range cases {
		if got := redactAccessLogPath(path); got != path {
			t.Errorf("redactAccessLogPath(%q) = %q, want unchanged", path, got)
		}
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it. accessLogger builds its own *log.Logger
// pointed at os.Stdout (mirroring chi's DefaultLogFormatter), so this is
// the only way to observe, end to end, what actually lands in the
// process's stdout (and therefore server.log, since that's how the relay
// is run in production - stdout redirected to a file).
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return string(out)
}

// TestAccessLoggerRedactsLoggedLine is an end-to-end test of the real
// middleware (not just the helper): it sends an HTTP request carrying a
// tunnel ID through accessLogger and inspects what actually gets printed,
// confirming the token never reaches the log line while the rest of the
// request (method, route shape, status) is still visible for operational
// use.
func TestAccessLoggerRedactsLoggedLine(t *testing.T) {
	const tunnelID = "supersecrettunnelid42"

	req := httptest.NewRequest(http.MethodPost, "/"+tunnelID+"/execute-query", nil)
	rec := httptest.NewRecorder()

	output := captureStdout(t, func() {
		// accessLogger builds its *log.Logger from os.Stdout at
		// construction time, so the handler must be built inside the
		// capture window (after os.Stdout has been swapped for the pipe),
		// not before it.
		handler := accessLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rec, req)
	})

	if strings.Contains(output, tunnelID) {
		t.Fatalf("access log line leaked the tunnel id: %q", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Fatalf("access log line missing redaction placeholder: %q", output)
	}
	if !strings.Contains(output, "execute-query") {
		t.Fatalf("access log line lost operational context (route): %q", output)
	}
	if !strings.Contains(output, http.MethodPost) {
		t.Fatalf("access log line lost operational context (method): %q", output)
	}
}

// TestAccessLoggerKeepsNonTunnelPathsVisible confirms routes outside the
// gpt-facing tunnel API are still logged in full - the redaction must not
// degrade into "delete all logging".
func TestAccessLoggerKeepsNonTunnelPathsVisible(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	output := captureStdout(t, func() {
		handler := accessLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(rec, req)
	})

	if !strings.Contains(output, "/ping") {
		t.Fatalf("expected /ping to be logged in full, got: %q", output)
	}
	if strings.Contains(output, "[REDACTED]") {
		t.Fatalf("non-tunnel path should not be redacted, got: %q", output)
	}
}

// newTestServer builds a *server whose slog output is captured in buf, for
// exercising handleConnectWS/handleDisconnectWS/handleMessage without a
// real websocket connection.
func newTestServer(buf *bytes.Buffer) *server {
	return &server{
		logger:   slog.New(slog.NewTextHandler(buf, nil)),
		sessions: xsync.NewMapOf[string, *melody.Session](),
	}
}

// newTestSession builds a *melody.Session with just the exported fields
// (Keys, Request) that the handlers under test touch. The unexported
// fields (conn, output, ...) are left zero-valued, which is fine because
// none of handleConnectWS/handleDisconnectWS/handleMessage ever call a
// Session method that needs them - they only read/write se.Keys and
// se.Request.
func newTestSession(tunnelID string) *melody.Session {
	return &melody.Session{
		Request: httptest.NewRequest(http.MethodGet, "/websocket-anyquery?tunnel_id="+tunnelID, nil),
		Keys: map[string]interface{}{
			"tunnel_id": tunnelID,
			"requests":  xsync.NewMapOf[string, chan Response](),
		},
	}
}

// TestConnectDisconnectDoNotLogTunnelID is a regression test for the
// residual leak flagged after the initial fix: handleConnectWS and
// handleDisconnectWS log the connection lifecycle, and must not include
// the tunnel ID (the bearer-equivalent capability token) in that log line.
// It also checks the two lines share the same session correlation label,
// so an operator can still tell "this connect and this disconnect were the
// same session" without ever seeing the secret.
func TestConnectDisconnectDoNotLogTunnelID(t *testing.T) {
	const tunnelID = "supersecrettunnelid99"

	var buf bytes.Buffer
	s := newTestServer(&buf)
	se := newTestSession(tunnelID)

	s.handleConnectWS(se)
	connectLog := buf.String()
	buf.Reset()

	s.handleDisconnectWS(se)
	disconnectLog := buf.String()

	if strings.Contains(connectLog, tunnelID) {
		t.Fatalf("handleConnectWS logged the tunnel id: %q", connectLog)
	}
	if strings.Contains(disconnectLog, tunnelID) {
		t.Fatalf("handleDisconnectWS logged the tunnel id: %q", disconnectLog)
	}

	logID := sessionLogID(se)
	if logID == "" || logID == "unknown" {
		t.Fatalf("expected handleConnectWS to assign a real session log id, got %q", logID)
	}
	if !strings.Contains(connectLog, logID) {
		t.Fatalf("expected connect log to contain the session id %q, got: %q", logID, connectLog)
	}
	if !strings.Contains(disconnectLog, logID) {
		t.Fatalf("expected disconnect log to contain the session id %q, got: %q", logID, disconnectLog)
	}
}

// TestHandleMessageErrorPathsDoNotLogTunnelID is a regression test for the
// three s.logger.Error calls in handleMessage that used to log
// se.Keys["tunnel_id"] directly. Each subtest drives handleMessage down one
// of the three error branches and asserts the tunnel ID never reaches the
// log, while the session correlation label does (so the lines remain
// useful for grouping repeated errors from the same connection).
func TestHandleMessageErrorPathsDoNotLogTunnelID(t *testing.T) {
	const tunnelID = "anothersecrettunnelid77"

	t.Run("malformed JSON", func(t *testing.T) {
		var buf bytes.Buffer
		s := newTestServer(&buf)
		se := newTestSession(tunnelID)
		se.Keys["log_id"] = "corr-malformed"

		s.handleMessage(se, []byte("not json"))

		out := buf.String()
		if strings.Contains(out, tunnelID) {
			t.Fatalf("logged the tunnel id: %q", out)
		}
		if !strings.Contains(out, "corr-malformed") {
			t.Fatalf("expected session correlation id in log, got: %q", out)
		}
	})

	t.Run("missing request_id", func(t *testing.T) {
		var buf bytes.Buffer
		s := newTestServer(&buf)
		se := newTestSession(tunnelID)
		se.Keys["log_id"] = "corr-missing-reqid"

		s.handleMessage(se, []byte(`{"result":"ok"}`))

		out := buf.String()
		if strings.Contains(out, tunnelID) {
			t.Fatalf("logged the tunnel id: %q", out)
		}
		if !strings.Contains(out, "corr-missing-reqid") {
			t.Fatalf("expected session correlation id in log, got: %q", out)
		}
	})

	t.Run("unknown request_id", func(t *testing.T) {
		var buf bytes.Buffer
		s := newTestServer(&buf)
		se := newTestSession(tunnelID)
		se.Keys["log_id"] = "corr-unknown-reqid"

		s.handleMessage(se, []byte(`{"request_id":"does-not-exist","result":"ok"}`))

		out := buf.String()
		if strings.Contains(out, tunnelID) {
			t.Fatalf("logged the tunnel id: %q", out)
		}
		if !strings.Contains(out, "corr-unknown-reqid") {
			t.Fatalf("expected session correlation id in log, got: %q", out)
		}
	})
}

// TestSessionLogIDFallsBackWhenUnset ensures sessionLogID degrades safely
// (a fixed placeholder, not a panic and not the tunnel ID) if it is ever
// called before handleConnectWS has set "log_id".
func TestSessionLogIDFallsBackWhenUnset(t *testing.T) {
	se := &melody.Session{Keys: map[string]interface{}{"tunnel_id": "whatever"}}
	if got := sessionLogID(se); got != "unknown" {
		t.Fatalf("expected fallback placeholder, got %q", got)
	}
}
