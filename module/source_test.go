package module

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, raw string) Source {
	t.Helper()
	s, err := ParseSource(raw)
	if err != nil {
		t.Fatalf("ParseSource(%q) = %v, want success", raw, err)
	}
	return s
}

func mustError(t *testing.T, raw string) {
	t.Helper()
	if _, err := ParseSource(raw); err == nil {
		t.Fatalf("ParseSource(%q) = nil error, want error", raw)
	}
}

func TestParseSourceEmpty(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t"} {
		mustError(t, raw)
	}
}

func TestParseSourceStdin(t *testing.T) {
	for _, raw := range []string{"stdin", "-", "/dev/stdin"} {
		s := mustParse(t, raw)
		if s.Kind != KindStdin {
			t.Fatalf("ParseSource(%q).Kind = %v, want KindStdin", raw, s.Kind)
		}
	}
}

// TestParseSourceLocalOpaque: a query/fragment on a bare local path must
// never be split off — "/etc/passwd?/../../{allowed}/x" once read outside the
// sandbox because the check and the open disagreed on where "?" ends the
// path. Path must equal the input byte-for-byte.
func TestParseSourceLocalOpaque(t *testing.T) {
	cases := []string{
		"/local/path.csv",
		"relative/path.csv",
		"/etc/passwd?/../../../../../../../allowed/decoy",
		"/etc/passwd#/../../../../../../../allowed/decoy",
		"data-secret/file.csv",
		`C:\Users\me\data.csv`,
		"..%2f..%2fetc/passwd",
		"file:/etc/passwd", // single slash: opaque file: form, not the "//" scheme_url production
		"file:etc/passwd",  // opaque form
	}
	for _, raw := range cases {
		s := mustParse(t, raw)
		if s.Kind != KindLocal {
			t.Fatalf("ParseSource(%q).Kind = %v, want KindLocal", raw, s.Kind)
		}
		if s.Path != raw {
			t.Fatalf("ParseSource(%q).Path = %q, want verbatim %q", raw, s.Path, raw)
		}
	}
}

// TestParseSourceForcedGettersRejected: no "<prefix>::…" input is a supported
// source, whatever the prefix. git::/hg:: could read any local repository,
// file:: wrapping hides an inner scheme from every later check, and s3::/gcs::
// name a transport that does not exist (a presigned https URL replaces it).
// None of them may fall through to a local path named "<prefix>::…" either.
func TestParseSourceForcedGettersRejected(t *testing.T) {
	denied := []string{
		"s3::https://s3.amazonaws.com/bucket/f.json?aws_access_key_id=AKIA&aws_access_key_secret=secret&region=us-east-1",
		"s3::https://s3.amazonaws.com/bucket/f.json",
		"S3::https://s3.amazonaws.com/bucket/f.json",
		"gcs::https://www.googleapis.com/storage/v1/bucket/f.json",
		"GCS::https://www.googleapis.com/storage/v1/bucket/f.json",
		"file::/etc/passwd",
		"file::https://example.com/f.csv",
		"git::https://github.com/o/r.git",
		"git::file:///secret/repo/secret.txt",
		"git::/secret/repo/secret.txt",
		"hg::https://example.com/repo",
		"bzr::https://example.com/repo",
		"http::https://example.com/f.csv",
	}
	for _, raw := range denied {
		mustError(t, raw)
	}
}

// TestParseSourceObjectStorageHint: every s3/gcs spelling must point the
// operator at the replacement (a presigned https URL) rather than at the
// generic unsupported-source or unsupported-scheme message.
func TestParseSourceObjectStorageHint(t *testing.T) {
	for _, raw := range []string{
		"s3://bucket/key.csv",
		"gs://bucket/key.csv",
		"s3::https://s3.amazonaws.com/bucket/key.csv",
		"gcs::https://www.googleapis.com/storage/v1/bucket/key.csv",
	} {
		_, err := ParseSource(raw)
		if err == nil {
			t.Fatalf("ParseSource(%q) = nil error, want error", raw)
		}
		if !strings.Contains(err.Error(), "presigned https://") {
			t.Fatalf("ParseSource(%q) = %v, want the presigned-URL suggestion", raw, err)
		}
	}
}

func TestParseSourceSchemeURL(t *testing.T) {
	s := mustParse(t, "https://example.com/file.json")
	if s.Kind != KindHTTP || s.URL.String() != "https://example.com/file.json" {
		t.Fatalf("got %+v", s)
	}

	s = mustParse(t, "http://example.com/file.json")
	if s.Kind != KindHTTP {
		t.Fatalf("got %+v", s)
	}

	// Object-storage schemes are not transports at all — see
	// TestParseSourceObjectStorageHint for the message they must carry.
	mustError(t, "s3://bucket/key.csv")
	mustError(t, "gs://bucket/key.csv")
	mustError(t, "ftp://example.com/file.csv")
	mustError(t, "ssh://example.com/file.csv")
}

func TestParseSourceFileURL(t *testing.T) {
	s := mustParse(t, "file:///etc/passwd")
	if s.Kind != KindLocal || s.Path != "/etc/passwd" {
		t.Fatalf("got %+v", s)
	}

	s = mustParse(t, "file://localhost/etc/passwd")
	if s.Kind != KindLocal || s.Path != "/etc/passwd" {
		t.Fatalf("got %+v", s)
	}

	s = mustParse(t, "FILE://LOCALHOST/etc/passwd")
	if s.Kind != KindLocal || s.Path != "/etc/passwd" {
		t.Fatalf("got %+v", s)
	}

	// Non-localhost host: rejected.
	mustError(t, "file://evil.example.com/etc/passwd")
	// Relative path: rejected.
	mustError(t, "file://localhost")
	// Query/fragment on a file:// URL: rejected outright (never silently split).
	mustError(t, "file:///etc/passwd?/../../allowed/x")
	mustError(t, "file:///etc/passwd#frag")
	// Userinfo: rejected.
	mustError(t, "file://user@localhost/etc/passwd")
}

// TestParseSourceDocumentedForms covers every publicly documented source form
// (local path, http/https — including a presigned object-storage URL — and the
// stdin spellings).
func TestParseSourceDocumentedForms(t *testing.T) {
	cases := []struct {
		raw  string
		kind Kind
	}{
		{"/local/path.csv", KindLocal},
		{"https://example.com/file.json", KindHTTP},
		{"https://bucket.s3.amazonaws.com/f.json?X-Amz-Credential=AKIA&X-Amz-Signature=abc123", KindHTTP},
		{"stdin", KindStdin},
		{"-", KindStdin},
		{"/dev/stdin", KindStdin},
	}
	for _, c := range cases {
		s := mustParse(t, c.raw)
		if s.Kind != c.kind {
			t.Fatalf("ParseSource(%q).Kind = %v, want %v", c.raw, s.Kind, c.kind)
		}
	}
}

// --- host rewrite table ---

func TestRewriteGitHub(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://github.com/o/r/blob/main/data.csv", "https://raw.githubusercontent.com/o/r/main/data.csv"},
		{"https://github.com/o/r/raw/main/data.csv", "https://raw.githubusercontent.com/o/r/main/data.csv"},
		{"https://github.com/o/r/blob/refs/heads/main/data.csv", "https://raw.githubusercontent.com/o/r/refs/heads/main/data.csv"},
		{"https://github.com/o/r/blob/feature/my-branch/data.csv", "https://raw.githubusercontent.com/o/r/feature/my-branch/data.csv"},
		{"https://www.github.com/o/r/blob/main/data.csv", "https://raw.githubusercontent.com/o/r/main/data.csv"},
	}
	for _, c := range cases {
		s := mustParse(t, c.raw)
		if s.Kind != KindHTTP || s.URL.String() != c.want {
			t.Fatalf("ParseSource(%q) = %+v, want URL %q", c.raw, s, c.want)
		}
	}

	mustError(t, "https://github.com/o/r/tree/main/dir")
	mustError(t, "https://github.com/o/r")
	mustError(t, "https://github.com/o")
	mustError(t, "https://user:pass@github.com/o/r/blob/main/data.csv")

	// Negative host matches must NOT be rewritten (must not error either —
	// they're just generic HTTP URLs).
	for _, raw := range []string{
		"https://github.com.evil.tld/o/r/blob/main/data.csv",
		"https://evil-github.com/o/r/blob/main/data.csv",
	} {
		s := mustParse(t, raw)
		if s.Kind != KindHTTP || s.URL.String() != raw {
			t.Fatalf("ParseSource(%q) = %+v, want passthrough", raw, s)
		}
	}
}

func TestRewriteGist(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://gist.github.com/octocat/6cad326836d38bd3a7ae", "https://gist.githubusercontent.com/octocat/6cad326836d38bd3a7ae/raw"},
		{"https://gist.github.com/octocat/6cad326836d38bd3a7ae/raw", "https://gist.githubusercontent.com/octocat/6cad326836d38bd3a7ae/raw"},
		{"https://gist.github.com/octocat/6cad326836d38bd3a7ae/raw/hello_world.rb", "https://gist.githubusercontent.com/octocat/6cad326836d38bd3a7ae/raw/hello_world.rb"},
	}
	for _, c := range cases {
		s := mustParse(t, c.raw)
		if s.Kind != KindHTTP || s.URL.String() != c.want {
			t.Fatalf("ParseSource(%q) = %+v, want URL %q", c.raw, s, c.want)
		}
	}

	mustError(t, "https://gist.github.com/6cad326836d38bd3a7ae") // ownerless

	// notgist.github.com is NOT a rewrite host — passthrough, not an error.
	raw := "https://notgist.github.com/octocat/abc"
	s := mustParse(t, raw)
	if s.Kind != KindHTTP || s.URL.String() != raw {
		t.Fatalf("ParseSource(%q) = %+v, want passthrough", raw, s)
	}
}

func TestRewriteGitLab(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		// The project path may be nested in subgroups, so the "/-/" separator
		// (not a segment count) is what marks the end of the project.
		{"https://gitlab.com/group/sub/proj/-/blob/main/dir/data.csv", "https://gitlab.com/group/sub/proj/-/raw/main/dir/data.csv"},
		{"https://gitlab.com/group/proj/-/blob/main/data.csv", "https://gitlab.com/group/proj/-/raw/main/data.csv"},
		// A ref may contain slashes; the tail is passed through verbatim.
		{"https://gitlab.com/group/proj/-/blob/feature/my-branch/data.csv", "https://gitlab.com/group/proj/-/raw/feature/my-branch/data.csv"},
		{"https://www.gitlab.com/group/proj/-/blob/main/data.csv", "https://gitlab.com/group/proj/-/raw/main/data.csv"},
		// GitLab's own UI query parameters are dropped.
		{"https://gitlab.com/group/proj/-/blob/main/data.csv?ref_type=heads", "https://gitlab.com/group/proj/-/raw/main/data.csv"},
	}
	for _, c := range cases {
		s := mustParse(t, c.raw)
		if s.Kind != KindHTTP || s.URL.String() != c.want {
			t.Fatalf("ParseSource(%q) = %+v, want URL %q", c.raw, s, c.want)
		}
		if s.RewriteNote == "" {
			t.Fatalf("ParseSource(%q).RewriteNote is empty, want the rewrite reported", c.raw)
		}
	}

	// Already-raw URLs pass through unchanged, with no note (same host, same path).
	raw := "https://gitlab.com/group/sub/proj/-/raw/main/data.csv"
	s := mustParse(t, raw)
	if s.Kind != KindHTTP || s.URL.String() != raw || s.RewriteNote != "" {
		t.Fatalf("ParseSource(%q) = %+v, want passthrough", raw, s)
	}

	mustError(t, "https://gitlab.com/group/proj/-/tree/main/dir")
	mustError(t, "https://gitlab.com/group/proj")            // bare project
	mustError(t, "https://gitlab.com/group")                 // no project
	mustError(t, "https://gitlab.com/group/proj/-/issues/1") // not a file resource
	mustError(t, "https://gitlab.com/group/proj/-/blob")     // no ref or path
	mustError(t, "https://user:pass@gitlab.com/group/proj/-/blob/main/data.csv")

	// Negative host matches must NOT be rewritten (must not error either —
	// they're just generic HTTP URLs).
	for _, raw := range []string{
		"https://gitlab.com.evil.tld/group/proj/-/blob/main/data.csv",
		"https://evil-gitlab.com/group/proj/-/blob/main/data.csv",
		"https://notgitlab.com/group/proj/-/blob/main/data.csv",
	} {
		s := mustParse(t, raw)
		if s.Kind != KindHTTP || s.URL.String() != raw || s.RewriteNote != "" {
			t.Fatalf("ParseSource(%q) = %+v, want passthrough", raw, s)
		}
	}
}

func TestRewriteCodeberg(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://codeberg.org/owner/repo/src/branch/main/data.csv", "https://codeberg.org/owner/repo/raw/branch/main/data.csv"},
		{"https://codeberg.org/owner/repo/src/branch/main/dir/data.csv", "https://codeberg.org/owner/repo/raw/branch/main/dir/data.csv"},
		{"https://codeberg.org/owner/repo/src/tag/v1.2.3/data.csv", "https://codeberg.org/owner/repo/raw/tag/v1.2.3/data.csv"},
		{"https://codeberg.org/owner/repo/src/commit/0123456789abcdef/data.csv", "https://codeberg.org/owner/repo/raw/commit/0123456789abcdef/data.csv"},
		// A ref may contain slashes; the tail is passed through verbatim.
		{"https://codeberg.org/owner/repo/src/branch/feature/my-branch/data.csv", "https://codeberg.org/owner/repo/raw/branch/feature/my-branch/data.csv"},
		// Viewer query parameters are dropped.
		{"https://codeberg.org/owner/repo/src/branch/main/data.csv?display=source", "https://codeberg.org/owner/repo/raw/branch/main/data.csv"},
	}
	for _, c := range cases {
		s := mustParse(t, c.raw)
		if s.Kind != KindHTTP || s.URL.String() != c.want {
			t.Fatalf("ParseSource(%q) = %+v, want URL %q", c.raw, s, c.want)
		}
		if s.RewriteNote == "" {
			t.Fatalf("ParseSource(%q).RewriteNote is empty, want the rewrite reported", c.raw)
		}
	}

	// Already-raw URLs pass through unchanged, with no note (same host, same path).
	raw := "https://codeberg.org/owner/repo/raw/branch/main/data.csv"
	s := mustParse(t, raw)
	if s.Kind != KindHTTP || s.URL.String() != raw || s.RewriteNote != "" {
		t.Fatalf("ParseSource(%q) = %+v, want passthrough", raw, s)
	}

	mustError(t, "https://codeberg.org/owner/repo")     // bare repo
	mustError(t, "https://codeberg.org/owner/repo/src") // no selector, ref or path
	mustError(t, "https://codeberg.org/owner/repo/src/branch")
	// A branch listing with no file after the ref: its raw form is never a file.
	mustError(t, "https://codeberg.org/owner/repo/src/branch/main")
	mustError(t, "https://codeberg.org/owner/repo/src/nonsense/main/data.csv")
	mustError(t, "https://codeberg.org/owner/repo/issues")
	mustError(t, "https://user:pass@codeberg.org/owner/repo/src/branch/main/data.csv")

	// Negative host matches must NOT be rewritten (must not error either —
	// they're just generic HTTP URLs).
	for _, raw := range []string{
		"https://codeberg.org.evil.tld/owner/repo/src/branch/main/data.csv",
		"https://evil-codeberg.org/owner/repo/src/branch/main/data.csv",
	} {
		s := mustParse(t, raw)
		if s.Kind != KindHTTP || s.URL.String() != raw || s.RewriteNote != "" {
			t.Fatalf("ParseSource(%q) = %+v, want passthrough", raw, s)
		}
	}
}

func TestRewriteHuggingFace(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"hf://datasets/user/dataset/data.csv", "https://huggingface.co/datasets/user/dataset/resolve/main/data.csv"},
		{"hf://datasets/user/dataset@v2/data.csv", "https://huggingface.co/datasets/user/dataset/resolve/v2/data.csv"},
		{"hf://datasets/user/dataset/dir/nested.csv", "https://huggingface.co/datasets/user/dataset/resolve/main/dir/nested.csv"},
		{"https://huggingface.co/datasets/user/dataset/blob/main/data.csv", "https://huggingface.co/datasets/user/dataset/resolve/main/data.csv"},
	}
	for _, c := range cases {
		s := mustParse(t, c.raw)
		if s.Kind != KindHTTP || s.URL.String() != c.want {
			t.Fatalf("ParseSource(%q) = %+v, want URL %q", c.raw, s, c.want)
		}
	}

	// Already-resolved huggingface.co URLs pass through unchanged.
	raw := "https://huggingface.co/datasets/user/dataset/resolve/main/data.csv"
	s := mustParse(t, raw)
	if s.URL.String() != raw {
		t.Fatalf("ParseSource(%q) = %+v, want passthrough", raw, s)
	}

	mustError(t, "hf://spaces/user/space/app.py")
	mustError(t, "hf://models/user/model/config.json")
	mustError(t, "hf://datasets/user/dataset/*.parquet")
	mustError(t, "hf://datasets/user/dataset/dir/*.parquet")
}

func TestRewriteSheets(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://docs.google.com/spreadsheets/d/ABC123/edit", "https://docs.google.com/spreadsheets/d/ABC123/export?format=csv"},
		{"https://docs.google.com/spreadsheets/d/ABC123", "https://docs.google.com/spreadsheets/d/ABC123/export?format=csv"},
		{"https://docs.google.com/spreadsheets/d/ABC123/export?format=csv", "https://docs.google.com/spreadsheets/d/ABC123/export?format=csv"},
	}
	for _, c := range cases {
		s := mustParse(t, c.raw)
		if s.Kind != KindHTTP || s.URL.String() != c.want {
			t.Fatalf("ParseSource(%q) = %+v, want URL %q", c.raw, s, c.want)
		}
	}

	s, err := ParseSource("https://docs.google.com/spreadsheets/d/ABC123/edit#gid=123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://docs.google.com/spreadsheets/d/ABC123/export?format=csv&gid=123456"
	if s.URL.String() != want {
		t.Fatalf("got %q, want %q", s.URL.String(), want)
	}

	mustError(t, "https://docs.google.com/spreadsheets/d/ABC123/edit#gid=abc")
	mustError(t, "https://docs.google.com/document/d/ABC123/edit")
}

func TestRewriteDropbox(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://www.dropbox.com/s/abc/file.csv?dl=0", "https://www.dropbox.com/s/abc/file.csv?raw=1"},
		{"https://www.dropbox.com/s/abc/file.csv?dl=1", "https://www.dropbox.com/s/abc/file.csv?raw=1"},
		{"https://www.dropbox.com/s/abc/file.csv", "https://www.dropbox.com/s/abc/file.csv?raw=1"},
		{"https://dropbox.com/s/abc/file.csv", "https://dropbox.com/s/abc/file.csv?raw=1"},
	}
	for _, c := range cases {
		s := mustParse(t, c.raw)
		if s.Kind != KindHTTP || s.URL.String() != c.want {
			t.Fatalf("ParseSource(%q) = %+v, want URL %q", c.raw, s, c.want)
		}
	}
}

// TestParseSourceTraversalVariants: every query/fragment/percent-encoding
// traversal variant must either be a ParseSource error, or (for the local
// forms) resolve to a KindLocal Source whose Path is untouched — so a later
// containment check sees the same string a reader would open, and nothing
// upstream ever strips a query/fragment off a bare path.
func TestParseSourceTraversalVariants(t *testing.T) {
	local := []string{
		"/allowed/../../../etc/passwd?/../../../../../../../allowed/decoy",
		"/allowed/../../../etc/passwd#/../../../../../../../allowed/decoy",
		"/allowed/..%2f..%2fetc/passwd",
		"/allowed//etc/passwd",
	}
	for _, raw := range local {
		s := mustParse(t, raw)
		if s.Kind != KindLocal || s.Path != raw {
			t.Fatalf("ParseSource(%q) = %+v, want verbatim KindLocal", raw, s)
		}
	}

	// file::<path>?<traversal> is not a supported forced getter at all.
	mustError(t, "file::/allowed/../../../etc/passwd?/../../../../../../../allowed/decoy")

	// git:: / hg:: are rejected entirely, regardless of scheme wrapping.
	mustError(t, "git::https://github.com/o/r.git")
	mustError(t, "hg::https://example.com/repo")
}
