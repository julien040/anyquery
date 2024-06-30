package module

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestJSONLModule(t *testing.T) {

	sql.Register("sqlite3-jsonl", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			conn.CreateModule("jsonl_extract", &JSONlModule{})
			return nil
		},
	})

	db, err := sql.Open("sqlite3-jsonl", ":memory:")
	require.NoError(t, err, "opening connection must not fail")

	defer db.Close()
	dbx := sqlx.NewDb(db, "sqlite3-jsonl")

	_, err = db.Exec("create virtual table smallTablel using jsonl_extract(filepath=\"https://csvbase.com/meripaterson/stock-exchanges.jsonl\")")
	require.NoError(t, err, "creating virtual table must not fail")

	t.Run("Simple select count(*)", func(t *testing.T) {
		rowCount := 0
		err = dbx.Get(&rowCount, "select count(*) from smallTablel")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, 251, rowCount, "row count must be 197")
	})

	t.Run("Ensure no column name is empty or has spaces", func(t *testing.T) {
		rows, err := dbx.Query("SELECT * FROM smallTablel LIMIT 1")
		require.NoError(t, err, "querying virtual table must not fail")
		columns, err := rows.Columns()
		require.NoError(t, err, "getting columns must not fail")
		for _, col := range columns {
			require.NotEmpty(t, col, "column name must not be empty")
			require.NotContains(t, col, " ", "column name must not contain spaces")
		}

		err = rows.Close()
		require.NoError(t, err, "closing rows must not fail")
	})

	t.Run("Select an object", func(t *testing.T) {
		var country string
		err = dbx.Get(&country, "select Country from smallTablel where Name = 'Euronext Paris'")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, "France", country, "country must be France")
	})

}
