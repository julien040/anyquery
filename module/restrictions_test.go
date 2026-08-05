package module

import (
	"os"
	"path/filepath"
	"testing"
)

// checkSrc parses raw and checks it against r, folding a parse error and a
// policy error into the same return value — restrictions_test.go mostly
// cares about "was this denied", not which of the two layers denied it.
func checkSrc(r *Restrictions, raw string) error {
	s, err := ParseSource(raw)
	if err != nil {
		return err
	}
	return r.Check(s)
}

func TestRestrictionsNilIsUnrestricted(t *testing.T) {
	var r *Restrictions // nil
	if err := checkSrc(r, "/etc/passwd"); err != nil {
		t.Errorf("nil restrictions should allow any source, got %v", err)
	}
	if err := checkSrc(r, "http://169.254.169.254/"); err != nil {
		t.Errorf("nil restrictions should allow remote, got %v", err)
	}
	if err := r.CheckFileRead("/etc/shadow"); err != nil {
		t.Errorf("nil restrictions should allow any file, got %v", err)
	}
	if !r.AllowAttachPath("/etc/cron.d/pwn") {
		t.Errorf("nil restrictions should allow any attach")
	}
}

func TestCheckSourceRemote(t *testing.T) {
	denied := &Restrictions{AllowRemote: false}
	if err := checkSrc(denied, "http://169.254.169.254/latest/meta-data/"); err == nil {
		t.Error("expected remote fetch to be denied when AllowRemote is false")
	}
	allowed := &Restrictions{AllowRemote: true}
	if err := checkSrc(allowed, "https://example.com/data.csv"); err != nil {
		t.Errorf("expected remote fetch to be allowed when AllowRemote is true, got %v", err)
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
	if err := os.WriteFile(filepath.Join(allowed, "x.csv"), []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(allowed, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(allowed, "sub", "x.csv"), []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sibling, "x.csv"), []byte("SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &Restrictions{AllowedDirs: []string{allowed}}

	if err := r.CheckFileRead(filepath.Join(allowed, "x.csv")); err != nil {
		t.Errorf("file directly in allowed dir should pass, got %v", err)
	}
	if err := r.CheckFileRead(filepath.Join(allowed, "sub", "x.csv")); err != nil {
		t.Errorf("file nested in allowed dir should pass, got %v", err)
	}
	if err := r.CheckFileRead(filepath.Join(sibling, "x.csv")); err == nil {
		t.Error("a sibling dir sharing a name prefix must NOT be treated as allowed")
	}
	if err := r.CheckFileRead("/etc/passwd"); err == nil {
		t.Error("a path outside the allowed dir must be denied")
	}
}

func TestCheckFileReadMissingButAllowedIsNotADenial(t *testing.T) {
	// Callers distinguish "refused by the policy" from "the file could not be
	// opened": controller/shell.go's `.read` echoes the former and hides the
	// latter, since an OS error tells the caller whether a path exists. The
	// two must therefore stay distinguishable — a missing file inside an
	// allowed dir is a plain not-exist error, not the sandbox-denial message.
	allowed := t.TempDir()
	r := &Restrictions{AllowedDirs: []string{allowed}}
	err := r.CheckFileRead(filepath.Join(allowed, "does-not-exist.csv"))
	if err == nil {
		t.Fatalf("expected an error for a nonexistent file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("got %v, want a not-exist error, not a sandbox denial", err)
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

// TestCheckSourceFileURIBypass is the port of the original CheckSource-based
// test: every spelling that go-getter would once have resolved to the
// absolute secret path must still be denied — now via ParseSource, which
// either resolves the same u.Path (file:// forms) or, for an opaque/bare
// spelling, resolves to a filename that simply doesn't match anything
// permitted. Either layer denying is sufficient; the point is no read escapes
// the allowed dir.
func TestCheckSourceFileURIBypass(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "data")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir := filepath.Join(allowed, "x.csv")
	if err := os.WriteFile(inDir, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &Restrictions{AllowedDirs: []string{allowed}}

	denied := []string{
		"file:" + secret,            // opaque form: not a scheme_url production, treated as an opaque local path
		"file://localhost" + secret, // host form
		"file://" + secret,          // file:///abs form
		"FILE:" + secret,            // uppercase scheme, opaque form
		"File://localhost" + secret, // mixed-case scheme + host form
		secret,                      // plain absolute path
	}
	for _, src := range denied {
		if err := checkSrc(r, src); err == nil {
			t.Errorf("%q should be denied (escapes allowed dir), but passed", src)
		}
	}

	// Percent-encoding must not sneak past: url.Parse decodes %2e%2e to ".." in
	// u.Path, and OpenLocal's filepath.Abs collapses it before the containment
	// check runs, so the escape is caught regardless.
	encoded := "file://" + filepath.Join(allowed, "%2e%2e", "secret.txt")
	if err := checkSrc(r, encoded); err == nil {
		t.Errorf("%q with percent-encoded traversal should be denied", encoded)
	}

	// A legitimate in-dir source must still pass (no false positive).
	if err := checkSrc(r, "file://"+inDir); err != nil {
		t.Errorf("file://%s inside the allowed dir should pass, got %v", inDir, err)
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

// TestForcedFileGetterEscapeRemainsDenied: "file::" wrapping any inner scheme
// must be rejected structurally. ParseSource supports no forced getter at all
// (see TestParseSourceForcedGettersRejected), so these fail at parse time,
// before any policy — including AllowRemote — is ever consulted, and there is
// no containment check left for a synthetic path to be resolved against.
func TestForcedFileGetterEscapeRemainsDenied(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "data")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}

	denied := []string{
		"file::http://attacker.example/etc/passwd",
		"file::https://attacker.example/etc/passwd",
		"file::s3::http://attacker.example/etc/passwd",
		"file::/etc/passwd",
	}
	for _, src := range denied {
		if _, err := ParseSource(src); err == nil {
			t.Errorf("ParseSource(%q) should be rejected outright (no forced getter is supported)", src)
		}
	}

	// The trap: even AllowRemote: true cannot rescue these, because
	// ParseSource fails before a Source ever exists to Check.
	remoteAllowed := &Restrictions{AllowedDirs: []string{allowed}, AllowRemote: true}
	if err := checkSrc(remoteAllowed, "file::http://attacker.example/etc/passwd"); err == nil {
		t.Error("file::http://... must be denied even when AllowRemote is true")
	}

	// A legitimate in-dir source must still pass (no false positive from the
	// forced-getter rejection).
	inDir := filepath.Join(allowed, "x.csv")
	if err := os.WriteFile(inDir, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &Restrictions{AllowedDirs: []string{allowed}}
	if err := checkSrc(r, inDir); err != nil {
		t.Errorf("%q inside the allowed dir should pass, got %v", inDir, err)
	}
}

func TestAllowAttachPath(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "db")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	inDir := filepath.Join(allowed, "ok.db") // deliberately not created: ATTACH creates it
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

	// AllowAttach enabled: on-disk permitted only within allowed dirs, and the
	// leaf file is not required to exist yet.
	withAttach := &Restrictions{AllowedDirs: []string{allowed}, AllowAttach: true}
	if !withAttach.AllowAttachPath(inDir) {
		t.Errorf("on-disk attach within allowed dir should be permitted, even for a not-yet-created file")
	}
	if withAttach.AllowAttachPath(outDir) {
		t.Errorf("on-disk attach outside allowed dirs must be denied")
	}
	if !withAttach.AllowAttachPath(":memory:") {
		t.Errorf("in-memory attach should always be permitted")
	}

	// Documented behavior change from the pre-os.Root implementation: that
	// version walked up to the longest *existing* ancestor, so an ATTACH
	// into a not-yet-created subdirectory of an allowed dir passed
	// containment (and then failed at SQLite's own open with a plain
	// "unable to open database file"). allowsAttachTarget instead requires
	// the immediate parent directory to already exist, so this now denies
	// at the sandbox layer instead — a different, and for an operator more
	// alarming-looking, error for the same ultimately-unusable path. This is
	// an accepted, deliberate narrowing (SQLite would never have created the
	// missing directory either), not a regression to silently fix.
	if withAttach.AllowAttachPath(filepath.Join(allowed, "no-such-subdir", "x.db")) {
		t.Errorf("attach into a nonexistent subdirectory of an allowed dir should be denied (parent dir must already exist)")
	}
}

func TestEmptyRestrictionsLockedDown(t *testing.T) {
	r := &Restrictions{} // zero value = maximally restrictive
	if err := checkSrc(r, "/any/file"); err == nil {
		t.Error("zero-value restrictions should deny all local reads (no allowed dirs)")
	}
	if err := checkSrc(r, "http://x/"); err == nil {
		t.Error("zero-value restrictions should deny remote")
	}
	if r.AllowAttachPath("/tmp/x.db") {
		t.Error("zero-value restrictions should deny on-disk attach")
	}
	if !r.AllowAttachPath(":memory:") {
		t.Error("zero-value restrictions should still allow in-memory attach")
	}
}
