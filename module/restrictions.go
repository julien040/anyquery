package module

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Restrictions is the sandboxing policy enforced in Server Mode (and optionally
// in CLI mode via --sandbox).
//
// A nil *Restrictions means "no restrictions" — the default for local CLI use,
// where the operator is trusted. A non-nil value enforces the policy; its zero
// value is maximally restrictive (no readable directories, no remote fetches,
// no on-disk ATTACH, and the database reader modules disabled). Every method is
// safe to call on a nil receiver.
//
// The policy is enforced in two layers that share this object:
//   - the SQLite authorizer (namespace package) gates ATTACH / VACUUM INTO,
//     which it can see the path of, via AllowAttachPath;
//   - the read_* modules gate file/URL access, which the authorizer cannot see
//     (the path lives in the virtual-table arguments), via CheckSource /
//     CheckFileRead.
type Restrictions struct {
	// AllowedDirs is the set of directories that read_* tables (and on-disk
	// ATTACH, when AllowAttach is set) may touch. Both the requested path and
	// each entry are resolved (absolute, symlinks evaluated) before the
	// containment check, so a symlink inside an allowed directory cannot escape
	// it. Empty => no local file access is permitted.
	AllowedDirs []string

	// AllowRemote permits non-file getters (http/https/s3/gcs/git/...). When
	// false, downloadFile restricts go-getter to the local file getter, so no
	// remote transport is reachable.
	AllowRemote bool

	// AllowAttach permits ATTACH DATABASE / VACUUM INTO targeting on-disk paths
	// (still confined to AllowedDirs). In-memory databases are always allowed.
	AllowAttach bool

	// AllowDBConnections permits registering the database reader modules
	// (duckdb/postgres/mysql/clickhouse/cassandra), which accept arbitrary
	// connection strings and would otherwise be an SSRF (and, for DuckDB, an
	// RCE) vector.
	AllowDBConnections bool
}

// forcedGetterRe matches go-getter's "type::url" forced-getter prefix.
var forcedGetterRe = regexp.MustCompile(`^([A-Za-z0-9]+)::(.+)$`)

// isRemoteSource reports whether src would be fetched over a non-file transport.
//
// This is the early, friendly gate (a clear error and an independent check);
// the getter allowlist in downloadFile is the authoritative SSRF control, so
// this does not need to replicate go-getter's full detection logic. A bare path
// (including a Windows drive path like C:\data) is treated as local because it
// has no "scheme://" component.
func isRemoteSource(src string) bool {
	if m := forcedGetterRe.FindStringSubmatch(src); m != nil {
		return !strings.EqualFold(m[1], "file")
	}
	if i := strings.Index(src, "://"); i > 0 {
		return !strings.EqualFold(src[:i], "file")
	}
	return false
}

// stripFileScheme resolves a reader source to the filesystem path go-getter
// will open. It removes a "file::" forced-getter prefix, then — because
// net/url (which go-getter uses) lowercases the scheme and reads u.Path — parses
// any "file:" URL case-insensitively so file:/p, file://host/p, file:///p, and
// their upper/mixed-case spellings all resolve to the real path being checked.
//
// A naive TrimPrefix here (the previous behavior) would validate file:/etc/passwd
// as the relative path "file:/etc/passwd" under the working directory while
// go-getter opened the absolute /etc/passwd, escaping the sandbox.
//
// The Opaque branch and the parse-failure fallthrough are deny-safe: go-getter
// reads only u.Path, so an opaque input (empty u.Path) fails there too.
func stripFileScheme(src string) string {
	if m := forcedGetterRe.FindStringSubmatch(src); m != nil && strings.EqualFold(m[1], "file") {
		src = m[2]
	}
	if len(src) >= len("file:") && strings.EqualFold(src[:len("file:")], "file:") {
		if u, err := url.Parse(src); err == nil {
			if u.Opaque != "" {
				return u.Opaque
			}
			return u.Path
		}
	}
	return src
}

// CheckSource validates a reader source (file path or URL) against the policy.
// It must be called on the original src, before go-getter copies it into the
// cache directory (which would always pass the path check).
func (r *Restrictions) CheckSource(src string) error {
	if r == nil {
		return nil // unrestricted
	}
	if strings.TrimSpace(src) == "" {
		return fmt.Errorf("sandbox: empty source is not allowed")
	}
	// A forced "file::" getter whose inner target is itself a URL (any scheme)
	// or another forced getter is never a plain filesystem path: once "file"
	// wins as the getter, go-getter's FileGetter opens the inner target's
	// u.Path and silently discards its scheme and host — regardless of
	// AllowRemote, because a forced getter always overrides scheme-based
	// dispatch (go-getter client.go: force, src := getForcedGetter(src); if
	// force == "" { force = u.Scheme }, so an already-forced "file" never
	// yields to the inner scheme). This check must run before, and
	// independently of, the isRemoteSource/AllowRemote branch below: if this
	// shape were instead classified as "remote", AllowRemote:true would return
	// nil here unconditionally while the transport that actually runs is still
	// the local FileGetter — turning --allow-remote into unrestricted local
	// disk read. Rather than re-deriving the exact path go-getter resolves to
	// (a validator-vs-consumer divergence that has already produced two prior
	// bugs in this file), refuse the ambiguous shape outright.
	if m := forcedGetterRe.FindStringSubmatch(src); m != nil && strings.EqualFold(m[1], "file") {
		if looksLikeURLOrForcedGetter(m[2]) {
			return fmt.Errorf("sandbox: %q is not a supported source; a file:: forced getter may not wrap another URL or forced getter", src)
		}
	}
	if isRemoteSource(src) {
		if r.AllowRemote {
			return nil
		}
		return fmt.Errorf("sandbox: remote fetching is disabled; %q is not a local file (enable with --allow-remote)", src)
	}
	return r.checkLocalPath(stripFileScheme(src))
}

// looksLikeURLOrForcedGetter reports whether s is itself a "scheme://…" URL or
// another "type::…" forced-getter prefix, i.e. not a plain filesystem path.
// Used to detect the file::<url> shape above; deliberately scheme-agnostic —
// the escape is not specific to http, it is any inner scheme, since FileGetter
// discards whatever scheme and host the inner URL carries.
func looksLikeURLOrForcedGetter(s string) bool {
	if forcedGetterRe.MatchString(s) {
		return true
	}
	return strings.Index(s, "://") > 0
}

// CheckFileRead validates a plain local file path (no scheme) against the
// policy. Used for read paths that bypass the go-getter chokepoint, such as the
// log reader's custom grok pattern file.
func (r *Restrictions) CheckFileRead(path string) error {
	if r == nil {
		return nil
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("sandbox: empty file path is not allowed")
	}
	return r.checkLocalPath(path)
}

// AllowAttachPath reports whether an ATTACH DATABASE / VACUUM INTO target is
// permitted. filename is the value the SQLite authorizer reports for
// SQLITE_ATTACH. In-memory databases are always allowed; an empty filename
// (e.g. a parameterized ATTACH at prepare time, whose value is bound later and
// never re-authorized) is denied.
func (r *Restrictions) AllowAttachPath(filename string) bool {
	if r == nil {
		return true // unrestricted
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return false
	}
	if isInMemoryDB(filename) {
		return true
	}
	if !r.AllowAttach {
		return false
	}
	return r.checkLocalPath(attachPathToFile(filename)) == nil
}

// isInMemoryDB reports whether an ATTACH target refers to an in-memory database.
// The mode is read from the parsed file: URI query, not matched as a substring,
// so a path like file:/etc/cron.d/pwn?x=mode=memory is not mistaken for memory.
//
// A repeated mode key is treated as a deny-safe anomaly rather than resolved:
// net/url's Query().Get returns the FIRST value of a repeated key, while
// SQLite's own URI parser resolves the LAST one (verified empirically —
// file:x?mode=memory&mode=rwc opens rwc on disk even though Get("mode") reports
// "memory"). Re-implementing SQLite's resolution order here would be a second
// place that has to track it and could silently drift; instead, any duplicate
// mode key is denied outright — no legitimate caller constructs one.
func isInMemoryDB(name string) bool {
	if name == ":memory:" {
		return true
	}
	if strings.HasPrefix(name, "file:") {
		if u, err := url.Parse(name); err == nil {
			modes := u.Query()["mode"]
			if len(modes) == 1 && strings.EqualFold(modes[0], "memory") {
				return true
			}
		}
	}
	return false
}

// attachPathToFile extracts the filesystem path from an ATTACH target, handling
// the SQLite file: URI form.
func attachPathToFile(name string) string {
	if strings.HasPrefix(name, "file:") {
		if u, err := url.Parse(name); err == nil {
			if u.Opaque != "" {
				return u.Opaque
			}
			return u.Path
		}
	}
	return name
}

// checkLocalPath confirms that path resolves inside one of AllowedDirs. Both
// the target and each allowed directory are canonicalized the same way so a
// symlink cannot be used to escape, and so platforms where a parent is itself a
// symlink (e.g. macOS /var -> /private/var) compare consistently.
func (r *Restrictions) checkLocalPath(path string) error {
	target := resolvePath(path)
	for _, dir := range r.AllowedDirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		if pathWithin(resolvePath(dir), target) {
			return nil
		}
	}
	return fmt.Errorf("sandbox: access to %q is not allowed; permitted directories: %v", path, r.AllowedDirs)
}

// resolvePath returns an absolute, symlink-resolved form of p. When p itself
// does not exist yet (a file about to be created/read), it resolves the longest
// existing ancestor — which also resolves any symlink in the existing portion —
// and re-appends the remainder, so containment checks are not defeated by a
// symlinked parent or fooled into mismatching by an unresolved leaf.
func resolvePath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	p = filepath.Clean(p)
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	dir := p
	var rest []string
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached the volume root
		}
		rest = append([]string{filepath.Base(dir)}, rest...)
		if resolved, err := filepath.EvalSymlinks(parent); err == nil {
			return filepath.Join(append([]string{resolved}, rest...)...)
		}
		dir = parent
	}
	return p
}

// pathWithin reports whether target is base itself or nested inside base.
func pathWithin(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false // e.g. different Windows volumes
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
