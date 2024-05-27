package cmd

import "github.com/spf13/cobra"

// This file defines a few functions that avoid code repetition for the commands declaration

func addFlag_commandModifiesConfiguration(cmd *cobra.Command) {
	cmd.Flags().StringP("config", "c", "", "Path to the configuration database")
}

// Set the flags for a command that prints data
//
// It helps specify the output format
func addFlag_commandPrintsData(cmd *cobra.Command) {
	cmd.Flags().String("format", "pretty", "Output format (pretty, json, csv, plain)")
	cmd.Flags().Bool("json", false, "Output format as JSON")
	cmd.Flags().Bool("csv", false, "Output format as CSV")
	cmd.Flags().Bool("plain", false, "Output format as plain text")
}
