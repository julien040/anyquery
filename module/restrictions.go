package module

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
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
//     (the path lives in the virtual-table arguments), via Check (on a parsed
//     Source, see source.go) for a source that may be remote, and via
//     OpenLocal / ReadLocalFile (localfile.go) for a bare local path, which
//     enforce containment as part of the open rather than ahead of it.
type Restrictions struct {
	// AllowedDirs is the set of directories that read_* tables (and on-disk
	// ATTACH, when AllowAttach is set) may touch. Both the requested path and
	// each entry are resolved (absolute, symlinks evaluated) before the
	// containment check, so a symlink inside an allowed directory cannot escape
	// it. Empty => no local file access is permitted.
	AllowedDirs []string

	// AllowRemote permits fetching a remote Source (KindHTTP, the only remote
	// kind in source.go). Enforced by Restrictions.Check, which Fetcher calls
	// before ever dialing out.
	AllowRemote bool

	// AllowAttach permits ATTACH DATABASE / VACUUM INTO targeting on-disk paths
	// (still confined to AllowedDirs). In-memory databases are always allowed.
	AllowAttach bool

	// AllowDBConnections permits registering the database reader modules
	// (duckdb/postgres/mysql/clickhouse/cassandra), which accept arbitrary
	// connection strings and would otherwise be an SSRF (and, for DuckDB, an
	// RCE) vector.
	AllowDBConnections bool

	// dirsOnce/dirs back OpenLocal (localfile.go): AllowedDirs resolved into
	// os.Root handles, built lazily on first use and cached for the lifetime
	// of this Restrictions value. Do not copy a Restrictions after it has
	// been used — sync.Once and *os.Root do not survive a value copy.
	dirsOnce sync.Once
	dirs     []allowedDir
}

// AllowStdin reports whether reading from stdin is permitted. Only an
// unrestricted (nil) policy allows it — every reader special-cases stdin
// before it ever reaches ParseSource/Fetcher, so each one calls this
// directly rather than relying on Fetcher.Open's own stdin gate.
func (r *Restrictions) AllowStdin() bool {
	return r == nil
}

// Check validates a parsed Source against the policy: KindLocal delegates to
// OpenLocal's containment (opening and immediately closing, since Check only
// answers yes/no — a caller that will actually read the source, like
// Fetcher, calls OpenLocal directly instead of Check+Open, so there is only
// ever one open on that path); KindStdin is denied under any non-nil policy;
// KindHTTP requires AllowRemote.
func (r *Restrictions) Check(s Source) error {
	if r == nil {
		return nil
	}
	switch s.Kind {
	case KindLocal:
		f, err := r.OpenLocal(s.Path)
		if err != nil {
			return err
		}
		f.Close()
		return nil
	case KindStdin:
		return fmt.Errorf("sandbox: reading from stdin is not allowed")
	case KindHTTP:
		if r.AllowRemote {
			return nil
		}
		return fmt.Errorf("sandbox: remote fetching is disabled; %q is not a local file (enable with --allow-remote)", s.Raw)
	default:
		return fmt.Errorf("sandbox: internal error: unhandled source kind %d", s.Kind)
	}
}

// CheckFileRead answers whether a plain local file path (no scheme) is
// readable under the policy, without reading it: it opens the path through
// OpenLocal and closes it again.
//
// This is only for callers that want the yes/no answer and nothing else. A
// caller that goes on to read the file must use ReadLocalFile (or OpenLocal
// and read from the returned handle) instead — re-opening the path by name
// after this check reopens the window in which the path can be swapped for a
// symlink escaping the allowed directories, which is precisely what opening
// through OpenLocal closes.
func (r *Restrictions) CheckFileRead(path string) error {
	if r == nil {
		return nil
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("sandbox: empty file path is not allowed")
	}
	f, err := r.OpenLocal(path)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

// AllowAttachPath reports whether an ATTACH DATABASE / VACUUM INTO target is
// permitted. filename is the value the SQLite authorizer reports for
// SQLITE_ATTACH. In-memory databases are always allowed; an empty filename
// (e.g. a parameterized ATTACH at prepare time, whose value is bound later and
// never re-authorized) is denied.
//
// os.Root cannot confine ATTACH itself — SQLite opens the path on its own and
// there is no portable way to hand it a confined descriptor — so containment
// is checked here by opening the target's parent directory (which must
// already exist; ATTACH routinely creates the leaf database file itself, so
// the leaf is deliberately not required to exist). This is kernel-enforced
// for the parent chain, same as OpenLocal, but a symlink planted inside an
// allowed directory between this check and SQLite's own open of the leaf is
// a known, accepted residual risk (it requires local write access to an
// allowed directory already).
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
	return r.allowsAttachTarget(attachPathToFile(filename))
}

// allowsAttachTarget reports whether path's parent directory resolves inside
// one of AllowedDirs, without requiring path itself to exist.
func (r *Restrictions) allowsAttachTarget(path string) bool {
	target, err := filepath.Abs(path)
	if err != nil {
		target = path
	}
	resolvedTarget := resolveHint(path)

	for _, d := range r.buildAllowedDirs() {
		rel, ok := relWithin(d.given, target)
		if !ok {
			rel, ok = relWithin(d.resolved, resolvedTarget)
		}
		if !ok || rel == "." {
			continue // not contained, or names the directory itself, not a file within it
		}
		f, err := d.root.Open(filepath.Dir(rel))
		if err != nil {
			continue
		}
		f.Close()
		return true
	}
	return false
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
