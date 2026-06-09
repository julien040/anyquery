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
