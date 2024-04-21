package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "anyq",
	Short: "A tool to query any data source",
	Long: `Anyquery is a tool that allows you to query any data source
by writing SQL queries. It is designed to be extended by anyone.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().Bool("no-input", false, "Do not launch an interactive input")
	rootCmd.Flags().BoolP("version", "v", false, "Print the version of the program")
}
