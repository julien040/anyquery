package module

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestYAMLModule(t *testing.T) {
	sql.Register("sqlite3-yaml", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			conn.CreateModule("yaml_extract", &YamlModule{})
			return nil
		},
	})

	db, err := sql.Open("sqlite3-yaml", ":memory:")
	require.NoError(t, err, "opening connection must not fail")

	defer db.Close()
	dbx := sqlx.NewDb(db, "sqlite3-yaml")

	_, err = db.Exec("create virtual table smallTabley using yaml_extract(filepath=\"https://raw.githubusercontent.com/yaml/yaml-test-suite/main/src/7T8X.yaml\")")
	require.NoError(t, err, "creating virtual table must not fail")

	t.Run("Simple select count(*)", func(t *testing.T) {
		rowCount := 0
		err = dbx.Get(&rowCount, "select count(*) from smallTabley")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, 7, rowCount, "row count must be 197")
	})

	t.Run("Get a key", func(t *testing.T) {
		var value string
		err = dbx.Get(&value, "select value from smallTabley where key = '[0].name'")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, "Spec Example 8.10. Folded Lines - 8.13. Final Empty Lines", value, "value must be value")
	})

}
