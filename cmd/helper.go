package cmd

import "github.com/spf13/cobra"

// This file defines a few functions that avoid code repetition for the commands declaration

func commandModifiesConfiguration(cmd *cobra.Command) {
	cmd.Flags().StringP("config", "c", "", "Path to the configuration database")
}
