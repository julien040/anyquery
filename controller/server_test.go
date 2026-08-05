package controller

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestServerRestrictionsDevImpliesNoSandbox covers the maintainer's decision
// that `server --dev` implies `--no-sandbox`: --dev must win over the default
// sandboxed-by-default behavior, and over every explicit sandbox flag
// (including an explicit --no-sandbox=false, and the relaxation flags), so
// that Restrictions ends up nil rather than any individual permission being
// flipped on.
func TestServerRestrictionsDevImpliesNoSandbox(t *testing.T) {
	newCmd := func() *cobra.Command {
		c := &cobra.Command{Use: "server"}
		addTestSandboxFlags(c, true)
		c.Flags().Bool("dev", false, "")
		return c
	}

	t.Run("no --dev: sandboxed by default", func(t *testing.T) {
		c := newCmd()
		if serverRestrictions(c) == nil {
			t.Error("server should be sandboxed by default without --dev")
		}
	})

	t.Run("--dev alone disables the sandbox", func(t *testing.T) {
		c := newCmd()
		_ = c.Flags().Set("dev", "true")
		if r := serverRestrictions(c); r != nil {
			t.Errorf("--dev should disable the sandbox, got restrictions: %+v", r)
		}
	})

	t.Run("--dev wins over an explicit --no-sandbox=false", func(t *testing.T) {
		c := newCmd()
		_ = c.Flags().Set("dev", "true")
		_ = c.Flags().Set("no-sandbox", "false") // i.e. "please keep sandboxing on"
		if r := serverRestrictions(c); r != nil {
			t.Errorf("--dev should win over an explicit --no-sandbox=false, got restrictions: %+v", r)
		}
	})

	t.Run("--dev wins over explicit relax flags", func(t *testing.T) {
		c := newCmd()
		_ = c.Flags().Set("dev", "true")
		_ = c.Flags().Set("allow-dirs", "/srv/data")
		_ = c.Flags().Set("allow-remote", "true")
		_ = c.Flags().Set("allow-attach", "true")
		_ = c.Flags().Set("allow-db-connections", "true")
		if r := serverRestrictions(c); r != nil {
			t.Errorf("--dev should result in nil restrictions regardless of relax flags, got: %+v", r)
		}
	})

	t.Run("--no-sandbox without --dev still disables the sandbox", func(t *testing.T) {
		c := newCmd()
		_ = c.Flags().Set("no-sandbox", "true")
		if r := serverRestrictions(c); r != nil {
			t.Errorf("--no-sandbox should still disable the sandbox on its own, got: %+v", r)
		}
	})
}
