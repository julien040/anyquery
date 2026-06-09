package controller

import (
	"strings"

	"github.com/julien040/anyquery/module"
	"github.com/spf13/cobra"
)

// isNetworkExposedHost reports whether binding to host exposes the server beyond
// the local loopback interface. An empty host is treated as exposed (it usually
// means "all interfaces").
func isNetworkExposedHost(host string) bool {
	switch strings.TrimSpace(strings.ToLower(host)) {
	case "127.0.0.1", "localhost", "::1", "[::1]":
		return false
	default:
		return true
	}
}

// RestrictionsFromFlags builds the sandboxing policy from a command's flags.
//
// It returns nil (no restrictions) unless sandboxing is active:
//   - server commands register a "no-sandbox" flag and are sandboxed by
//     default (active unless --no-sandbox is passed);
//   - CLI commands register a "sandbox" flag and are unrestricted by default
//     (active only when --sandbox is passed).
//
// A command that registers neither flag is always unrestricted, so calling
// this from a shared namespace builder is safe regardless of the command.
func RestrictionsFromFlags(cmd *cobra.Command) *module.Restrictions {
	active := false
	switch {
	case cmd.Flags().Lookup("no-sandbox") != nil:
		noSandbox, _ := cmd.Flags().GetBool("no-sandbox")
		active = !noSandbox
	case cmd.Flags().Lookup("sandbox") != nil:
		active, _ = cmd.Flags().GetBool("sandbox")
	}
	if !active {
		return nil
	}

	allowDirs, _ := cmd.Flags().GetStringSlice("allow-dirs")
	allowRemote, _ := cmd.Flags().GetBool("allow-remote")
	allowAttach, _ := cmd.Flags().GetBool("allow-attach")
	allowDB, _ := cmd.Flags().GetBool("allow-db-connections")

	return &module.Restrictions{
		AllowedDirs:        allowDirs,
		AllowRemote:        allowRemote,
		AllowAttach:        allowAttach,
		AllowDBConnections: allowDB,
	}
}
