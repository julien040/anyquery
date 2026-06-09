package namespace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/julien040/anyquery/module"
)

// TestSandboxDeniesDangerousFunctions covers follow-up issues #1 and #2 over the
// SQL surface: the file-reading and cache-deleting scalar functions are blocked
// by the sandbox authorizer (SQLITE_FUNCTION deny-list), so the LFR and the
// arbitrary-directory-delete PoCs cannot run.
func TestSandboxDeniesDangerousFunctions(t *testing.T) {
	ctx := context.Background()
	conn := sandboxConn(t, &module.Restrictions{AllowedDirs: []string{t.TempDir()}})

	cases := []string{
		"SELECT load_file('/etc/passwd')",
		"SELECT load_file_bytes('/etc/passwd')",
		"SELECT clear_plugin_cache('github')",
		"SELECT clear_file_cache()",
		// load_extension must never become reachable (RCE). go-sqlite3 disables
		// the SQL function by default; the authorizer also deny-lists it.
		"SELECT load_extension('/tmp/evil.so')",
	}
	for _, q := range cases {
		if _, err := conn.ExecContext(ctx, q); err == nil {
			t.Errorf("expected %q to be denied under sandbox", q)
		}
	}
}

// TestSandboxPragmaAllowlist covers follow-up issue #8: only read-only
// introspection pragmas are allowed; schema-corruption and memory-inflation
// pragmas are denied.
func TestSandboxPragmaAllowlist(t *testing.T) {
	ctx := context.Background()
	conn := sandboxConn(t, &module.Restrictions{AllowedDirs: []string{t.TempDir()}})

	if _, err := conn.ExecContext(ctx, "CREATE TABLE t (a INT, b TEXT)"); err != nil {
		t.Fatalf("CREATE TABLE should be allowed under sandbox, got: %v", err)
	}

	// Denied: write/escalation/memory pragmas.
	for _, q := range []string{
		"PRAGMA writable_schema = ON",
		"PRAGMA cache_size = 1000000",
		"PRAGMA mmap_size = 1000000000",
	} {
		if _, err := conn.ExecContext(ctx, q); err == nil {
			t.Errorf("expected %q to be denied under sandbox", q)
		}
	}

	// Allowed: read-only introspection pragmas the engine/handlers rely on.
	// Close the rows so the dedicated connection is released before cleanup.
	for _, q := range []string{"PRAGMA table_info(t)", "PRAGMA database_list"} {
		rows, err := conn.QueryContext(ctx, q)
		if err != nil {
			t.Errorf("%q should be allowed under sandbox, got: %v", q, err)
			continue
		}
		rows.Close()
	}
}

// TestNoSandboxLoadFileWorks confirms a nil policy leaves load_file unchanged:
// it reads files as before. (CheckFileRead is a no-op on a nil receiver.)
func TestNoSandboxLoadFileWorks(t *testing.T) {
	ctx := context.Background()
	conn := sandboxConn(t, nil) // unrestricted

	path := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got string
	if err := conn.QueryRowContext(ctx, "SELECT load_file('"+path+"')").Scan(&got); err != nil {
		t.Fatalf("load_file should work without a sandbox, got: %v", err)
	}
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

// TestClearPluginCachePathTraversal covers the core of issue #2: the path-
// traversal hardening in clear_plugin_cache itself, independent of the
// authorizer. A traversal payload must be rejected without deleting anything
// outside the cache root; a plain name still works.
func TestClearPluginCachePathTraversal(t *testing.T) {
	// Isolate the cache root so the test never touches the real cache.
	cacheHome := t.TempDir()
	orig := xdg.CacheHome
	xdg.CacheHome = cacheHome
	t.Cleanup(func() { xdg.CacheHome = orig })

	pluginsRoot := filepath.Join(cacheHome, "anyquery", "plugins")

	// A legitimate plugin directory is removed and reports success.
	legit := filepath.Join(pluginsRoot, "github")
	if err := os.MkdirAll(legit, 0o755); err != nil {
		t.Fatal(err)
	}
	if msg := clear_plugin_cache("github"); msg != "" {
		t.Errorf("clear_plugin_cache(\"github\") should succeed, got: %q", msg)
	}
	if _, err := os.Stat(legit); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed", legit)
	}

	// A victim directory outside the cache root must survive a traversal payload
	// that resolves to it.
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.MkdirAll(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	traversal, err := filepath.Rel(pluginsRoot, victim)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(traversal, "..") {
		t.Fatalf("test setup: expected a traversal payload with \"..\", got %q", traversal)
	}
	if msg := clear_plugin_cache(traversal); msg != "invalid plugin name" {
		t.Errorf("traversal payload should be rejected, got: %q", msg)
	}
	if _, err := os.Stat(victim); err != nil {
		t.Errorf("victim directory must survive a traversal payload: %v", err)
	}

	// A bare separator is rejected too.
	if msg := clear_plugin_cache("a/b"); msg != "invalid plugin name" {
		t.Errorf("separator payload should be rejected, got: %q", msg)
	}
}
