package main

import (
	"database/sql"
	"embed"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed public/*
var staticFiles embed.FS

type routerArgs struct {
	db     *sql.DB
	logger *slog.Logger
}

func newRouter(_ routerArgs) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Handle("/", http.FileServer(http.FS(staticFiles)))

	// Define your routes here
	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {})

	r.Route("/dashboard", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Dashboard"))
		})
		r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Settings"))
		})
	})

	return r
}
