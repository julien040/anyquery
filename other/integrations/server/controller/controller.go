package controller

import (
	"database/sql"
	"log/slog"
)

type Controller struct {
	// Add any necessary fields here
	Db     *sql.DB
	Logger *slog.Logger
}

func NewController() *Controller {
	return &Controller{}
}
