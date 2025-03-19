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
	s.logger.Info("New connection", "remoteAddr", se.Request.RemoteAddr)
	s.sessions.Store(se.Keys["tunnel_id"].(string), se)
}

func (s *server) handleDisconnectWS(se *melody.Session) {
	s.logger.Info("Connection closed", "remoteAddr", se.Request.RemoteAddr)
	s.sessions.Delete(se.Keys["tunnel_id"].(string))
}

// Response from the client
func (s *server) handleMessage(se *melody.Session, msg []byte) {
	// Deserialize the message
	var response Response
	err := json.Unmarshal(msg, &response)
	if err != nil {
		s.logger.Error("Error deserializing message", "error", err, "id", se.Keys["tunnel_id"])
		return
	}

	// Make sure the request ID is provided
	if response.RequestID == "" {
		s.logger.Error("Request ID not provided", "id", se.Keys["tunnel_id"])
		return
	}

	// Get the response channel
	responseChan, ok := se.Keys["requests"].(*xsync.MapOf[string, chan Response]).Load(response.RequestID)
	if !ok {
		s.logger.Error("Response channel not found", "id", se.Keys["tunnel_id"], "requestID", response.RequestID)
		return
	}

	// Send the response
	/* responseChan.(chan Response) <- response */
	responseChan <- response

	// Delete the response channel
	se.Keys["requests"].(*xsync.MapOf[string, chan Response]).Delete(response.RequestID)

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

func (s *server) listTables(id string) ([]byte, error) {
	// Retrieve the websocket session
	session, err := s.retrieveSession(id)
	if err != nil {
		return nil, err
	}

	// Send the request
	request := Request{
		Method:    "list-tables",
		Args:      []interface{}{},
		RequestID: generateRandomIDWithNumbers(16), // 62^16 possibilities = 4.767x10^28 (I think we don't need to check for collisions)
	}

	// Serialize the request
	serialized, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error serializing request: %w", err)
	}

	// Send the request
	err = session.Write(serialized)
	if err != nil {
		return nil, fmt.Errorf("error sending request to client: %w", err)
	}

	// Create a channel to wait for the response
	responseChan := make(chan Response)
	session.Keys["requests"].(*xsync.MapOf[string, chan Response]).Store(request.RequestID, responseChan)

	// Wait for the response
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

func (s *server) describeTable(id, tableName string) ([]byte, error) {
	// Retrieve the websocket session
	session, err := s.retrieveSession(id)
	if err != nil {
		return nil, err
	}

	// Send the request
	request := Request{
		Method:    "describe-table",
		Args:      []interface{}{tableName},
		RequestID: generateRandomIDWithNumbers(16),
	}

	// Serialize the request
	serialized, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error serializing request: %w", err)
	}

	// Send the request
	err = session.Write(serialized)
	if err != nil {
		return nil, fmt.Errorf("error sending request to client: %w", err)
	}

	// Create a channel to wait for the response
	responseChan := make(chan Response)
	session.Keys["requests"].(*xsync.MapOf[string, chan Response]).Store(request.RequestID, responseChan)

	// Wait for the response
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

func (s *server) executeQuery(id, query string) ([]byte, error) {
	// Retrieve the websocket session
	session, err := s.retrieveSession(id)
	if err != nil {
		return nil, err
	}

	// Send the request
	request := Request{
		Method:    "execute-query",
		Args:      []interface{}{query},
		RequestID: generateRandomIDWithNumbers(16),
	}

	// Serialize the request
	serialized, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error serializing request: %w", err)
	}

	// Send the request
	err = session.Write(serialized)
	if err != nil {
		return nil, fmt.Errorf("error sending request to client: %w", err)
	}

	// Create a channel to wait for the response
	responseChan := make(chan Response)
	session.Keys["requests"].(*xsync.MapOf[string, chan Response]).Store(request.RequestID, responseChan)

	// Wait for the response
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
