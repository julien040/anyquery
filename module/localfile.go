package module

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// allowedDir is one entry of Restrictions.AllowedDirs, resolved once and
// backed by an os.Root so that containment is enforced by the kernel
// (openat2 + RESOLVE_BENEATH on Linux), not by string comparison.
type allowedDir struct {
	given    string   // filepath.Abs of the configured entry, unresolved
	resolved string   // given with symlinks evaluated
	root     *os.Root // opened at resolved
}

// buildAllowedDirs lazily opens an os.Root for every entry of r.AllowedDirs.
// An entry that fails to resolve or open (e.g. it doesn't exist) is skipped
// and logged rather than failing the whole policy — the remaining entries
// still work, and containment against a skipped entry always denies.
func (r *Restrictions) buildAllowedDirs() []allowedDir {
	r.dirsOnce.Do(func() {
		for _, given := range r.AllowedDirs {
			given = strings.TrimSpace(given)
			if given == "" {
				continue
			}
			abs, err := filepath.Abs(given)
			if err != nil {
				hclog.Default().Warn("sandbox: skipping allowed directory", "dir", given, "error", err)
				continue
			}
			resolved, err := filepath.EvalSymlinks(abs)
			if err != nil {
				hclog.Default().Warn("sandbox: skipping allowed directory", "dir", given, "error", err)
				continue
			}
			root, err := os.OpenRoot(resolved)
			if err != nil {
				hclog.Default().Warn("sandbox: skipping allowed directory", "dir", given, "error", err)
				continue
			}
			r.dirs = append(r.dirs, allowedDir{given: abs, resolved: resolved, root: root})
		}
	})
	return r.dirs
}

// relWithin reports the path of target relative to base, when target is base
// itself or nested inside it. This is only ever used as a hint for which
// os.Root to try — see the comment on OpenLocal.
func relWithin(base, target string) (string, bool) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false // e.g. different Windows volumes
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return rel, true
}

// resolveHint returns an absolute, symlink-resolved form of p, purely as a
// second candidate for relWithin — so a target reached through a symlinked
// ancestor (e.g. macOS /var -> /private/var) still matches an allowed dir's
// resolved form, even though the unresolved comparison would not line up.
// When p does not exist yet, the longest existing ancestor is resolved and
// the remainder re-appended, so a not-yet-created file is still comparable.
func resolveHint(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	dir := abs
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
	return abs
}

// OpenLocal opens p confined to r.AllowedDirs. r == nil means unrestricted.
//
// The string comparison in relWithin is only a hint for which root to try;
// os.Root is the actual enforcement, so a symlink planted inside an allowed
// directory (or a parent directory that is itself a symlink, e.g. macOS
// /var -> /private/var) cannot be used to escape it, and there is no window
// between checking a path and opening it — the same call does both.
//
// When p falls within an allowed directory but the open itself fails (file
// missing, symlink escape detected by the kernel, permission denied, …),
// that raw error is returned as-is, not reworded as a sandbox denial — only
// "no allowed directory contains p at all" produces the
// "sandbox: access to %q is not allowed" error that callers may match on.
//
// That split is a contract callers rely on: a refusal by the policy is the
// only error whose message starts with "sandbox: ", every other failure is
// the unwrapped OS error. controller/shell.go's `.read` distinguishes the two
// by that prefix, echoing a refusal (which reveals nothing about the
// filesystem) while hiding OS errors (which are an existence/readability
// oracle), so keep new error strings on that side of the line.
func (r *Restrictions) OpenLocal(p string) (*os.File, error) {
	if r == nil {
		return os.Open(p)
	}
	if strings.TrimSpace(p) == "" {
		return nil, fmt.Errorf("sandbox: empty file path is not allowed")
	}
	target, err := filepath.Abs(p)
	if err != nil {
		target = p
	}
	resolvedTarget := resolveHint(p)

	var lastErr error
	for _, d := range r.buildAllowedDirs() {
		rel, ok := relWithin(d.given, target)
		if !ok {
			rel, ok = relWithin(d.resolved, resolvedTarget)
		}
		if !ok {
			continue
		}
		f, err := d.root.Open(rel)
		if err == nil {
			return f, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("sandbox: access to %q is not allowed; permitted directories: %v", p, r.AllowedDirs)
}

// ReadLocalFile reads the whole of p confined to r.AllowedDirs. r == nil means
// unrestricted.
//
// The bytes are read from the very handle OpenLocal returned, which is what
// makes the containment decision and the use of the file one and the same
// open: nothing can replace p with a symlink pointing outside the allowed
// directories in between, because there is no in between. A caller that
// instead validates p and then re-opens it by name reintroduces exactly that
// window, so anything that reads a file should call this (or read from
// OpenLocal's handle directly when streaming) rather than checking first.
//
// Errors come straight from OpenLocal, keeping its distinction between a
// policy refusal ("sandbox: …") and an ordinary open failure (the raw OS
// error).
func (r *Restrictions) ReadLocalFile(p string) ([]byte, error) {
	if r == nil {
		return os.ReadFile(p)
	}
	f, err := r.OpenLocal(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
