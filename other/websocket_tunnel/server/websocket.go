package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/olahol/melody"
	"github.com/puzpuzpuz/xsync/v3"
)

type Request struct {
	// The method to call (e.g. "execute-query", "list-tables", etc.)
	Method string `json:"method"`
	// The arguments to the method
	Args []interface{} `json:"args"`
	// A random ID (unique for the whole lifetime of the server) to identify the request
	RequestID string `json:"request_id"`
}

type Response struct {
	// The ID of the request this response is for
	RequestID string `json:"request_id"`
	// The result of the request
	Result interface{} `json:"result"`
	// An error message if the request failed
	Error string `json:"error"`
}

func (s *server) upgradeWS(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("Trying to upgrade connection", "remoteAddr", r.RemoteAddr)
	// Ensure the tunnel ID is provided
	params := r.URL.Query()
	tunnelID := params.Get("tunnel_id")
	if tunnelID == "" {
		s.logger.Debug("missing tunnel_id parameter")
		http.Error(w, "missing tunnel_id parameter", http.StatusBadRequest)
		return
	}

	// Get the bearer token
	bearerToken := r.Header.Get("Authorization")
	if bearerToken == "" {
		s.logger.Debug("missing Authorization header")
		http.Error(w, "missing Authorization header", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(bearerToken, "Bearer ") {
		s.logger.Debug("invalid Authorization header. Must be a bearer token")
		http.Error(w, "invalid Authorization header. Must be a bearer token", http.StatusBadRequest)
		return
	}

	// Get the tunnel from the database
	t, err := GetTunnel(s.db, tunnelID)
	if err != nil {
		s.logger.Debug("error getting tunnel", "error", err)
		if err == sql.ErrNoRows {
			http.Error(w, "tunnel not found. Make sure the tunnel ID is correct", http.StatusBadRequest)
			return
		}
		http.Error(w, "error getting tunnel. Make sure the tunnel ID is correct", http.StatusBadRequest)
		return
	}

	// Ensure the auth token matches
	if t.AuthToken != bearerToken[7:] {
		s.logger.Debug("invalid Authorization header. Make sure the bearer token is correct")
		http.Error(w, "invalid Authorization header. Make sure the bearer token is correct", http.StatusBadRequest)
		return
	}

	// Check if the tunnel has expired
	if t.ExpiresAt.Before(time.Now()) {
		s.logger.Debug("tunnel has expired. Restart Anyquery to get a new tunnel ID, and modify your LLM client to use the new tunnel ID")
		http.Error(w, "tunnel has expired. Restart Anyquery to get a new tunnel ID, and modify your LLM client to use the new tunnel ID", http.StatusBadRequest)
		return
	}

	// Check if the tunnel is already connected
	if _, ok := s.sessions.Load(tunnelID); ok {
		s.logger.Debug("tunnel already connected")
		http.Error(w, "tunnel already connected", http.StatusBadRequest)
		return
	}

	// Upgrade the connection
	s.logger.Info("Upgrading connection", "remoteAddr", r.RemoteAddr)
	s.melody.HandleRequestWithKeys(w, r, map[string]interface{}{
		"tunnel_id": tunnelID,
		"requests":  xsync.NewMapOf[string, chan Response](),
	})
}

func (s *server) handleConnectWS(se *melody.Session) {
	// logID is a random, non-secret label used ONLY for correlating log
	// lines belonging to the same live connection. Unlike the tunnel ID, it
	// carries no capability - it is generated fresh per connection and
	// bears no relationship to the tunnel ID (it isn't even derived from
	// it), so logging it cannot leak or help reconstruct the real secret.
	// See sessionLogID and the comment on the three s.logger.Error calls in
	// handleMessage below for why this replaced logging the tunnel ID
	// directly.
	logID, err := generateRandomIDWithNumbers(8)
	if err != nil {
		logID = "unknown"
	}
	se.Keys["log_id"] = logID
	s.logger.Info("New connection", "remoteAddr", se.Request.RemoteAddr, "session", sessionLogID(se))
	s.sessions.Store(se.Keys["tunnel_id"].(string), se)
}

func (s *server) handleDisconnectWS(se *melody.Session) {
	s.logger.Info("Connection closed", "remoteAddr", se.Request.RemoteAddr, "session", sessionLogID(se))
	s.sessions.Delete(se.Keys["tunnel_id"].(string))
}

// sessionLogID returns the non-secret per-connection correlation label set
// in handleConnectWS, falling back to a fixed placeholder if it is somehow
// missing (e.g. a message handled before handleConnectWS has run) rather
// than panicking or falling back to the tunnel ID.
func sessionLogID(se *melody.Session) string {
	if id, ok := se.Keys["log_id"].(string); ok {
		return id
	}
	return "unknown"
}

// Response from the client
func (s *server) handleMessage(se *melody.Session, msg []byte) {
	// Deserialize the message
	var response Response
	err := json.Unmarshal(msg, &response)
	if err != nil {
		// Logged by session correlation label, not by tunnel ID: the
		// tunnel ID is a bearer-equivalent capability token, and these
		// error paths fire on attacker/malformed input, which is exactly
		// when an operator is most likely to be looking at the logs.
		s.logger.Error("Error deserializing message", "error", err, "session", sessionLogID(se))
		return
	}

	// Make sure the request ID is provided
	if response.RequestID == "" {
		s.logger.Error("Request ID not provided", "session", sessionLogID(se))
		return
	}

	// Get the response channel
	requests := se.Keys["requests"].(*xsync.MapOf[string, chan Response])
	responseChan, ok := requests.Load(response.RequestID)
	if !ok {
		s.logger.Error("Response channel not found", "session", sessionLogID(se), "requestID", response.RequestID)
		return
	}

	// Delete it before sending: this is what stops a second response sharing
	// the same (attacker-controlled) request_id from finding this channel
	// again once we've claimed it.
	requests.Delete(response.RequestID)

	// Non-blocking: sendRequest's own timeout may have already fired and
	// deleted this same entry (a benign race with the Load above), in which
	// case sendRequest has stopped receiving from responseChan. The channel
	// is buffered (size 1, see sendRequest), so the first send here never
	// blocks even with no reader — but without this select/default, a
	// concurrent duplicate response (this code path racing with itself
	// before the Delete above lands) would try to send a second time to an
	// already-full buffer and block forever, leaking this goroutine.
	select {
	case responseChan <- response:
	default:
		s.logger.Debug("no receiver for response, dropping", "session", sessionLogID(se), "requestID", response.RequestID)
	}
}

const requestTimeout = 70 * time.Second

// Request to the client
func (s *server) retrieveSession(id string) (*melody.Session, error) {
	session, ok := s.sessions.Load(id)
	if !ok {
		return nil, fmt.Errorf("anyquery instance not connected. Make sure to start anyquery with anyquery gpt")
	}
	return session, nil
}

// sendRequest builds a Request for method/args, sends it over session, and
// waits up to requestTimeout for the matching response.
//
// The response channel is registered in session.Keys["requests"] *before*
// Write is called, and is buffered (size 1): registering after Write left a
// window where a client fast enough to answer before Store completed found
// no channel in handleMessage and silently lost its response, forcing a full
// 70s timeout. The deferred Delete here is the requesting side's half of the
// timeout-cleanup contract: on a timeout, this removes the entry so a very
// late response arriving afterwards finds nothing to send to (see
// handleMessage's own Delete-then-non-blocking-send for its half).
func (s *server) sendRequest(session *melody.Session, method string, args []interface{}) ([]byte, error) {
	requestID, err := generateRandomIDWithNumbers(16) // 62^16 possibilities = 4.767x10^28 (I think we don't need to check for collisions)
	if err != nil {
		return nil, fmt.Errorf("error generating request id: %w", err)
	}
	request := Request{Method: method, Args: args, RequestID: requestID}

	serialized, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error serializing request: %w", err)
	}

	responseChan := make(chan Response, 1)
	requests := session.Keys["requests"].(*xsync.MapOf[string, chan Response])
	requests.Store(request.RequestID, responseChan)
	defer requests.Delete(request.RequestID)

	if err := session.Write(serialized); err != nil {
		return nil, fmt.Errorf("error sending request to client: %w", err)
	}

	select {
	case response := <-responseChan:
		if response.Error != "" {
			return nil, fmt.Errorf("error from client: %s", response.Error)
		}
		byteVal, ok := response.Result.(string)
		if !ok {
			return nil, fmt.Errorf("error converting response to string")
		}
		return []byte(byteVal), nil
	case <-time.After(requestTimeout):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (s *server) listTables(id string) ([]byte, error) {
	session, err := s.retrieveSession(id)
	if err != nil {
		return nil, err
	}
	return s.sendRequest(session, "list-tables", []interface{}{})
}

func (s *server) describeTable(id, tableName string) ([]byte, error) {
	session, err := s.retrieveSession(id)
	if err != nil {
		return nil, err
	}
	return s.sendRequest(session, "describe-table", []interface{}{tableName})
}

func (s *server) executeQuery(id, query string) ([]byte, error) {
	session, err := s.retrieveSession(id)
	if err != nil {
		return nil, err
	}
	return s.sendRequest(session, "execute-query", []interface{}{query})
}
