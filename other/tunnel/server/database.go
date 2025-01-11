package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type tunnel struct {
	// The tunnel ID
	ID string `json:"id"`
	// The hashed auth token using SHA256
	AuthToken string `json:"auth_token"`
	// When the tunnel was created
	CreatedAt time.Time `json:"created_at"`
	// When the tunnel will expire
	ExpiresAt time.Time `json:"expires_at"`
	// Metadata about the tunnel
	Metadata map[string]interface{} `json:"metadata"`
}

// Insert the tunnel into the database
func InsertTunnel(db *sql.DB, tunnel *tunnel) error {
	serializedMetadata, err := json.Marshal(tunnel.Metadata)
	if err != nil {
		return fmt.Errorf("error serializing metadata: %w", err)
	}

	_, err = db.Exec("INSERT INTO tunnels (id, hashedToken, createdAt, expiresAt, metadata) VALUES (?, ?, ?, ?, ?)",
		tunnel.ID, tunnel.AuthToken, tunnel.CreatedAt.Format(time.RFC3339), tunnel.ExpiresAt.Format(time.RFC3339), string(serializedMetadata))
	if err != nil {
		return err
	}

	return nil
}

// Get the tunnel from the database
func GetTunnel(db *sql.DB, id string) (*tunnel, error) {
	row := db.QueryRow("SELECT id, hashedToken, createdAt, expiresAt, metadata FROM tunnels WHERE id = ?", id)
	var CreatedAt, ExpiresAt string

	var tunnel tunnel
	var metadata string
	err := row.Scan(&tunnel.ID, &tunnel.AuthToken, &CreatedAt, &ExpiresAt, &metadata)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(metadata), &tunnel.Metadata)
	if err != nil {
		return nil, fmt.Errorf("error deserializing metadata: %w", err)
	}

	// Parse the dates
	tunnel.CreatedAt, err = time.Parse(time.RFC3339, CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("error parsing created at date: %w", err)
	}
	tunnel.ExpiresAt, err = time.Parse(time.RFC3339, ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("error parsing expires at date: %w", err)
	}

	return &tunnel, nil
}

func SetLastConnection(db *sql.DB, id string) error {
	_, err := db.Exec("UPDATE tunnels SET metadata = JSON_SET(metadata, '$.lastConnection', ?) WHERE id = ?", time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("error setting last connection: %w", err)
	}

	return nil
}
