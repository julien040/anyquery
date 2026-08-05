package client

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
)

const requestTunnelEndpoint = "https://tunnel.anyquery.xyz/tunnel/new"

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type requestTunnelAPIResponse struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
	ServerURL string `json:"server_url"`
}

type TunnelRequest struct {
	AuthToken string `json:"auth_token"`
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
	ServerURL string `json:"server_url"`
}

// randString returns a cryptographically random string of length n drawn from
// the full alphabet. It generates the tunnel's auth token (see RequestTunnel),
// which is the bearer secret authenticating the WebSocket upgrade, so it must
// use crypto/rand and must not silently exclude a character: the previous
// implementation indexed with rand.IntN(len(alphabet)-1), which never emitted
// the last letter of the alphabet ('9') and used math/rand.
func randString(n int) (string, error) {
	alphabetSize := big.NewInt(int64(len(alphabet)))
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", fmt.Errorf("error generating random string: %w", err)
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b), nil
}

func hashToken(token string) string {
	summed := sha256.Sum256([]byte(token))
	return hex.EncodeToString(summed[:])
}

// Request a new tunnel to the API
func RequestTunnel() (TunnelRequest, error) {
	// Generate a random password
	password, err := randString(128)
	if err != nil {
		return TunnelRequest{}, err
	}

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
	t.ServerURL = data.ServerURL

	return t, nil
}
