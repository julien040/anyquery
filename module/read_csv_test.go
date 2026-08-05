package module

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
	"github.com/stretchr/testify/require"
)

func TestCSVModule(t *testing.T) {
	// Create a SQLite connection and register the JSON module
	sql.Register("sqlite3-csv", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			conn.CreateModule("csv_extract", &CsvModule{})
			return nil
		},
	})

	// Open a new connection
	db, err := sql.Open("sqlite3-csv", ":memory:")
	require.NoError(t, err, "opening connection must not fail")

	dbx := sqlx.NewDb(db, "sqlite3-csv")
	defer db.Close()

	_, err = db.Exec("create virtual table smallTable using csv_extract(filepath=\"https://csvbase.com/meripaterson/stock-exchanges\", header=true)")
	require.NoError(t, err, "creating virtual table must not fail")

	// Query the virtual table
	t.Run("Simple select count(*)", func(t *testing.T) {
		rowCount := 0
		err = dbx.Get(&rowCount, "select count(*) from smallTable")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, 251, rowCount, "row count must be 197")
	})

	t.Run("Ensure no column name is empty or has spaces", func(t *testing.T) {
		rows, err := dbx.Query("SELECT * FROM smallTable LIMIT 1")
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
		err = dbx.Get(&country, "select Country from smallTable where Name = 'Euronext Paris'")
		require.NoError(t, err, "querying virtual table must not fail")
		require.Equal(t, "France", country, "country must be France")
	})

}

// TestCsvAutoDetectionSQL reads the same file twice: once with no argument at
// all, where the delimiter, the header and the column types are inferred, and
// once with header=false, which must turn the header part of that inference off
// and hand the first row over as data.
func TestCsvAutoDetectionSQL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.csv")
	require.NoError(t, os.WriteFile(path, []byte("name;age\nalice;30\nbob;25\n"), 0o600),
		"writing the fixture must not fail")

	name := "sqlite3-csv-autodetect"
	sql.Register(name, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			// A nil policy means unrestricted, which the temp dir read needs.
			return conn.CreateModule("csv_reader", &CsvModule{Restrictions: nil})
		},
	})
	db, err := sql.Open(name, ":memory:")
	require.NoError(t, err, "opening connection must not fail")
	db.SetMaxOpenConns(1)
	defer db.Close()
	dbx := sqlx.NewDb(db, name)

	t.Run("no argument", func(t *testing.T) {
		_, err := db.Exec(fmt.Sprintf("create virtual table auto using csv_reader('%s')", path))
		require.NoError(t, err, "creating the virtual table must not fail")

		rows, err := dbx.Query("select * from auto limit 1")
		require.NoError(t, err, "querying must not fail")
		columns, err := rows.Columns()
		require.NoError(t, err, "getting columns must not fail")
		require.NoError(t, rows.Close())
		require.Equal(t, []string{"name", "age"}, columns, "columns must be named after the header row")

		var rowCount int
		require.NoError(t, dbx.Get(&rowCount, "select count(*) from auto"))
		require.Equal(t, 2, rowCount, "the header row must not be counted as data")

		var declaredType string
		require.NoError(t, dbx.Get(&declaredType, "select typeof(age) from auto where name = 'alice'"))
		require.Equal(t, "integer", declaredType, "the age column must be typed")

		var age int
		require.NoError(t, dbx.Get(&age, "select age from auto where name = 'bob'"))
		require.Equal(t, 25, age, "value of the age column")
	})

	t.Run("explicit header=false", func(t *testing.T) {
		_, err := db.Exec(fmt.Sprintf("create virtual table raw using csv_reader('%s', header=false)", path))
		require.NoError(t, err, "creating the virtual table must not fail")

		rows, err := dbx.Query("select * from raw limit 1")
		require.NoError(t, err, "querying must not fail")
		columns, err := rows.Columns()
		require.NoError(t, err, "getting columns must not fail")
		require.NoError(t, rows.Close())
		require.Equal(t, []string{"col0", "col1"}, columns, "columns must fall back to positional names")

		var rowCount int
		require.NoError(t, dbx.Get(&rowCount, "select count(*) from raw"))
		require.Equal(t, 3, rowCount, "the header row must be returned as data")

		// The header row is part of the column values, so the column holds text.
		var declaredType string
		require.NoError(t, dbx.Get(&declaredType, "select typeof(col1) from raw where col0 = 'alice'"))
		require.Equal(t, "text", declaredType, "the column must not be typed as a number")

		var first string
		require.NoError(t, dbx.Get(&first, "select col1 from raw limit 1"))
		require.Equal(t, "age", first, "the first row must be the header line")
	})
}

const sampleCSVForCache = "id,name\n1,Alice\n2,Bob\n"

// openCsvCacheDB returns a connection where csv_reader is registered without
// any restriction. A single connection is enforced because each pooled
// connection to ":memory:" is a distinct empty database, and the tables below
// must stay visible across statements.
func openCsvCacheDB(t *testing.T) (*sql.DB, *sqlx.DB) {
	t.Helper()
	name := fmt.Sprintf("sqlite3-csv-cache-%p", t)
	sql.Register(name, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			conn.CreateModule("csv_reader", &CsvModule{})
			return nil
		},
	})
	db, err := sql.Open(name, ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db, sqlx.NewDb(db, name)
}

// csvCacheServer serves sampleCSVForCache and counts how many times it is
// actually downloaded. The URL embeds a per-run port, so the cache entry it
// maps to is never shared with another test or another run.
func csvCacheServer(t *testing.T) (url string, hits *int32) {
	t.Helper()
	hits = new(int32)
	mux := http.NewServeMux()
	mux.HandleFunc("/data.csv", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		w.Write([]byte(sampleCSVForCache))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL + "/data.csv", hits
}

// TestCsvCacheTTLRemote: the download happens in Connect, so every
// CREATE VIRTUAL TABLE either re-fetches or reuses the cache entry depending on
// the TTL it was given.
func TestCsvCacheTTLRemote(t *testing.T) {
	url, hits := csvCacheServer(t)
	db, dbx := openCsvCacheDB(t)

	_, err := db.Exec(fmt.Sprintf("create virtual table t1 using csv_reader(file=%q, header=true, cache='1')", url))
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(hits), "the first connect must download the file")

	var count int
	require.NoError(t, dbx.Get(&count, "select count(*) from t1"))
	require.Equal(t, 2, count)

	// Past the one-second TTL, the entry is stale and must be downloaded again.
	time.Sleep(1100 * time.Millisecond)
	_, err = db.Exec(fmt.Sprintf("create virtual table t2 using csv_reader(file=%q, header=true, cache='1')", url))
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(hits), "a stale cache entry must be re-fetched")
	require.NoError(t, dbx.Get(&count, "select count(*) from t2"))
	require.Equal(t, 2, count)

	// The entry written just above is well within an hour, so it is reused.
	_, err = db.Exec(fmt.Sprintf("create virtual table t3 using csv_reader(file=%q, header=true, cache='3600')", url))
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(hits), "a fresh cache entry must not be re-fetched")
	require.NoError(t, dbx.Get(&count, "select count(*) from t3"))
	require.Equal(t, 2, count)
}

func TestCsvCacheTTLInvalid(t *testing.T) {
	url, hits := csvCacheServer(t)
	db, _ := openCsvCacheDB(t)

	_, err := db.Exec(fmt.Sprintf("create virtual table t using csv_reader(file=%q, header=true, cache='abc')", url))
	require.ErrorContains(t, err, "failed to parse the cache TTL")
	require.Equal(t, int32(0), atomic.LoadInt32(hits), "an unparseable TTL must be rejected before any download")
}
