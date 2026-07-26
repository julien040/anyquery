package namespace

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/module"
)

// sandboxDevConn is sandboxConn (see sandbox_test.go) plus DevMode, so both
// gates on the dev-UDF registration (n.devMode && n.restrictions == nil) can be
// exercised independently of the CLI layer.
func sandboxDevConn(t *testing.T, r *module.Restrictions) *sql.Conn {
	t.Helper()
	ns, err := NewNamespace(NamespaceConfig{
		InMemory:     true,
		Logger:       hclog.NewNullLogger(),
		DevMode:      true,
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

// TestDevModeSandboxedFunctionUnavailable covers the fail-open gap the finding
// describes: a Namespace built with DevMode: true and a non-nil Restrictions
// (something a programmatic caller can still do even though the CLI can no
// longer produce this combination now that --dev implies --no-sandbox) must
// not register load_dev_plugin. If it did, the SQLITE_FUNCTION authorizer
// branch would be reached but the name isn't consulted at all by the
// registration gate, so the only thing that can save us is not registering
// the function in the first place.
func TestDevModeSandboxedFunctionUnavailable(t *testing.T) {
	ctx := context.Background()
	conn := sandboxDevConn(t, &module.Restrictions{AllowedDirs: []string{t.TempDir()}})

	for _, fn := range []string{
		"load_dev_plugin('x', 'y')",
		"reload_dev_plugin('x')",
		"unload_dev_plugin('x')",
	} {
		_, err := conn.ExecContext(ctx, "SELECT "+fn)
		if err == nil {
			t.Fatalf("expected %s to be unavailable under DevMode+sandbox", fn)
		}
		if !strings.Contains(strings.ToLower(err.Error()), "no such function") {
			t.Errorf("expected %s to fail with 'no such function', got: %v", fn, err)
		}
	}
}

// TestDevModeUnsandboxedFunctionAvailable confirms the developer workflow is
// preserved: DevMode with a nil Restrictions (the CLI's `--dev` without
// --sandbox, or `server --dev` after --dev forces restrictions to nil) still
// registers load_dev_plugin. The manifest path points at a nonexistent file
// under a fresh temp dir so the call fails fast at os.ReadFile and never
// reaches exec.Command/os.OpenFile — we only need to distinguish "registered"
// from "not registered", not exercise LoadDevPlugin's internals.
func TestDevModeUnsandboxedFunctionAvailable(t *testing.T) {
	ctx := context.Background()
	conn := sandboxDevConn(t, nil) // unrestricted: the developer workflow

	missing := filepath.Join(t.TempDir(), "does-not-exist.json")
	var out string
	err := conn.QueryRowContext(ctx, "SELECT load_dev_plugin('x', '"+missing+"')").Scan(&out)
	if err != nil {
		t.Fatalf("load_dev_plugin should be registered without a sandbox, got query error: %v", err)
	}
	if strings.Contains(strings.ToLower(out), "no such function") {
		t.Errorf("load_dev_plugin should be registered without a sandbox, got: %q", out)
	}
	if !strings.Contains(out, "error reading manifest") {
		t.Errorf("expected the call to fail at os.ReadFile on the missing manifest, got: %q", out)
	}
}

// TestDevModeSandboxedOtherFunctionsUnaffected confirms the gate is scoped to
// the three dev UDFs: ordinary sandboxed behavior (e.g. load_file denied by
// the pre-existing denylist) is unchanged when DevMode is also set.
func TestDevModeSandboxedOtherFunctionsUnaffected(t *testing.T) {
	ctx := context.Background()
	allowed := t.TempDir()
	csvPath := filepath.Join(allowed, "data.csv")
	if err := os.WriteFile(csvPath, []byte("name\nalice\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	conn := sandboxDevConn(t, &module.Restrictions{AllowedDirs: []string{allowed}})

	if _, err := conn.ExecContext(ctx, "SELECT load_file('/etc/passwd')"); err == nil {
		t.Fatal("expected load_file('/etc/passwd') to remain denied under DevMode+sandbox")
	}
	if _, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE ok USING csv_reader('"+csvPath+"', header=true)"); err != nil {
		t.Fatalf("csv_reader on an allowed path should still work under DevMode+sandbox, got: %v", err)
	}
}
