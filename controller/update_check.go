package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/adrg/xdg"
)

// CurrentVersion holds the raw CLI version (e.g. "0.4.5"). It is set from main
// through cmd.Execute. It is empty for `go build` (dev) builds and holds the Go
// module tag (e.g. "v0.1.6") for `go install` builds — both of which disable the
// update check, see computeUpdateNotice.
var CurrentVersion string

const (
	updateCheckRepo    = "julien040/anyquery"
	updateCheckTTL     = 24 * time.Hour
	updateCacheRelPath = "anyquery/update_check.json"
)

// bareSemverRe matches an official goreleaser CLI version such as "0.4.5".
//
// The repository carries two tag families in the same repo: bare "0.x.y" tags
// are the CLI releases, while "v0.1.x" tags are the Go *library* module. Only
// goreleaser CLI builds report a bare version, so requiring this format cleanly
// excludes library tags ("v0.1.6"), "dev", "(devel)", and Go pseudo-versions —
// none of which can be meaningfully compared against CLI releases.
var bareSemverRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

type updateCache struct {
	LastCheck     int64  `json:"last_check"`
	LatestVersion string `json:"latest_version"`
}

// computeUpdateNotice is the pure core of the update check. Given the running
// version, the latest version last seen on disk, the update instruction to
// append, and whether checking is disabled, it returns the message to print, or
// "" when there is nothing to say. It performs no I/O, which is what makes the
// update logic unit-testable.
func computeUpdateNotice(current, cachedLatest, updateHint string, skip bool) string {
	if skip {
		return ""
	}
	if !bareSemverRe.MatchString(current) {
		return ""
	}
	if cachedLatest == "" {
		return ""
	}

	cur, err := semver.NewVersion(current)
	if err != nil {
		return ""
	}
	latest, err := semver.NewVersion(strings.TrimPrefix(cachedLatest, "v"))
	if err != nil {
		return ""
	}
	if !latest.GreaterThan(cur) {
		return ""
	}

	return fmt.Sprintf("A new version of anyquery is available: %s → %s\n%s",
		current, latest.String(), updateHint)
}

// updateCommandHint returns the update instruction that matches how anyquery was
// installed, inferred from the executable path and the OS. It is pure (no I/O):
// the caller resolves symlinks and passes the real path plus, for Linux system
// installs, the detected package manager ("pacman", "apt", "dnf", or ""). Path
// separators are normalized so the matching is independent of the OS running the
// code.
func updateCommandHint(exePath, goos, linuxPkgMgr string) string {
	p := strings.ReplaceAll(strings.ToLower(exePath), "\\", "/")

	switch {
	case strings.Contains(p, "/scoop/"):
		return "Run `scoop update anyquery` to update."
	case strings.Contains(p, "/chocolatey/"):
		return "Run `choco upgrade anyquery` to update."
	case strings.Contains(p, "microsoft/winget") || strings.Contains(p, "/winget/"):
		return "Run `winget upgrade JulienCagniart.anyquery` to update."
	case strings.Contains(p, "/cellar/") || strings.Contains(p, "/homebrew/") || strings.Contains(p, "linuxbrew"):
		return "Run `brew upgrade anyquery` to update."
	case goos == "linux" && (strings.HasPrefix(p, "/usr/bin/") || strings.HasPrefix(p, "/bin/")):
		// A native package installed into a system path. The distro's package
		// manager (detected by the caller) decides the exact command.
		switch linuxPkgMgr {
		case "pacman":
			return "Update it via the AUR, e.g. `yay -S anyquery-git` (or `paru -S anyquery-git`)."
		case "apt":
			return "Run `sudo apt update && sudo apt upgrade anyquery` to update."
		case "dnf":
			return "Run `sudo dnf upgrade anyquery` to update."
		default:
			return "Update it with your system package manager (apt, dnf, pacman/AUR, …)."
		}
	case goos == "windows":
		return "Update it with Scoop, Winget, or Chocolatey — see https://anyquery.dev/docs/#installation"
	default:
		return "Re-run the install script (https://anyquery.dev/install.sh) or use your package manager to update."
	}
}

// detectLinuxPkgManager reports the system package manager on Linux ("pacman",
// "apt", "dnf", or "" when unknown/non-Linux). pacman is checked first so Arch
// is not mistaken for another distro. Impure: it probes the PATH.
func detectLinuxPkgManager() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	for _, m := range []struct{ bin, name string }{
		{"pacman", "pacman"},
		{"apt-get", "apt"},
		{"dnf", "dnf"},
		{"yum", "dnf"},
	} {
		if _, err := exec.LookPath(m.bin); err == nil {
			return m.name
		}
	}
	return ""
}

// resolvedExecutable returns the path to the running binary with symlinks
// resolved, so a Homebrew/Scoop shim points at its real install location.
func resolvedExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

// updateNotice reads the cached latest version, refreshes it in the background
// when stale, and returns the notice computed from the cache. It never blocks on
// the network (the refresh runs in a goroutine, safe because the update check
// only fires from the long-lived interactive shell) and never returns an error —
// any failure simply yields no notice.
func updateNotice() string {
	skip := os.Getenv("ANYQUERY_SKIP_UPDATE_CHECK") != ""
	if skip || !bareSemverRe.MatchString(CurrentVersion) {
		// Nothing to show and nothing worth refreshing for dev/library builds.
		return ""
	}

	cache, _ := readUpdateCache()

	if cache.LatestVersion == "" || time.Now().Unix()-cache.LastCheck > int64(updateCheckTTL.Seconds()) {
		go refreshUpdateCache()
	}

	hint := updateCommandHint(resolvedExecutable(), runtime.GOOS, detectLinuxPkgManager())
	return computeUpdateNotice(CurrentVersion, cache.LatestVersion, hint, skip)
}

// refreshUpdateCache fetches the latest CLI version and rewrites the cache.
// Best-effort: all failures are ignored.
func refreshUpdateCache() {
	latest, err := fetchLatestVersion()
	if err != nil || latest == "" {
		return
	}
	_ = writeUpdateCache(updateCache{
		LastCheck:     time.Now().Unix(),
		LatestVersion: latest,
	})
}

// fetchLatestVersion returns the latest CLI release version (bare, e.g. "0.4.5").
// GitHub's "latest release" is always a CLI release because the library "v0.1.x"
// tags are git tags only, not published GitHub Releases.
func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet,
		"https://api.github.com/repos/"+updateCheckRepo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "anyquery-cli")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned status %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	return strings.TrimPrefix(strings.TrimSpace(payload.TagName), "v"), nil
}

func updateCachePath() (string, error) {
	return xdg.CacheFile(updateCacheRelPath)
}

func readUpdateCache() (updateCache, error) {
	var c updateCache
	path, err := updateCachePath()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

func writeUpdateCache(c updateCache) error {
	path, err := updateCachePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
