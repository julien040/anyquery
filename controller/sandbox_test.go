package controller

import (
	"testing"

	"github.com/spf13/cobra"
)

func addTestSandboxFlags(c *cobra.Command, isServer bool) {
	c.Flags().StringSlice("allow-dirs", nil, "")
	c.Flags().Bool("allow-remote", false, "")
	c.Flags().Bool("allow-attach", false, "")
	c.Flags().Bool("allow-db-connections", false, "")
	if isServer {
		c.Flags().Bool("no-sandbox", false, "")
	} else {
		c.Flags().Bool("sandbox", false, "")
	}
}

func TestRestrictionsFromFlags(t *testing.T) {
	t.Run("server style is on by default", func(t *testing.T) {
		c := &cobra.Command{Use: "x"}
		addTestSandboxFlags(c, true)
		if RestrictionsFromFlags(c) == nil {
			t.Error("server command should be sandboxed by default")
		}
	})

	t.Run("server style --no-sandbox disables", func(t *testing.T) {
		c := &cobra.Command{Use: "x"}
		addTestSandboxFlags(c, true)
		_ = c.Flags().Set("no-sandbox", "true")
		if RestrictionsFromFlags(c) != nil {
			t.Error("--no-sandbox should disable the sandbox")
		}
	})

	t.Run("cli style is off by default", func(t *testing.T) {
		c := &cobra.Command{Use: "x"}
		addTestSandboxFlags(c, false)
		if RestrictionsFromFlags(c) != nil {
			t.Error("cli command should be unrestricted by default")
		}
	})

	t.Run("cli style --sandbox enables and reads relax flags", func(t *testing.T) {
		c := &cobra.Command{Use: "x"}
		addTestSandboxFlags(c, false)
		_ = c.Flags().Set("sandbox", "true")
		_ = c.Flags().Set("allow-dirs", "/srv/data")
		_ = c.Flags().Set("allow-remote", "true")
		r := RestrictionsFromFlags(c)
		if r == nil {
			t.Fatal("--sandbox should enable the sandbox")
		}
		if len(r.AllowedDirs) != 1 || r.AllowedDirs[0] != "/srv/data" {
			t.Errorf("allow-dirs not propagated: %v", r.AllowedDirs)
		}
		if !r.AllowRemote {
			t.Error("allow-remote not propagated")
		}
		if r.AllowAttach || r.AllowDBConnections {
			t.Error("unset relax flags should remain false")
		}
	})

	t.Run("no sandbox flags means unrestricted", func(t *testing.T) {
		c := &cobra.Command{Use: "x"}
		c.Flags().String("host", "", "")
		if RestrictionsFromFlags(c) != nil {
			t.Error("a command without sandbox flags should be unrestricted")
		}
	})
}

func TestIsNetworkExposedHost(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1": false,
		"localhost": false,
		"::1":       false,
		"[::1]":     false,
		"LocalHost": false,
		"":          true,
		"0.0.0.0":   true,
		"192.168.1.5": true,
		"example.com": true,
	}
	for host, want := range cases {
		if got := isNetworkExposedHost(host); got != want {
			t.Errorf("isNetworkExposedHost(%q) = %v, want %v", host, got, want)
		}
	}
}
