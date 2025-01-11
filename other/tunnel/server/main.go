package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	_ "embed"

	_ "modernc.org/sqlite"
)

var port = 8056
var host = "127.0.0.1"
var databaseURI = "httpConnection.db"
var proxyDomain = "reverse.anyquery.xyz"

//go:embed schema.sql
var dbSchema string

func main() {

	mux := http.NewServeMux()

	// Try to get the host and port from the ENV
	if h := os.Getenv("HOST"); h != "" {
		host = h
	}
	if p := os.Getenv("PORT"); p != "" {
		// Parse the port
		var err error
		port, err = strconv.Atoi(p)
		if err != nil {
			panic("Invalid PORT env variable")
		}
	}

	if dbURI := os.Getenv("DATABASE_URI"); dbURI != "" {
		databaseURI = dbURI
	}

	if pd := os.Getenv("PROXY_DOMAIN"); pd != "" {
		proxyDomain = pd
	}

	logger := slog.New(slog.Default().Handler())

	// Open the database
	db, err := sql.Open("sqlite", databaseURI)
	if err != nil {
		logger.Error("Error opening database", "error", err)
		return
	}
	defer db.Close()

	// Create the table if it doesn't exist
	_, err = db.Exec(dbSchema)
	if err != nil {
		logger.Error("Error creating table", "error", err)
		return
	}

	// Register the routes
	r := &routes{logger: logger, db: db}
	mux.HandleFunc("/frp-handler", r.frpHandler)
	mux.HandleFunc("/tunnel/new", r.newTunnel)

	// Start the server
	logger.Info("Starting server", "host", host, "port", port)
	err = http.ListenAndServe(host+":"+strconv.Itoa(port), mux)
	if err != nil {
		slog.Error("Error starting server", "error", err)
	}

}
