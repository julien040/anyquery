package main

import (
	"log/slog"
	"os"

	flag "github.com/spf13/pflag"
	_ "modernc.org/sqlite"
)

func main() {
	var dbWriteable bool
	var dbPath string
	var addr string
	var help bool
	var debug bool
	flag.BoolVarP(&dbWriteable, "writeable", "w", false, "if set to true, the server will enable an endpoint to create new connection IDs")
	flag.StringVarP(&addr, "addr", "a", "127.0.0.1:5566", "the address to listen on")
	flag.StringVarP(&dbPath, "db", "d", "httpConnection.db", "the path to the database file")
	flag.BoolVarP(&help, "help", "h", false, "show this help message")
	flag.BoolVarP(&debug, "debug", "v", false, "enable debug logging")
	flag.Parse()

	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	if help {
		flag.Usage()
		return
	}

	s := newServer(dbWriteable, addr, dbPath, logger)
	err := s.start()
	if err != nil {
		logger.Error("Error starting server", "error", err)
	}

}
