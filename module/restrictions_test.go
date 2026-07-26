package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestrictionsNilIsUnrestricted(t *testing.T) {
	var r *Restrictions // nil
	if err := r.CheckSource("/etc/passwd"); err != nil {
		t.Errorf("nil restrictions should allow any source, got %v", err)
	}
	if err := r.CheckSource("http://169.254.169.254/"); err != nil {
		t.Errorf("nil restrictions should allow remote, got %v", err)
	}
	if err := r.CheckFileRead("/etc/shadow"); err != nil {
		t.Errorf("nil restrictions should allow any file, got %v", err)
	}
	if !r.AllowAttachPath("/etc/cron.d/pwn") {
		t.Errorf("nil restrictions should allow any attach")
	}
}

func TestIsRemoteSource(t *testing.T) {
	cases := map[string]bool{
		"http://example.com/x":  true,
		"https://example.com/x": true,
		"s3://bucket/key":       true,
		"git::https://x/y":      true,
		"file:///etc/passwd":    false,
		"file::/etc/passwd":     false,
		"/etc/passwd":           false,
		"data.csv":              false,
		"./rel/data.csv":        false,
		`C:\data\x.csv`:         false,
	}
	for src, want := range cases {
		if got := isRemoteSource(src); got != want {
			t.Errorf("isRemoteSource(%q) = %v, want %v", src, got, want)
		}
	}
}

func TestCheckSourceRemote(t *testing.T) {
	denied := &Restrictions{AllowRemote: false}
	if err := denied.CheckSource("http://169.254.169.254/latest/meta-data/"); err == nil {
		t.Error("expected remote fetch to be denied when AllowRemote is false")
	}
	allowed := &Restrictions{AllowRemote: true}
	if err := allowed.CheckSource("https://example.com/data.csv"); err != nil {
		t.Errorf("expected remote fetch to be allowed when AllowRemote is true, got %v", err)
	}
}

func TestCheckSourceEmpty(t *testing.T) {
	r := &Restrictions{}
	if err := r.CheckSource(""); err == nil {
		t.Error("expected empty source to be denied")
	}
}

func TestCheckFileReadContainment(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "data")
	sibling := filepath.Join(root, "data-secret") // prefix-of-allowed trap
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}

	r := &Restrictions{AllowedDirs: []string{allowed}}

	if err := r.CheckFileRead(filepath.Join(allowed, "x.csv")); err != nil {
		t.Errorf("file directly in allowed dir should pass, got %v", err)
	}
	if err := r.CheckFileRead(filepath.Join(allowed, "sub", "x.csv")); err != nil {
		t.Errorf("file nested in allowed dir should pass, got %v", err)
	}
	if err := r.CheckFileRead(allowed); err != nil {
		t.Errorf("the allowed dir itself should pass, got %v", err)
	}
	if err := r.CheckFileRead(filepath.Join(sibling, "x.csv")); err == nil {
		t.Error("a sibling dir sharing a name prefix must NOT be treated as allowed")
	}
	if err := r.CheckFileRead("/etc/passwd"); err == nil {
		t.Error("a path outside the allowed dir must be denied")
	}
}

func TestCheckFileReadSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "data")
	secret := filepath.Join(root, "secret")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secret, 0o755); err != nil {
		t.Fatal(err)
	}
	secretFile := filepath.Join(secret, "x.csv")
	if err := os.WriteFile(secretFile, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A symlink inside the allowed dir that points outside it.
	link := filepath.Join(allowed, "link")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	r := &Restrictions{AllowedDirs: []string{allowed}}
	if err := r.CheckFileRead(filepath.Join(link, "x.csv")); err == nil {
		t.Error("a symlink escaping the allowed dir must be denied")
	}
}

func TestCheckSourceFileURIBypass(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "data")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	// A secret file outside the allowed directory.
	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A legitimate file inside the allowed directory.
	inDir := filepath.Join(allowed, "x.csv")
	if err := os.WriteFile(inDir, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &Restrictions{AllowedDirs: []string{allowed}}

	// Every file: spelling that go-getter would resolve to the absolute secret
	// path must be denied. These are validated against the same path go-getter
	// opens (u.Path), not the raw string.
	denied := []string{
		"file:" + secret,                  // single-slash form (was checked as relative)
		"file://localhost" + secret,       // host form (host ignored by go-getter)
		"file://" + secret,                // file:///abs form
		"file::file://localhost" + secret, // forced-getter + host form
		"FILE:" + secret,                  // uppercase scheme (net/url lowercases it)
		"File://localhost" + secret,       // mixed-case scheme + host form
		secret,                            // plain absolute path
	}
	for _, src := range denied {
		if err := r.CheckSource(src); err == nil {
			t.Errorf("CheckSource(%q) should be denied (escapes allowed dir), but passed", src)
		}
	}

	// Percent-encoding must not sneak past: our u.Path resolves %2e%2e to ".."
	// and denies; go-getter would stat the literal %2e path and fail. Either
	// way, no read outside the allowed dir.
	encoded := "file://" + filepath.Join(allowed, "%2e%2e", "secret.txt")
	if err := r.CheckSource(encoded); err == nil {
		t.Errorf("CheckSource(%q) with percent-encoded traversal should be denied", encoded)
	}

	// A legitimate in-dir source must still pass (no false positive).
	if err := r.CheckSource("file://" + inDir); err != nil {
		t.Errorf("CheckSource(%q) for a file inside the allowed dir should pass, got %v", "file://"+inDir, err)
	}
}

// TestIsInMemoryDB pins the mode= duplicate-key behavior directly. Go's
// url.Values.Get returns the FIRST value of a repeated key while SQLite's own
// URI parser resolves the LAST, so file:/tmp/x?mode=memory&mode=rwc used to be
// classified in-memory (and so exempted from the disk-write gate in
// AllowAttachPath) while SQLite actually opened it read-write-create on disk.
// Any duplicate mode key is now denied outright rather than resolved, so this
// divergence can no longer be exploited regardless of which value SQLite
// happens to prefer.
func TestIsInMemoryDB(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"bare memory literal", ":memory:", true},
		{"file uri, non-empty path, shared cache (existing fixture)", "file:memdb1?mode=memory&cache=shared", true},
		{"file uri, empty path", "file:?mode=memory", true},
		{"duplicate mode key, memory then rwc", "file:/tmp/x?mode=memory&mode=rwc", false},
		{"duplicate mode key, percent-encoded memory then rwc", "file:/tmp/x?mode=%6demory&mode=rwc", false},
		{"duplicate mode key, both memory", "file:/tmp/x?mode=memory&mode=memory", false},
		{"mode spoofed as a query VALUE, not a key", "file:/etc/cron.d/pwn?x=mode=memory", false},
		{"case-sensitive MODE key alongside real mode=rwc", "file:/tmp/x?MODE=memory&mode=rwc", false},
	}
	for _, c := range cases {
		if got := isInMemoryDB(c.input); got != c.want {
			t.Errorf("%s: isInMemoryDB(%q) = %v, want %v", c.name, c.input, got, c.want)
		}
	}
}

// TestCheckSourceForcedFileGetterEscape covers the file::<scheme>://... shape:
// isRemoteSource sees the forced getter "file" and calls it local; stripFileScheme
// strips the "file::" prefix and returns the remainder (a URL) unchanged; the
// containment check then resolves the synthetic, non-existent path
// <cwd>/http:/host/... down to its nearest existing ancestor (cwd) and passes.
// Meanwhile go-getter keeps force="file" (a forced getter always wins over
// scheme-based dispatch) and its FileGetter opens the inner URL's u.Path — the
// real absolute path — discarding scheme and host entirely. The fix denies the
// ambiguous shape outright, in CheckSource, before the isRemoteSource/AllowRemote
// branch is ever reached.
func TestCheckSourceForcedFileGetterEscape(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "data")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}

	// The bug required cwd to resolve inside an allowed dir (otherwise the
	// pre-fix containment check already denied it for an unrelated reason);
	// reproduce that precondition here so a regression would actually fail.
	t.Chdir(allowed)

	r := &Restrictions{AllowedDirs: []string{allowed}}

	// Proves the precondition above is real and the fix below is what's doing
	// the denying, not an accident of this test's setup: the OLD code path
	// (strip the file:: prefix, resolve, check containment) on its own still
	// evaluates this as "allowed", because the synthetic, non-existent path
	// <cwd>/http:/attacker.example/etc/passwd has no existing ancestor other
	// than cwd, which resolvePath falls back to — and cwd is inside
	// AllowedDirs. If this assertion ever starts failing, the precondition has
	// rotted and TestCheckSourceForcedFileGetterEscape below would pass even
	// against unpatched code.
	if err := r.checkLocalPath(stripFileScheme("file::http://attacker.example/etc/passwd")); err != nil {
		t.Fatalf("precondition lost: the pre-fix code path no longer resolves inside the allowed dir (%v); "+
			"this test would no longer catch a regression of the file:: forced-getter fix", err)
	}

	// The vulnerable family: file:: wrapping any scheme://, not just http.
	denied := []string{
		"file::http://attacker.example/etc/passwd",
		"file::https://attacker.example/etc/passwd",
		"file::s3::http://attacker.example/etc/passwd", // nested forced getter
	}
	for _, src := range denied {
		if err := r.CheckSource(src); err == nil {
			t.Errorf("CheckSource(%q) should be denied (file:: forced-getter escape), but passed", src)
		}
	}

	// The trap: this must stay denied even when AllowRemote is true. Making
	// isRemoteSource classify this shape as "remote" would let AllowRemote
	// short-circuit CheckSource to nil while go-getter's FileGetter still runs
	// locally — turning --allow-remote into unrestricted local disk read.
	remoteAllowed := &Restrictions{AllowedDirs: []string{allowed}, AllowRemote: true}
	if err := remoteAllowed.CheckSource("file::http://attacker.example/etc/passwd"); err == nil {
		t.Error("file::http://... must be denied even when AllowRemote is true")
	}

	// Negative controls: shapes that must NOT be affected by the new check.
	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustDeny := map[string]string{
		"/etc/passwd":             "plain absolute path outside allowed dir",
		"file:" + secret:          "file: scheme outside allowed dir (fixed pre-existing)",
		"file://" + secret:        "file:// scheme outside allowed dir (fixed pre-existing)",
		"file::file://" + secret:  "nested file:: + file:// (already denied, stays denied)",
		"../secret.txt":           "relative traversal outside allowed dir",
		"git::file:///etc/passwd": "non-file forced getter, caught by the AllowRemote gate",
	}
	for src, why := range mustDeny {
		if err := r.CheckSource(src); err == nil {
			t.Errorf("CheckSource(%q) should still be denied (%s), but passed", src, why)
		}
	}

	// Must-not-break: file::/etc/passwd is a plain forced-file path (no inner
	// scheme), not the wrapped-URL shape, and must still be treated as local
	// (see TestIsRemoteSource) and denied only by ordinary containment.
	if isRemoteSource("file::/etc/passwd") {
		t.Error("file::/etc/passwd must still be classified as local, not remote")
	}
	if err := r.CheckSource("file::/etc/passwd"); err == nil {
		t.Error("file::/etc/passwd should be denied by containment (outside allowed dir), but passed")
	}

	// A legitimate in-dir source must still pass (no false positive from the
	// new check).
	inDir := filepath.Join(allowed, "x.csv")
	if err := os.WriteFile(inDir, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := r.CheckSource(inDir); err != nil {
		t.Errorf("CheckSource(%q) for a plain file inside the allowed dir should pass, got %v", inDir, err)
	}
}

func TestAllowAttachPath(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "db")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	inDir := filepath.Join(allowed, "ok.db")
	outDir := filepath.Join(root, "elsewhere.db")

	// AllowAttach disabled: only in-memory permitted.
	noAttach := &Restrictions{AllowedDirs: []string{allowed}, AllowAttach: false}
	for _, c := range []struct {
		name string
		path string
		want bool
	}{
		{"empty denied", "", false},
		{"memory literal", ":memory:", true},
		{"file uri memory", "file:m.db?mode=memory&cache=shared", true},
		{"spoofed memory in wrong param", "file:/etc/cron.d/pwn?x=mode=memory", false},
		{"on-disk denied when AllowAttach off", inDir, false},
		// Regression: file:<path>?mode=memory&mode=rwc used to be classified
		// in-memory by isInMemoryDB (Go's Query().Get takes the first value)
		// and so was returned as "allowed" here before AllowAttach was ever
		// consulted, while SQLite itself honors the LAST mode= value and opens
		// the file read-write-create on disk — an arbitrary-file-create
		// bypass of the sandbox with AllowAttach left at its default false.
		{"duplicate mode key bypass denied", "file:" + filepath.Join(root, "pwned.db") + "?mode=memory&mode=rwc", false},
		{"duplicate mode key bypass, percent-encoded, denied", "file:" + filepath.Join(root, "pwned2.db") + "?mode=%6demory&mode=rwc", false},
		{"duplicate mode key, both memory, denied", "file:" + filepath.Join(root, "pwned3.db") + "?mode=memory&mode=memory", false},
	} {
		if got := noAttach.AllowAttachPath(c.path); got != c.want {
			t.Errorf("AllowAttach=false AllowAttachPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}

	// AllowAttach enabled: on-disk permitted only within allowed dirs.
	withAttach := &Restrictions{AllowedDirs: []string{allowed}, AllowAttach: true}
	if !withAttach.AllowAttachPath(inDir) {
		t.Errorf("on-disk attach within allowed dir should be permitted")
	}
	if withAttach.AllowAttachPath(outDir) {
		t.Errorf("on-disk attach outside allowed dirs must be denied")
	}
	if !withAttach.AllowAttachPath(":memory:") {
		t.Errorf("in-memory attach should always be permitted")
	}
}

func TestEmptyRestrictionsLockedDown(t *testing.T) {
	r := &Restrictions{} // zero value = maximally restrictive
	if err := r.CheckSource("/any/file"); err == nil {
		t.Error("zero-value restrictions should deny all local reads (no allowed dirs)")
	}
	if err := r.CheckSource("http://x/"); err == nil {
		t.Error("zero-value restrictions should deny remote")
	}
	if r.AllowAttachPath("/tmp/x.db") {
		t.Error("zero-value restrictions should deny on-disk attach")
	}
	if !r.AllowAttachPath(":memory:") {
		t.Error("zero-value restrictions should still allow in-memory attach")
	}
}
