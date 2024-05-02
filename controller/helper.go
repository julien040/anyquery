package controller

import (
	"database/sql"
	"path/filepath"

	"github.com/julien040/anyquery/controller/config"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/spf13/pflag"
)

// This file defines a few functions to help commands to run

// requestDatabase requests a database connection from the config package
//
// Its main purpose is to avoid code duplication in the commands.
// It parses the flags to see if the user wants to use a custom database (or the default one)
func requestDatabase(flags *pflag.FlagSet) (*sql.DB, *model.Queries, error) {
	// We get the path to the database
	path, err := flags.GetString("config")
	if err != nil {
		path = ""
	}

	// We get an absolute path if we have one
	if path != "" {
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, nil, err
		}
	}

	// We open the database
	return config.OpenDatabaseConnection(path)

}
