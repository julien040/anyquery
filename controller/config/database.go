// This file defines code to manage the internal database
package config

import (
	"database/sql"
	"fmt"
	"os"

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
func OpenDatabaseConnection(path string, readOnly bool) (*sql.DB, *model.Queries, error) {
	// We get the path to open or default to the XDG Base Directory Specification
	var err error
	if path == "" {
		path, err = xdg.ConfigFile("anyquery/config.db")
		if err != nil {
			return nil, nil, err
		}
	}

	// We check if the file exists
	// If it doesn't, we set readOnly to false to create the file
	// If we weren't setting the flag to false, because we are trying to read a non-existing file,
	// it would fail
	_, err = os.Stat(path)
	if os.IsNotExist(err) {

		readOnly = false
	}

	// We disable foreign keys because a registry can be deleted
	// while its plugins are still in the database
	//
	// We add ?_loc=auto to let SQLite use the local timezone
	sqlitePath := "file:" + path + "?cache=shared&_cache_size=-50000&_foreign_keys=OFF&_loc=auto"
	if readOnly {
		sqlitePath += "&mode=ro"
	}

	// We open the database
	db, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return nil, nil, err
	}

	// We create the schema
	_, err = db.Exec(schema)
	if err != nil {
		return nil, nil, err
	}

	// Apply the migrations in migrations.go
	err = applyMigrations(db)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	// We create the querier
	querier := model.New(db)

	return db, querier, nil
}
