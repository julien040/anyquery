package namespace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/julien040/anyquery/module"
)

// TestSandboxQueryFragmentTraversal: a query/fragment appended to a bare
// local path must never let a read escape --allow-dirs.
//
// The original bug was a *divergence*: the fetch layer opened only the part
// of the source before "?"/"#" (the real, unrelated absolute path), while
// the sandbox check validated the whole string, which filepath.Clean folds
// into something that looks contained. Checked path and opened path
// disagreed.
//
// ParseSource removes the divergence rather than re-deriving the exact
// bytes the fetch layer would have opened: a local path is opaque (never
// split on "?"/"#"), so whatever OpenLocal checks is exactly what it opens. That
// means a payload shaped like "/etc/passwd?/../../{allowed}/decoy" is,
// under ordinary POSIX path semantics with "?" as a plain filename
// character, *literally a path to {allowed}/decoy* — and reading that file
// is correct, not a bypass, once check and open are the same operation.
// This test's invariant is therefore not "must be denied" but "must never
// return content from outside the allowed dir": either the create errors,
// or the query returns exactly the decoy's own content.
func TestSandboxQueryFragmentTraversal(t *testing.T) {
	ctx := context.Background()
	allowed := t.TempDir()
	decoyPath := filepath.Join(allowed, "decoy")
	const decoyMarker = "DECOY-MARKER-9f3a"
	if err := os.WriteFile(decoyPath, []byte("name\n"+decoyMarker+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	conn := sandboxConn(t, &module.Restrictions{AllowedDirs: []string{allowed}})

	payloads := []string{
		"/etc/passwd?/../../../../../../../" + allowed + "/decoy",
		"/etc/passwd#/../../../../../../../" + allowed + "/decoy",
	}
	for i, payload := range payloads {
		table := "x" + string(rune('a'+i))
		_, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE "+table+" USING csv_reader('"+payload+"', header=true)")
		if err != nil {
			continue // denied outright: an acceptable outcome
		}
		var name string
		if err := conn.QueryRowContext(ctx, "SELECT name FROM "+table).Scan(&name); err != nil {
			t.Fatalf("payload %q: table created but query failed: %v", payload, err)
		}
		if name != decoyMarker {
			t.Fatalf("payload %q: expected either denial or the decoy's own content (%q), got %q — a real cross-boundary read", payload, decoyMarker, name)
		}
	}

	// A payload that genuinely cannot collapse into the allowed dir (an
	// insufficient/wrong traversal depth) must be denied outright — this
	// one exercises the deny path itself, not the benign-collapse path.
	genuinelyOutside := "/etc/passwd?/../../nonexistent-dir-xyz/file.csv"
	if _, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE z USING csv_reader('"+genuinelyOutside+"')"); err == nil {
		t.Fatalf("payload %q should be denied (does not resolve inside the allowed dir)", genuinelyOutside)
	}
}

// TestSandboxGitForcedGetterRejected: git::/hg:: forced getters used to
// bypass the local-path containment check entirely once classified as
// "remote", so --allow-remote turned into unrestricted local disk read of
// any git/hg working repo. No forced getter is a supported source at all, so
// these are a parse error — denied regardless of AllowRemote.
func TestSandboxGitForcedGetterRejected(t *testing.T) {
	ctx := context.Background()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("SECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// AllowRemote: true is the exact condition under which the bypass used to trigger.
	conn := sandboxConn(t, &module.Restrictions{AllowedDirs: []string{t.TempDir()}, AllowRemote: true})

	payloads := []string{
		"git::file://" + secret,
		"git::" + secret,
		"hg::file://" + secret,
	}
	for i, payload := range payloads {
		table := "y" + string(rune('a'+i))
		_, err := conn.ExecContext(ctx, "CREATE VIRTUAL TABLE "+table+" USING csv_reader('"+payload+"')")
		if err == nil {
			t.Fatalf("payload %q: expected csv_reader to be denied even with AllowRemote: true", payload)
		}
	}
}
