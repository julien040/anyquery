// This file defines code to manage the internal database
package config

import (
	"database/sql"

	_ "embed"

	"github.com/adrg/xdg"
	_ "github.com/mattn/go-sqlite3"

	"github.com/julien040/anyquery/controller/config/model"
)

//go:embed schema.sql
var schema string

// OpenDatabaseConnection opens a connection to the database
//
// If the path is empty, it defaults to XDG_CONFIG_HOME/anyquery/config.db
// However, if the path is not empty, it will use the provided path (must be created beforehand)
func OpenDatabaseConnection(path string) (*sql.DB, *model.Queries, error) {
	// We get the path to open or default to the XDG Base Directory Specification
	var err error
	if path == "" {
		path, err = xdg.ConfigFile("anyquery/config.db")
		if err != nil {
			return nil, nil, err
		}
	}

	// We open the database
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, nil, err
	}

	// We create the schema
	_, err = db.Exec(schema)
	if err != nil {
		return nil, nil, err
	}

	// We create the querier
	querier := model.New(db)

	return db, querier, nil
}
