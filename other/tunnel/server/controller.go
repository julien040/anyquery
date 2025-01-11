package main

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"
)

type routes struct {
	logger *slog.Logger
	db     *sql.DB
}

func resJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func generateRandomID(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rand.IntN(len(alphabet)-1)] // I don't like Z
	}
	return string(b)
}

// Match /frp-handler
//
// It receives event from frp and accept/reject the request
func (r *routes) frpHandler(w http.ResponseWriter, req *http.Request) {
	op := req.URL.Query().Get("op")
	if op == "" {
		resJSON(w, http.StatusBadRequest, map[string]string{"error": "missing op"})
		return
	}

	type newProxyContent struct {
		Content struct {
			ProxyName          string            `json:"proxy_name"`
			ProxyType          string            `json:"proxy_type"`
			BandwidthLimit     string            `json:"bandwidth_limit"`
			BandwidthLimitMode string            `json:"bandwidth_limit_mode"`
			CustomDomains      []string          `json:"custom_domains"`
			Subdomain          string            `json:"subdomain"`
			Metas              map[string]string `json:"metas"`
		} `json:"content"`
	}

	type requestResponse struct {
		Reject       bool   `json:"reject"`
		RejectReason string `json:"reject_reason,omitempty"`
		Unchange     bool   `json:"unchange"`
		Content      *struct {
			BandwidthLimit     string `json:"bandwidth_limit"`
			BandwidthLimitMode string `json:"bandwidth_limit_mode"`
		}
	}

	if op != "NewProxy" {
		resJSON(w, http.StatusBadRequest, requestResponse{Reject: false, Unchange: true})
		return
	}

	var content newProxyContent
	if err := json.NewDecoder(req.Body).Decode(&content); err != nil {
		resJSON(w, http.StatusBadRequest, requestResponse{Reject: true, RejectReason: "Sidecar: invalid json"})
		return
	}

	id := content.Content.Subdomain

	r.logger.Info("NewProxy request", "id", id)

	if id == "" {
		r.logger.Info("Missing subdomain", "id", id)
		resJSON(w, http.StatusBadRequest, requestResponse{Reject: true, RejectReason: "Sidecar: missing subdomain"})
		return
	}

	if id != content.Content.ProxyName {
		r.logger.Info("Subdomain and proxy name must be the same", "id", id)
		resJSON(w, http.StatusBadRequest, requestResponse{Reject: true, RejectReason: "Sidecar: subdomain and proxy name must be the same"})
		return
	}

	// The subdomain is the ID of the tunnel
	// We need to check if the tunnel exists
	t, err := GetTunnel(r.db, id)
	if err == sql.ErrNoRows {
		r.logger.Info("Tunnel not found", "id", id)
		resJSON(w, http.StatusOK, requestResponse{Reject: true, RejectReason: "Sidecar: tunnel not found"})
		return
	}

	if err != nil {
		r.logger.Error("Error getting tunnel", "error", err)
		resJSON(w, http.StatusInternalServerError, requestResponse{Reject: true, RejectReason: "Sidecar: internal error"})
		return
	}

	// Check if the tunnel is expired
	if t.ExpiresAt.Before(time.Now()) {
		r.logger.Info("Tunnel expired", "id", id)
		resJSON(w, http.StatusOK, requestResponse{Reject: true, RejectReason: "Sidecar: tunnel expired"})
		return
	}

	// Check if the auth token is correct
	metaAuthToken, ok := content.Content.Metas["auth_token"]
	if !ok || metaAuthToken != t.AuthToken {
		r.logger.Info("Invalid auth token", "id", id)
		resJSON(w, http.StatusOK, requestResponse{Reject: true, RejectReason: "Sidecar: invalid auth token"})
		return
	}

	// Check if the tunnel is of http type
	if content.Content.ProxyType != "http" {
		r.logger.Info("Invalid proxy type", "id", id)
		resJSON(w, http.StatusOK, requestResponse{Reject: true, RejectReason: "Sidecar: invalid proxy type"})
		return
	}

	// Check if custom domain is empty
	if len(content.Content.CustomDomains) > 0 {
		resJSON(w, http.StatusOK, requestResponse{Reject: true, RejectReason: "Sidecar: custom domains not allowed"})
		return
	}

	// Accept the request
	resJSON(w, http.StatusOK, requestResponse{Reject: false, Unchange: true})

	// Update the last connection time
	if err := SetLastConnection(r.db, id); err != nil {
		r.logger.Error("Error setting last connection", "error", err)
	}

	r.logger.Info("Accepted tunnel", "id", id)

}

// Matches /tunnel/new
func (r *routes) newTunnel(w http.ResponseWriter, req *http.Request) {
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

	id := generateRandomID(8)

	t := &tunnel{
		ID:        id,
		AuthToken: authToken,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 24 * 120), // 120 days
		Metadata:  map[string]interface{}{},
	}

	if err := InsertTunnel(r.db, t); err != nil {
		r.logger.Error("Error inserting tunnel", "error", err)
		resJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resJSON(w, http.StatusOK, map[string]string{"id": id, "expires_at": t.ExpiresAt.Format(time.RFC3339)})
}
