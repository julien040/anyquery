package namespace

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/module"
)

// sandboxConn builds an in-memory namespace with the given policy and returns a
// single dedicated connection (so multi-statement tests are deterministic — the
// in-memory database and the per-connection authorizer live on one connection).
func sandboxConn(t *testing.T, r *module.Restrictions) *sql.Conn {
	t.Helper()
	ns, err := NewNamespace(NamespaceConfig{
		InMemory:     true,
		Logger:       hclog.NewNullLogger(),
		Restrictions: r,
	})
	if err != nil {
		t.Fatalf("NewNamespace: %v", err)
	}
	db, err := ns.Register("")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("Conn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// TestSandboxLocalFileRead covers the LFR vulnerability: a sandboxed reader must
// refuse files outside the allowed directories but serve files inside them.
func TestSandboxLocalFileRead(t *testing.T) {
	ctx := context.Background()
	allowed := t.TempDir()
	csvPath := filepath.Join(allowed, "data.csv")
	if err := os.WriteFile(csvPath, []byte("name,age\nalice,30\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	conn := sandboxConn(t, &module.Restrictions{AllowedDirs: []string{allowed}})

	// Outside the allowed dir => denied (the PoC payload).
	_, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE passwd USING csv_reader('/etc/passwd')")
	if err == nil {
		t.Fatal("expected csv_reader('/etc/passwd') to be denied under sandbox")
	}
	if !strings.Contains(err.Error(), "sandbox") {
		t.Errorf("expected a sandbox error, got: %v", err)
	}

	// Inside the allowed dir => permitted.
	if _, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE ok USING csv_reader('"+csvPath+"', header=true)"); err != nil {
		t.Fatalf("csv_reader on an allowed path should work, got: %v", err)
	}
	var n int
	if err := conn.QueryRowContext(ctx, "SELECT count(*) FROM ok").Scan(&n); err != nil {
		t.Fatalf("select from allowed csv: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row from allowed csv, got %d", n)
	}
}

// TestSandboxSSRF covers the SSRF vulnerability: remote fetches are refused when
// remote access is disabled (no network is touched — the check is before fetch).
func TestSandboxSSRF(t *testing.T) {
	ctx := context.Background()
	conn := sandboxConn(t, &module.Restrictions{AllowedDirs: []string{t.TempDir()}})

	_, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE meta USING csv_reader('http://169.254.169.254/latest/meta-data/')")
	if err == nil {
		t.Fatal("expected remote fetch to be denied under sandbox")
	}
	if !strings.Contains(err.Error(), "sandbox") {
		t.Errorf("expected a sandbox error, got: %v", err)
	}
}

// TestSandboxArbitraryFileWrite covers the AFW/RCE vulnerability via both native
// write primitives: ATTACH DATABASE and VACUUM INTO. Both must be denied and
// must not create a file. In-memory ATTACH must still work.
func TestSandboxArbitraryFileWrite(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	conn := sandboxConn(t, &module.Restrictions{}) // fully locked down

	attackPath := filepath.Join(tmp, "pwn.db")
	if _, err := conn.ExecContext(ctx, "ATTACH DATABASE '"+attackPath+"' AS pwn"); err == nil {
		t.Error("expected ATTACH to an arbitrary path to be denied")
	}
	if _, statErr := os.Stat(attackPath); statErr == nil {
		t.Errorf("ATTACH created a file despite being denied: %s", attackPath)
	}

	vacuumPath := filepath.Join(tmp, "vac.db")
	if _, err := conn.ExecContext(ctx, "VACUUM main INTO '"+vacuumPath+"'"); err == nil {
		t.Error("expected VACUUM INTO an arbitrary path to be denied")
	}
	if _, statErr := os.Stat(vacuumPath); statErr == nil {
		t.Errorf("VACUUM INTO created a file despite being denied: %s", vacuumPath)
	}

	// In-memory ATTACH is legitimate and must remain allowed.
	if _, err := conn.ExecContext(ctx, "ATTACH DATABASE ':memory:' AS scratch"); err != nil {
		t.Errorf("in-memory ATTACH should be allowed under sandbox, got: %v", err)
	}
}

// TestSandboxDBReadersDisabled covers the connection-string SSRF / DuckDB-RCE
// vector: the database reader modules are not registered under a sandbox unless
// explicitly allowed.
func TestSandboxDBReadersDisabled(t *testing.T) {
	ctx := context.Background()
	conn := sandboxConn(t, &module.Restrictions{})

	_, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE d USING duckdb_reader(':memory:', 'x')")
	if err == nil {
		t.Fatal("expected duckdb_reader to be unavailable under sandbox")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "no such module") {
		t.Errorf("expected 'no such module', got: %v", err)
	}
}

// TestSandboxDBReadersAllowed confirms the opt-in re-registers the DB readers
// (the module loads; the connection itself is expected to fail, which is a
// different error than "no such module").
func TestSandboxDBReadersAllowed(t *testing.T) {
	ctx := context.Background()
	conn := sandboxConn(t, &module.Restrictions{AllowDBConnections: true})

	_, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE d USING duckdb_reader(':memory:', 'nonexistent')")
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "no such module") {
		t.Errorf("duckdb_reader should be registered when allowed, got: %v", err)
	}
}

// duckDBFile creates a DuckDB database in a fresh directory by running setup
// with the DuckDB CLI, skipping the test when the CLI is not installed.
func duckDBFile(t *testing.T, setup string) (dir, path string) {
	t.Helper()
	if _, err := exec.LookPath("duckdb"); err != nil {
		t.Skip("duckdb CLI not installed")
	}
	dir = t.TempDir()
	path = filepath.Join(dir, "x.duckdb")
	if out, err := exec.Command("duckdb", path, "-c", setup).CombinedOutput(); err != nil {
		t.Fatalf("creating duckdb file: %v: %s", err, out)
	}
	return dir, path
}

// TestDuckDBReaderTableNameInjection: the table argument is interpolated into
// a query run by the DuckDB CLI, so a quote in it must not end the string
// literal. Unescaped, the injected SELECT's rows become the table's columns.
func TestDuckDBReaderTableNameInjection(t *testing.T) {
	dir, dbPath := duckDBFile(t, "CREATE TABLE t(a INT)")
	ctx := context.Background()
	conn := sandboxConn(t, &module.Restrictions{AllowDBConnections: true, AllowedDirs: []string{dir}})

	_, err := conn.ExecContext(ctx, `CREATE VIRTUAL TABLE leak USING duckdb_reader('`+dbPath+`', "x');SELECT 'injected' AS column_name,'varchar' AS data_type,1 AS ordinal_position WHERE 'x'=('x")`)
	if err == nil {
		t.Fatal("injected SELECT ran: the virtual table was created from its rows")
	}
	if strings.Contains(err.Error(), "sandbox: ") {
		t.Fatalf("expected the path to be allowed so the injection itself is tested, got: %v", err)
	}
}

// TestDuckDBReaderSandboxConfinement: under a sandbox, duckdb_reader only
// opens local databases inside the allowed directories, and a view stored in
// an allowed database cannot read files outside them.
func TestDuckDBReaderSandboxConfinement(t *testing.T) {
	dir, dbPath := duckDBFile(t, "CREATE TABLE t(a INT); INSERT INTO t VALUES (42); CREATE VIEW leak AS SELECT content AS a FROM read_text('/etc/hosts');")
	ctx := context.Background()
	conn := sandboxConn(t, &module.Restrictions{AllowDBConnections: true, AllowedDirs: []string{dir}})

	for _, dsn := range []string{
		"http://127.0.0.1:1/evil.duckdb",
		"s3://bucket/evil.duckdb",
		"/etc/hosts",
		":memory:",
	} {
		_, err := conn.ExecContext(ctx, `CREATE VIRTUAL TABLE bad USING duckdb_reader('`+dsn+`', 't')`)
		if err == nil || !strings.Contains(err.Error(), "sandbox: ") {
			t.Errorf("duckdb_reader(%q) should be refused by the sandbox, got: %v", dsn, err)
		}
	}

	// A legitimate table in an allowed database still works.
	if _, err := conn.ExecContext(ctx, `CREATE VIRTUAL TABLE ok USING duckdb_reader('`+dbPath+`', 't')`); err != nil {
		t.Fatalf("allowed duckdb file should open: %v", err)
	}
	var a int
	if err := conn.QueryRowContext(ctx, "SELECT a FROM ok").Scan(&a); err != nil || a != 42 {
		t.Fatalf("reading allowed table: a=%d err=%v", a, err)
	}

	// The view's read_text must be blocked by DuckDB's external access lock.
	if _, err := conn.ExecContext(ctx, `CREATE VIRTUAL TABLE v USING duckdb_reader('`+dbPath+`', 'leak')`); err != nil {
		t.Fatalf("declaring the view table: %v", err)
	}
	var content string
	if err := conn.QueryRowContext(ctx, "SELECT a FROM v").Scan(&content); err == nil {
		t.Fatalf("view read a file outside the allowed dirs: %q", content)
	}
}

// TestNoSandboxUnrestricted confirms a nil policy leaves behavior unchanged:
// ATTACH to an on-disk path works (no authorizer is installed).
func TestNoSandboxUnrestricted(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	conn := sandboxConn(t, nil) // unrestricted

	dbPath := filepath.Join(tmp, "legit.db")
	if _, err := conn.ExecContext(ctx, "ATTACH DATABASE '"+dbPath+"' AS extra"); err != nil {
		t.Errorf("unrestricted ATTACH should work, got: %v", err)
	}
}
