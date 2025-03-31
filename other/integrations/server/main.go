package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
)

func main() {
	// Parse command line arguments
	adr := ":7654"
	verbose := false
	pflag.BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	pflag.StringVarP(&adr, "addr", "a", adr, "address (and port) to listen on")
	pflag.Parse()

	// Initialize the logger
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	var db *sql.DB

	// Initialize the router
	r := newRouter(routerArgs{
		db:     db,
		logger: logger,
	})

	// Listen for SIGINT and SIGTERM signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-c
		logger.Info("Received shutdown signal, shutting down server...")
		if err := db.Close(); err != nil {
			logger.Error("Error closing database connection", "error", err)
		}
	}()

	// Start the server
	if err := http.ListenAndServe(adr, r); err != nil {
		switch err {
		case http.ErrServerClosed:
			// Server closed
		default:
			// Handle other errors
			panic(err)
		}

	}

}
