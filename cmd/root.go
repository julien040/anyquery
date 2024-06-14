package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "anyquery",
	Short: "A tool to query any data source",
	Long: `Anyquery allows you to query any data source
by writing SQL queries. It can be extended with plugins`,
	// Avoid writing help when an error occurs
	// Thanks https://github.com/spf13/cobra/issues/340#issuecomment-243790200
	SilenceUsage: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().Bool("no-input", false, "Do not launch an interactive input")
	rootCmd.Flags().BoolP("version", "v", false, "Print the version of the program")
}
