package module

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
	"github.com/stretchr/testify/require"
)

func openHTMLDB(t *testing.T, r *Restrictions) (*sql.DB, *sqlx.DB) {
	t.Helper()
	name := fmt.Sprintf("sqlite3-html-%p", t)
	sql.Register(name, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			conn.CreateModule("html_reader", &HtmlModule{Restrictions: r})
			return nil
		},
	})
	db, err := sql.Open(name, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, sqlx.NewDb(db, name)
}

const sampleHTMLTable = `<html><body><table>
<thead><tr><th>id</th><th>name</th></tr></thead>
<tbody>
<tr><td>1</td><td>Alice</td></tr>
<tr><td>2</td><td>Bob</td></tr>
</tbody>
</table></body></html>`

func TestHtmlModuleLocalFile(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "table.html", sampleHTMLTable)

	db, dbx := openHTMLDB(t, &Restrictions{AllowedDirs: []string{dir}})
	_, err := db.Exec(fmt.Sprintf("create virtual table t using html_reader(file=%q, selector='table')", p))
	require.NoError(t, err)

	var count int
	require.NoError(t, dbx.Get(&count, "select count(*) from t"))
	require.Equal(t, 2, count)

	var name string
	require.NoError(t, dbx.Get(&name, "select name from t where id = '2'"))
	require.Equal(t, "Bob", name)
}

func TestHtmlModuleDeniesOutsideAllowedDirs(t *testing.T) {
	outside := t.TempDir()
	p := writeTempFile(t, outside, "secret.html", sampleHTMLTable)

	db, _ := openHTMLDB(t, &Restrictions{AllowedDirs: []string{t.TempDir()}})
	_, err := db.Exec(fmt.Sprintf("create virtual table t using html_reader(file=%q)", p))
	require.Error(t, err)
}

func TestHtmlModuleStdinDeniedUnderSandbox(t *testing.T) {
	db, _ := openHTMLDB(t, &Restrictions{})
	_, err := db.Exec("create virtual table t using html_reader(file='stdin')")
	require.Error(t, err, "reading stdin under a sandbox must be denied")
}

// TestHtmlModuleCacheTTLHonouredForRemote: cache=<ttl> is a no-op for local
// sources, but must still be honoured for a remote one.
func TestHtmlModuleCacheTTLHonouredForRemote(t *testing.T) {
	var hits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/t.html", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write([]byte(sampleHTMLTable))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	db, dbx := openHTMLDB(t, nil)
	url := srv.URL + "/t.html"
	_, err := db.Exec(fmt.Sprintf("create virtual table t1 using html_reader(file=%q, selector='table', cache=3600)", url))
	require.NoError(t, err)
	var count int
	require.NoError(t, dbx.Get(&count, "select count(*) from t1"))
	require.Equal(t, 2, count)

	_, err = db.Exec(fmt.Sprintf("create virtual table t2 using html_reader(file=%q, selector='table', cache=3600)", url))
	require.NoError(t, err)
	require.NoError(t, dbx.Get(&count, "select count(*) from t2"))
	require.Equal(t, 2, count)

	require.Equal(t, int32(1), atomic.LoadInt32(&hits), "a fresh cache entry must not be re-fetched")
}
