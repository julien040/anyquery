package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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

type Tunnel struct {
	// The tunnel ID to connect to
	ID string
	// The unhashed auth token
	AuthToken string
	// The websocket URL to connect to
	ServerURL string
	// When the tunnel will expire
	ExpiresAt string

	// Internal fields
	// The websocket connection
	conn *websocket.Conn
}

func (t *Tunnel) Connect() error {
	return t._connect()
}

func (t *Tunnel) Close() error {
	if t.conn == nil {
		return nil
	}

	// Send a close message
	if err := t.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "The CLI is disconnecting")); err != nil {
		return fmt.Errorf("error sending close message: %w", err)
	}

	// Close the connection
	if err := t.conn.Close(); err != nil {
		return fmt.Errorf("error closing connection: %w", err)
	}

	return nil
}

func (t *Tunnel) _connect() error {
	// Connect to the server
	maxAttempts := 5
	attempt := 0
	currentDelay := 1
	var lastError error
	for {
		if attempt >= maxAttempts {
			return fmt.Errorf("failed to connect to server after %d attempts: %w", maxAttempts, lastError)
		}

		if currentDelay > 1 {
			<-time.After(time.Duration(currentDelay) * time.Second)
		}

		headers := http.Header{}
		headers.Set("Authorization", "Bearer "+hashToken(t.AuthToken))

		//toDial := /* fmt.Sprintf("%s?tunnel_id=%s", t.ServerURL, url.QueryEscape(t.ID)) */ serverAddr

		conn, res, err := websocket.DefaultDialer.Dial(t.ServerURL, headers)
		if err != nil {
			lastError = err
			currentDelay = min(currentDelay*2, 10)
			attempt++
			if err == websocket.ErrBadHandshake {
				// Get the error from the response
				text, _ := io.ReadAll(res.Body)
				return fmt.Errorf("error connecting to server (cannot handshake): %s (%w)", string(text), err)
			}
			continue
		}

		t.conn = conn
		break
	}

	return nil
}

// Wait for a request from the server
//
// If the connection is not already established, this function will establish it.
// If the connection is closed, this function will attempt to reconnect 5 times with exponential backoff.
func (t *Tunnel) WaitRequest() (Request, error) {
	if t.conn == nil {
		if err := t._connect(); err != nil {
			return Request{}, fmt.Errorf("error connecting to server: %w", err)
		}
	}

	mType, messageReader, err := t.conn.NextReader()
	if err != nil {
		// Try to reconnect
		if err := t._connect(); err != nil {
			return Request{}, fmt.Errorf("error connecting to server: %w", err)
		}
		mType, messageReader, err = t.conn.NextReader()
		if err != nil {
			return Request{}, fmt.Errorf("error getting message reader: %w", err)
		}
	}

	if mType == websocket.PingMessage || mType == websocket.PongMessage {
		// Ignore ping/pong messages
		return t.WaitRequest()
	}

	var request Request
	if err := json.NewDecoder(messageReader).Decode(&request); err != nil {
		return Request{}, fmt.Errorf("error decoding message: %w", err)
	}

	return request, nil
}

func (t *Tunnel) SendResponse(response Response) error {
	if t.conn == nil {
		if err := t._connect(); err != nil {
			return fmt.Errorf("error connecting to server: %w", err)
		}
	}

	if err := t.conn.WriteJSON(response); err != nil {
		return fmt.Errorf("error sending response: %w", err)
	}

	return nil
}
