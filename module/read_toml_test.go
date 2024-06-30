package module

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestTOMLModule(t *testing.T) {
	sql.Register("sqlite3-toml", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			conn.CreateModule("toml_extract", &TomlModule{})
			return nil
		},
	})

	db, err := sql.Open("sqlite3-toml", ":memory:")
	require.NoError(t, err, "opening connection must not fail")

	defer db.Close()
	dbx := sqlx.NewDb(db, "sqlite3-toml")

	_, err = db.Exec("create virtual table smallTablet using toml_extract(filepath=\"https://raw.githubusercontent.com/rust-lang/cc-rs/96c9e44ce3dfccc47c16fcfd743a33b3b4205daf/Cargo.toml\")")
	require.NoError(t, err, "creating virtual table must not fail")

	t.Run("Simple select count(*)", func(t *testing.T) {
		rowCount := 0
		err = dbx.Get(&rowCount, "select count(*) from smallTablet")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, 6, rowCount, "row count must be 197")
	})

	// Select all keys and print them
	t.Run("Get all keys", func(t *testing.T) {
		rows, err := dbx.Query("SELECT * FROM smallTablet")
		require.NoError(t, err, "querying virtual table must not fail")
		for rows.Next() {
			var key string
			var value string
			err = rows.Scan(&key, &value)
			require.NoError(t, err, "scanning row must not fail")
			t.Logf("TOML %s: %s", key, value)
		}
		rows.Close()
	})

}
