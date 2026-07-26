package main

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
)

func resJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

const alphabet = "abcdefghijklmnopqrstuvwxyz"
const alphabetNumbersUpper = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generateRandomID generates a cryptographically random ID of length n over
// the full alphabet (26^n possibilities). This is used for the tunnel ID,
// which acts as a bearer-equivalent capability token, so it must be both
// unbiased/unpredictable (crypto/rand, not math/rand) and drawn from the
// entire alphabet (no character silently excluded).
func generateRandomID(n int) (string, error) {
	alphabetSize := big.NewInt(int64(len(alphabet)))
	b := make([]byte, n)
	for i := range b {
		idx, err := crand.Int(crand.Reader, alphabetSize)
		if err != nil {
			return "", fmt.Errorf("error generating random id: %w", err)
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b), nil
}

// Generate a random ID with 62^n possibilities
func generateRandomIDWithNumbers(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabetNumbersUpper[rand.IntN(len(alphabetNumbersUpper)-1)]
	}
	return string(b)
}

// Matches /tunnel/new
func (r *server) newTunnel(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		resJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var authToken string
	if req.Header.Get("Authorization") != "" {
		authToken = req.Header.Get("Authorization")
	} else {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "missing Authorization header"})
		return
	}

	// Ensure the auth token is of max length 128
	if len(authToken) > 128 {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "auth token too long"})
		return
	}

	// 16 chars mirrors the request-ID length used for the WS request/response
	// correlation (websocket.go). The tunnel ID is the capability token for
	// the gpt-facing HTTP API, so it needs real keyspace.
	id, err := generateRandomID(16)
	if err != nil {
		r.logger.Error("Error generating tunnel id", "error", err)
		resJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	t := &tunnel{
		ID:        id,
		AuthToken: authToken,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 24 * 120), // 120 days
		Metadata:  map[string]interface{}{},
		ServerUrl: fmt.Sprintf("wss://eu-central-1-websocket.anyquery.xyz/websocket-anyquery?tunnel_id=%s", id),
	}

	if err := InsertTunnel(r.db, t); err != nil {
		r.logger.Error("Error inserting tunnel", "error", err)
		resJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resJSON(w, http.StatusOK, map[string]string{"id": id, "expires_at": t.ExpiresAt.Format(time.RFC3339), "server_url": t.ServerUrl, "created_at": t.CreatedAt.Format(time.RFC3339)})
}

func (s *server) listTablesAPI(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id parameter"})
		return
	}

	res, err := s.listTables(id)
	if err != nil {
		s.logger.Error("Error listing tables", "error", err)
		resJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(res)
}

func (s *server) describeTableAPI(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id parameter"})
		return
	}

	// Parse the request body
	var body struct {
		Table string `json:"table_name"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	res, err := s.describeTable(id, body.Table)
	if err != nil {
		s.logger.Error("Error describing table", "error", err)
		resJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(res)
}

func (s *server) executeQueryAPI(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id parameter"})
		return
	}

	// Parse the request body
	var body struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	res, err := s.executeQuery(id, body.Query)
	if err != nil {
		s.logger.Error("Error executing query", "error", err)
		resJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(res)
}

// Matches /tunnel/oauth2/token
// Use to make fake oauth2 callback (take the code, or refresh token, and return it as the access token)
// Supports both code and refresh_token grant types
func (s *server) tunnelOauth2Token(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		resJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	type callbackRequest struct {
		GrantType string `json:"grant_type"`
		Code      string `json:"code"`
		Refresh   string `json:"refresh_token"`
	}

	var cb callbackRequest

	body, err := io.ReadAll(io.LimitReader(req.Body, 16384))
	if err != nil {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	// Extract the request body from url encoded form
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	cb.Code = parsed.Get("code")
	cb.Refresh = parsed.Get("refresh_token")

	if cb.Code == "" && cb.Refresh == "" {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "missing code or refresh_token"})
		return
	}

	if cb.Code != "" && cb.Refresh != "" {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "both code and refresh_token are provided"})
		return
	}

	retrieveExpiresIn := func(code string) (int, error) {
		row := s.db.QueryRow("SELECT expiresAt FROM tunnels WHERE id = ?", code)
		var expiresAt string
		if err := row.Scan(&expiresAt); err != nil {
			return 0, err
		}

		expiredAtTime, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return 0, fmt.Errorf("parsing expires at time with error: %w", err)
		}

		return int(time.Until(expiredAtTime).Seconds()), nil
	}

	var code string
	if cb.Code != "" {
		code = cb.Code
	} else {
		code = cb.Refresh
	}

	expiresIn, err := retrieveExpiresIn(code)
	if err != nil {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid code or refresh_token. Make sure the Anyquery ID passed by the user is correct"})
		return
	}

	if expiresIn < 0 {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "The code has expired. Restart Anyquery to get a new tunnel ID, and modify your LLM client to use the new tunnel ID"})
		return
	}

	resJSON(w, http.StatusOK, map[string]interface{}{"access_token": code, "token_type": "bearer", "expires_in": expiresIn, "refresh_token": code})

}
