package module

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestJsonModule(t *testing.T) {

	// Create a SQLite connection and register the JSON module
	sql.Register("sqlite3-json", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			conn.CreateModule("json_extract", &JSONModule{})
			return nil
		},
	})

	// Open a new connection
	db, err := sql.Open("sqlite3-json", ":memory:")
	require.NoError(t, err, "opening connection must not fail")
	defer db.Close()

	dbx := sqlx.NewDb(db, "sqlite3-json")

	_, err = db.Exec("create virtual table smallTable using json_extract(filepath=\"https://microsoftedge.github.io/Demos/json-dummy-data/64KB.json\")")
	require.NoError(t, err, "creating virtual table must not fail")

	// Query the virtual table
	t.Run("Simple select count(*) on array", func(t *testing.T) {
		rowCount := 0
		err = dbx.Get(&rowCount, "select count(*) from smallTable")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, 197, rowCount, "row count must be 197")
	})

	t.Run("Select on object", func(t *testing.T) {
		_, err = db.Exec("create virtual table objectTable using json_extract(filepath=\"https://formulae.brew.sh/api/cask/docker.json\")")
		require.NoError(t, err, "creating virtual table must not fail")

		var name string
		err = dbx.Get(&name, "select token from objectTable")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, "docker", name, "name must be docker")

		var tap string
		err = dbx.Get(&tap, "select tap from objectTable where token = 'docker'")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, "homebrew/cask", tap, "tap must be homebrew/cask")
	})

	t.Run("Select with JSONPath", func(t *testing.T) {
		_, err = db.Exec("create virtual table jsonPathTable using json_extract(filepath=\"https://microsoftedge.github.io/Demos/json-dummy-data/64KB.json\", jsonpath=\"$[0]\")")
		require.NoError(t, err, "creating virtual table must not fail")
		var name string
		err = dbx.Get(&name, "select name from jsonPathTable")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, "Adeel Solangi", name, "name must be Adeel Solangi")
	})

}
