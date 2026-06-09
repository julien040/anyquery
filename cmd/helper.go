package cmd

import "github.com/spf13/cobra"

// This file defines a few functions that avoid code repetition for the commands declaration

// Set the flags for a command that modifies the configuration
func addFlag_commandModifiesConfiguration(cmd *cobra.Command) {
	cmd.Flags().StringP("config", "c", "", "Path to the configuration database")
}

// Same as addFlag_commandModifiesConfiguration but flags will be applied
// to all subcommands
func addPersistentFlag_commandModifiesConfiguration(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("config", "c", "", "Path to the configuration database")
}

// Set the flags for a command that prints data
//
// It helps specify the output format
func addFlag_commandPrintsData(cmd *cobra.Command) {
	// We don't add a default value to the format flag
	// because cmd wouldn't be able to overwrite the default format
	cmd.Flags().String("format", "", "Output format (pretty, json, csv, plain)")
	cmd.Flags().Bool("json", false, "Output format as JSON")
	cmd.Flags().Bool("csv", false, "Output format as CSV")
	cmd.Flags().Bool("plain", false, "Output format as plain text")
}

// Same as addFlag_commandPrintsData but flags will be applied
// to all subcommands
func addPersistentFlag_commandPrintsData(cmd *cobra.Command) {
	cmd.PersistentFlags().String("format", "", "Output format (pretty, json, csv, plain)")
	cmd.PersistentFlags().Bool("json", false, "Output format as JSON")
	cmd.PersistentFlags().Bool("csv", false, "Output format as CSV")
	cmd.PersistentFlags().Bool("plain", false, "Output format as plain text")
}

func addFlag_commandCanBeInteractive(cmd *cobra.Command) {
	cmd.Flags().Bool("no-input", false, "Do not ask for input")
}

// addSandboxFlags registers the sandboxing flags.
//
// For the server, sandboxing is on by default and --no-sandbox disables it; for
// CLI commands, --sandbox opts in (off by default, since local use is trusted).
// The relaxation flags are identical in both cases.
func addSandboxFlags(cmd *cobra.Command, isServer bool) {
	cmd.Flags().StringSlice("allow-dirs", []string{}, "When sandboxed, directories that read_* tables (and on-disk ATTACH) may access (repeatable, comma-separated)")
	cmd.Flags().Bool("allow-remote", false, "When sandboxed, allow read_* tables to fetch remote URLs (http/https/s3/...)")
	cmd.Flags().Bool("allow-attach", false, "When sandboxed, allow ATTACH/VACUUM INTO to on-disk paths within --allow-dirs")
	cmd.Flags().Bool("allow-db-connections", false, "When sandboxed, allow the database reader modules (duckdb/postgres/mysql/clickhouse/cassandra)")
	if isServer {
		cmd.Flags().Bool("no-sandbox", false, "Disable server sandboxing entirely (UNSAFE: exposes local file read, SSRF, and arbitrary file write)")
	} else {
		cmd.Flags().Bool("sandbox", false, "Apply server-style sandboxing restrictions (off by default in CLI mode)")
	}
}
