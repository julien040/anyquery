package module

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", p, err)
	}
	return p
}

func TestOpenLocalNilUnrestricted(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "f.txt", "hello")
	var r *Restrictions
	f, err := r.OpenLocal(p)
	if err != nil {
		t.Fatalf("OpenLocal with nil Restrictions: %v", err)
	}
	f.Close()

	// Anywhere at all, since nil is unrestricted.
	f2, err := r.OpenLocal("/etc/hosts")
	if err == nil {
		f2.Close()
	}
	// Not asserting success (platform-dependent), just that nil never
	// produces a sandbox-denial error.
	if err != nil && strings.Contains(err.Error(), "sandbox:") {
		t.Fatalf("nil Restrictions produced a sandbox error: %v", err)
	}
}

func TestOpenLocalContainment(t *testing.T) {
	allowed := t.TempDir()
	nested := filepath.Join(allowed, "sub")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, allowed, "top.txt", "top")
	writeTempFile(t, nested, "deep.txt", "deep")

	outside := t.TempDir()
	secretOutside := writeTempFile(t, outside, "secret.txt", "SECRET")

	// Sibling-prefix trap: "allowed-secret" must not be treated as inside
	// "allowed" by a naive strings.HasPrefix check.
	trapDir := allowed + "-secret"
	if err := os.Mkdir(trapDir, 0o700); err != nil {
		t.Fatal(err)
	}
	trapFile := writeTempFile(t, trapDir, "trap.txt", "TRAP")

	r := &Restrictions{AllowedDirs: []string{allowed}}

	if f, err := r.OpenLocal(filepath.Join(allowed, "top.txt")); err != nil {
		t.Fatalf("allowed top-level file denied: %v", err)
	} else {
		f.Close()
	}
	if f, err := r.OpenLocal(filepath.Join(nested, "deep.txt")); err != nil {
		t.Fatalf("allowed nested file denied: %v", err)
	} else {
		f.Close()
	}
	if _, err := r.OpenLocal(secretOutside); err == nil {
		t.Fatalf("file outside AllowedDirs was permitted")
	}
	if _, err := r.OpenLocal(trapFile); err == nil {
		t.Fatalf("sibling-prefix trap directory was permitted")
	}
}

func TestOpenLocalSymlinkEscape(t *testing.T) {
	allowed := t.TempDir()
	outside := t.TempDir()
	secret := writeTempFile(t, outside, "secret.txt", "SECRET")

	link := filepath.Join(allowed, "escape")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	r := &Restrictions{AllowedDirs: []string{allowed}}
	if f, err := r.OpenLocal(link); err == nil {
		f.Close()
		t.Fatalf("symlink escaping the allowed dir was permitted")
	}
}

// TestOpenLocalSymlinkedAllowedDirParent covers the macOS /var -> /private/var
// shape: the allowed dir's own ancestor is a symlink, not the leaf.
func TestOpenLocalSymlinkedAllowedDirParent(t *testing.T) {
	real := t.TempDir()
	realAllowed := filepath.Join(real, "allowed")
	if err := os.Mkdir(realAllowed, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, realAllowed, "f.txt", "hi")

	linkParent := filepath.Join(t.TempDir(), "linked-parent")
	if err := os.Symlink(real, linkParent); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	allowedViaLink := filepath.Join(linkParent, "allowed")

	r := &Restrictions{AllowedDirs: []string{allowedViaLink}}
	if f, err := r.OpenLocal(filepath.Join(allowedViaLink, "f.txt")); err != nil {
		t.Fatalf("file under a symlinked allowed-dir parent was denied: %v", err)
	} else {
		f.Close()
	}
	// And the real (non-symlinked) path to the same file must also work,
	// since resolvedhe allowed dir is canonicalized once at construction.
	if f, err := r.OpenLocal(filepath.Join(realAllowed, "f.txt")); err != nil {
		t.Fatalf("file via the real allowed-dir path was denied: %v", err)
	} else {
		f.Close()
	}
}

// TestOpenLocalTOCTOU swaps a symlink between two OpenLocal calls and checks
// that the second call is re-evaluated live, not served from a stale result:
// containment is enforced by os.Root at open time, every time, not cached
// from an earlier check. Symlink targets are relative — an absolute-path
// symlink is rejected by os.Root outright regardless of where it points
// (tested separately in TestOpenLocalSymlinkEscape), so a relative target is
// what makes this a same-directory-vs-escaping-directory comparison.
func TestOpenLocalTOCTOU(t *testing.T) {
	allowed := t.TempDir()
	writeTempFile(t, allowed, "inside.txt", "inside")
	outside := t.TempDir()
	outsideTarget := writeTempFile(t, outside, "secret.txt", "SECRET")
	relOutside, err := filepath.Rel(allowed, outsideTarget)
	if err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(allowed, "swap")
	if err := os.Symlink("inside.txt", link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	r := &Restrictions{AllowedDirs: []string{allowed}}
	if f, err := r.OpenLocal(link); err != nil {
		t.Fatalf("initial relative symlink-to-inside was denied: %v", err)
	} else {
		f.Close()
	}

	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(relOutside, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	if f, err := r.OpenLocal(link); err == nil {
		f.Close()
		t.Fatalf("symlink swapped to a relative path outside AllowedDirs was permitted")
	}
}

// TestOpenLocalNonexistentLeaf checks that a missing file under an allowed
// dir surfaces as a plain not-exist error, not the sandbox-denial message —
// callers (e.g. load_file) distinguish "denied" from "not found".
func TestOpenLocalNonexistentLeaf(t *testing.T) {
	allowed := t.TempDir()
	r := &Restrictions{AllowedDirs: []string{allowed}}

	_, err := r.OpenLocal(filepath.Join(allowed, "does-not-exist.txt"))
	if err == nil {
		t.Fatalf("expected an error for a nonexistent leaf")
	}
	if strings.Contains(err.Error(), "sandbox: access to") {
		t.Fatalf("nonexistent leaf under an allowed dir produced a sandbox-denial error: %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestReadLocalFileNilUnrestricted(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "f.txt", "hello")

	var r *Restrictions
	got, err := r.ReadLocalFile(p)
	if err != nil {
		t.Fatalf("ReadLocalFile with nil Restrictions: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}

	// Anywhere at all, since nil is unrestricted. Success is
	// platform-dependent, so only the absence of a denial is asserted.
	if _, err := r.ReadLocalFile("/etc/hosts"); err != nil && strings.Contains(err.Error(), "sandbox:") {
		t.Fatalf("nil Restrictions produced a sandbox error: %v", err)
	}
}

func TestReadLocalFileContainment(t *testing.T) {
	allowed := t.TempDir()
	nested := filepath.Join(allowed, "sub")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, allowed, "top.txt", "top")
	writeTempFile(t, nested, "deep.txt", "deep")

	outside := t.TempDir()
	secretOutside := writeTempFile(t, outside, "secret.txt", "SECRET")

	r := &Restrictions{AllowedDirs: []string{allowed}}

	got, err := r.ReadLocalFile(filepath.Join(allowed, "top.txt"))
	if err != nil {
		t.Fatalf("allowed top-level file denied: %v", err)
	}
	if string(got) != "top" {
		t.Fatalf("got %q, want %q", got, "top")
	}

	got, err = r.ReadLocalFile(filepath.Join(nested, "deep.txt"))
	if err != nil {
		t.Fatalf("allowed nested file denied: %v", err)
	}
	if string(got) != "deep" {
		t.Fatalf("got %q, want %q", got, "deep")
	}

	got, err = r.ReadLocalFile(secretOutside)
	if err == nil {
		t.Fatalf("file outside AllowedDirs was read, content %q", got)
	}
	// controller/shell.go's `.read` echoes a refusal but hides OS errors, and
	// tells them apart by this prefix.
	if !strings.HasPrefix(err.Error(), "sandbox: ") {
		t.Fatalf("a refusal by the policy must be prefixed %q, got: %v", "sandbox: ", err)
	}
}

// TestReadLocalFileSymlinkEscape covers the point of reading through the
// handle: containment is the kernel's answer to the read's own open, so the
// escape fails at the read itself rather than at a separate check that a
// subsequent open could no longer be relied on to match.
func TestReadLocalFileSymlinkEscape(t *testing.T) {
	allowed := t.TempDir()
	outside := t.TempDir()
	secret := writeTempFile(t, outside, "secret.txt", "SECRET")

	link := filepath.Join(allowed, "escape")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	r := &Restrictions{AllowedDirs: []string{allowed}}
	got, err := r.ReadLocalFile(link)
	if err == nil {
		t.Fatalf("symlink escaping the allowed dir was read, content %q", got)
	}
	if strings.Contains(string(got), "SECRET") {
		t.Fatalf("content outside the allowed dir leaked: %q", got)
	}
}

// TestReadLocalFileNonexistentLeaf mirrors TestOpenLocalNonexistentLeaf: a
// missing file under an allowed dir must surface as a plain not-exist error,
// never as the sandbox-denial message, since callers (controller/shell.go's
// `.read`) key on that prefix to decide whether the error is safe to echo.
func TestReadLocalFileNonexistentLeaf(t *testing.T) {
	allowed := t.TempDir()
	r := &Restrictions{AllowedDirs: []string{allowed}}

	_, err := r.ReadLocalFile(filepath.Join(allowed, "does-not-exist.txt"))
	if err == nil {
		t.Fatalf("expected an error for a nonexistent leaf")
	}
	if strings.HasPrefix(err.Error(), "sandbox: ") {
		t.Fatalf("nonexistent leaf under an allowed dir produced a sandbox-denial error: %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestOpenLocalEmptyPath(t *testing.T) {
	r := &Restrictions{AllowedDirs: []string{t.TempDir()}}
	if _, err := r.OpenLocal(""); err == nil {
		t.Fatalf("empty path was permitted")
	}
	if _, err := r.OpenLocal("   "); err == nil {
		t.Fatalf("whitespace-only path was permitted")
	}
}

func TestOpenLocalEmptyAllowedDirsLocksDown(t *testing.T) {
	r := &Restrictions{}
	dir := t.TempDir()
	p := writeTempFile(t, dir, "f.txt", "hi")
	if _, err := r.OpenLocal(p); err == nil {
		t.Fatalf("zero-value Restrictions permitted a local read")
	}
}
