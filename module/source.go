package module

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Kind identifies how a Source is fetched. See ParseSource.
type Kind uint8

const (
	KindLocal Kind = iota
	KindStdin
	KindHTTP // http + https, after host rewriting (source_rewrite.go)
)

// Source is the result of parsing a reader argument. It is the only value
// that either the sandbox policy or the fetcher ever look at — neither one
// re-derives anything from Raw.
type Source struct {
	Kind Kind
	Path string   // KindLocal only: the exact string that will be opened, verbatim
	URL  *url.URL // KindHTTP only; query preserved verbatim (presigned URLs carry their signature there)
	Raw  string   // original input, for error messages only

	// RewriteNote is set when a host rewrite (source_rewrite.go) fired, "original → rewritten",
	// so a cross-host fetch is never invisible in errors or verbose logging.
	RewriteNote string
}

var forcedGetterSourceRe = regexp.MustCompile(`^([A-Za-z0-9]+)::(.*)$`)

// ParseSource parses a reader source string into a Source. This is the only
// place a reader source is ever interpreted: the grammar is closed, and
// anything that does not match one of its productions is an error rather than
// a fallback to some other interpretation.
//
//	source     := stdin | scheme_url | local
//	stdin      := "stdin" | "-" | "/dev/stdin"
//	scheme_url := ("http" | "https" | "hf") "://" …
//	            | "file://" abs_path
//	local      := anything else                  ← OPAQUE, never URL-parsed
//
// There is no forced-getter production: any "<prefix>::…" input is an error,
// whatever the prefix.
//
// KindLocal.Path is never split on "?" or "#", never passed through
// url.Parse, and never filepath.Clean'd here: a local path is opaque. That is
// what makes a query/fragment on a local path inert rather than merely
// checked-for — there is no stage left that could strip it off.
func ParseSource(raw string) (Source, error) {
	if strings.TrimSpace(raw) == "" {
		return Source{}, fmt.Errorf("source: empty source is not allowed")
	}

	switch raw {
	case "stdin", "-", "/dev/stdin":
		return Source{Kind: KindStdin, Raw: raw}, nil
	}

	// No forced getter is accepted, whatever the prefix. Matching them here
	// rather than letting them fall through is what stops "xxx::y" from being
	// opened as a local file literally named "xxx::y": git::/hg:: would read
	// any local repository, and file:: (or s3::) wrapping hides the inner
	// scheme from every later check.
	if m := forcedGetterSourceRe.FindStringSubmatch(raw); m != nil {
		prefix := m[1]
		if strings.EqualFold(prefix, "s3") || strings.EqualFold(prefix, "gcs") {
			return Source{}, errObjectStorageSource(raw)
		}
		return Source{}, fmt.Errorf("source: %q is not a supported source; forced getters are not allowed", raw)
	}

	if i := strings.Index(raw, "://"); i > 0 && isValidScheme(raw[:i]) {
		scheme := strings.ToLower(raw[:i])
		u, err := url.Parse(raw)
		if err != nil {
			return Source{}, fmt.Errorf("source: %q is not a valid URL: %w", raw, err)
		}
		switch scheme {
		case "http", "https":
			return rewriteAndClassify(raw, u)
		case "s3", "gs":
			// Matched explicitly so these get the presigned-URL hint instead
			// of default's generic unsupported-scheme message.
			return Source{}, errObjectStorageSource(raw)
		case "hf":
			rewritten, err := rewriteHuggingFace(raw, u)
			if err != nil {
				return Source{}, err
			}
			return rewriteAndClassify(raw, rewritten)
		case "file":
			return parseFileURL(raw, u)
		default:
			return Source{}, fmt.Errorf("source: %q uses an unsupported scheme %q", raw, scheme)
		}
	}

	return Source{Kind: KindLocal, Path: raw, Raw: raw}, nil
}

// isValidScheme reports whether s could be a URL scheme, so that a bare local
// path containing "://" purely by coincidence (which cannot happen on any
// real filesystem, but this keeps the check honest) is never misrouted.
func isValidScheme(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.'):
		default:
			return false
		}
	}
	return true
}

// errObjectStorageSource rejects every s3/gcs spelling ("s3://", "gs://",
// "s3::…", "gcs::…"). There is no object-storage transport: a presigned HTTPS
// URL reaches the same object over the ordinary KindHTTP path, and its
// signature query parameters are kept out of cache keys and error messages by
// redactedURL/redactSourceError (fetch.go).
func errObjectStorageSource(raw string) error {
	return fmt.Errorf("source: %q is not supported: object-storage sources were removed; use a presigned https:// URL instead", raw)
}

// parseFileURL implements the "file://" scheme_url production: an absolute
// path, empty or localhost host, and no query or fragment.
func parseFileURL(raw string, u *url.URL) (Source, error) {
	if u.User != nil {
		return Source{}, fmt.Errorf("source: %q may not contain userinfo", raw)
	}
	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return Source{}, fmt.Errorf("source: %q must not specify a host", raw)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return Source{}, fmt.Errorf("source: %q must not contain a query or fragment", raw)
	}
	if u.Path == "" || !strings.HasPrefix(u.Path, "/") {
		return Source{}, fmt.Errorf("source: %q must be an absolute path", raw)
	}
	return Source{Kind: KindLocal, Path: u.Path, Raw: raw}, nil
}
