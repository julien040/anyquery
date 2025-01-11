package client

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
)

const requestTunnelEndpoint = "https://tunnel.anyquery.xyz/tunnel/new"

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type requestTunnelAPIResponse struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
}

type TunnelRequest struct {
	AuthToken string `json:"auth_token"`
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
}

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rand.IntN(len(alphabet)-1)]
	}
	return string(b)
}

// Request a new tunnel to the API
func RequestTunnel() (TunnelRequest, error) {
	// Generate a random password
	password := randString(128)

	t := TunnelRequest{
		AuthToken: password,
	}

	// Hash the password
	hashed := hashToken(password)

	// Send the request to the API
	req, err := http.NewRequest(http.MethodPost, requestTunnelEndpoint, nil)
	if err != nil {
		return t, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", hashed)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return t, fmt.Errorf("error sending request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return t, fmt.Errorf("error getting tunnel: %d", resp.StatusCode)
	}

	var data requestTunnelAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return t, fmt.Errorf("error decoding json response: %w", err)
	}

	t.ID = data.ID
	t.ExpiresAt = data.ExpiresAt

	return t, nil
}
