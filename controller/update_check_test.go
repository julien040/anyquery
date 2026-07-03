package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/adrg/xdg"
)

func TestComputeUpdateNotice(t *testing.T) {
	cases := []struct {
		name         string
		current      string
		cachedLatest string
		skip         bool
		wantNotice   bool
	}{
		{name: "newer available", current: "0.4.5", cachedLatest: "0.4.6", wantNotice: true},
		{name: "newer available with v prefix", current: "0.4.5", cachedLatest: "v0.4.6", wantNotice: true},
		{name: "up to date", current: "0.4.5", cachedLatest: "0.4.5", wantNotice: false},
		{name: "current is newer", current: "0.5.0", cachedLatest: "0.4.6", wantNotice: false},
		{name: "skip disables check", current: "0.4.5", cachedLatest: "0.4.6", skip: true, wantNotice: false},
		{name: "empty cache", current: "0.4.5", cachedLatest: "", wantNotice: false},
		{name: "dev build", current: "dev", cachedLatest: "0.4.6", wantNotice: false},
		{name: "devel build", current: "(devel)", cachedLatest: "0.4.6", wantNotice: false},
		{name: "empty current", current: "", cachedLatest: "0.4.6", wantNotice: false},
		{name: "library module tag", current: "v0.1.6", cachedLatest: "0.4.6", wantNotice: false},
		{name: "pseudo version", current: "0.4.5-0.20240101000000-abcdef123456", cachedLatest: "0.5.0", wantNotice: false},
	}

	const hint = "UPDATE-HINT"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeUpdateNotice(tc.current, tc.cachedLatest, hint, tc.skip)
			if (got != "") != tc.wantNotice {
				t.Fatalf("computeUpdateNotice(%q, %q, %v) = %q; wantNotice=%v",
					tc.current, tc.cachedLatest, tc.skip, got, tc.wantNotice)
			}
			if tc.wantNotice && !strings.Contains(got, hint) {
				t.Fatalf("notice %q does not include the update hint", got)
			}
		})
	}
}

func TestUpdateCommandHint(t *testing.T) {
	cases := []struct {
		name        string
		exePath     string
		goos        string
		linuxPkgMgr string
		want        string // substring that must appear
	}{
		{"homebrew apple silicon", "/opt/homebrew/Cellar/anyquery/0.4.5/bin/anyquery", "darwin", "", "brew upgrade anyquery"},
		{"homebrew intel", "/usr/local/Cellar/anyquery/0.4.5/bin/anyquery", "darwin", "", "brew upgrade anyquery"},
		{"homebrew linux", "/home/linuxbrew/.linuxbrew/Cellar/anyquery/0.4.5/bin/anyquery", "linux", "apt", "brew upgrade anyquery"},
		{"scoop", `C:\Users\me\scoop\shims\anyquery.exe`, "windows", "", "scoop update anyquery"},
		{"chocolatey", `C:\ProgramData\chocolatey\bin\anyquery.exe`, "windows", "", "choco upgrade anyquery"},
		{"winget", `C:\Users\me\AppData\Local\Microsoft\WinGet\Packages\JulienCagniart.anyquery_x\anyquery.exe`, "windows", "", "winget upgrade JulienCagniart.anyquery"},
		{"arch AUR system path", "/usr/bin/anyquery", "linux", "pacman", "anyquery-git"},
		{"debian system path", "/usr/bin/anyquery", "linux", "apt", "apt upgrade anyquery"},
		{"fedora system path", "/usr/bin/anyquery", "linux", "dnf", "dnf upgrade anyquery"},
		{"linux system path unknown mgr", "/usr/bin/anyquery", "linux", "", "system package manager"},
		{"curl script /usr/local/bin linux", "/usr/local/bin/anyquery", "linux", "apt", "install.sh"},
		{"curl script ~/.local/bin linux", "/home/me/.local/bin/anyquery", "linux", "pacman", "install.sh"},
		{"curl script /usr/local/bin mac", "/usr/local/bin/anyquery", "darwin", "", "install.sh"},
		{"windows manual extract", `C:\tools\anyquery.exe`, "windows", "", "Scoop, Winget, or Chocolatey"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := updateCommandHint(tc.exePath, tc.goos, tc.linuxPkgMgr)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("updateCommandHint(%q, %q, %q) = %q; want it to contain %q",
					tc.exePath, tc.goos, tc.linuxPkgMgr, got, tc.want)
			}
		})
	}
}

// TestUpdateNoticeReadsCache exercises the cache-reading wrapper end-to-end
// (minus the network) by pointing xdg at a temp cache dir and seeding a fresh
// entry, so no background refresh fires.
func TestUpdateNoticeReadsCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("ANYQUERY_SKIP_UPDATE_CHECK", "")
	xdg.Reload()

	prev := CurrentVersion
	CurrentVersion = "0.4.5"
	t.Cleanup(func() { CurrentVersion = prev })

	// Fresh timestamp → wrapper won't spawn a network refresh.
	if err := writeUpdateCache(updateCache{LastCheck: time.Now().Unix(), LatestVersion: "9.9.9"}); err != nil {
		t.Fatalf("writeUpdateCache: %v", err)
	}

	if got := updateNotice(); got == "" {
		t.Fatalf("updateNotice() = \"\"; want a notice for 0.4.5 -> 9.9.9")
	}

	// Same version cached → no notice.
	if err := writeUpdateCache(updateCache{LastCheck: time.Now().Unix(), LatestVersion: "0.4.5"}); err != nil {
		t.Fatalf("writeUpdateCache: %v", err)
	}
	if got := updateNotice(); got != "" {
		t.Fatalf("updateNotice() = %q; want \"\" when up to date", got)
	}

	// Opt-out env var wins.
	t.Setenv("ANYQUERY_SKIP_UPDATE_CHECK", "1")
	if err := writeUpdateCache(updateCache{LastCheck: time.Now().Unix(), LatestVersion: "9.9.9"}); err != nil {
		t.Fatalf("writeUpdateCache: %v", err)
	}
	if got := updateNotice(); got != "" {
		t.Fatalf("updateNotice() = %q; want \"\" when ANYQUERY_SKIP_UPDATE_CHECK is set", got)
	}
}
