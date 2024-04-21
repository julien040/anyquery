package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query [sql query]",
	Short: "Run a SQL query",
	Long: `Run a SQL query on the data sources installed on the system.
The query can be specified as an argument or read from stdin.
If no query is provided, the command will launch an interactive input.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("query called")
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().String("format", "pretty", "Output format (pretty, json, csv, plain)")
	queryCmd.Flags().Bool("json", false, "Output format as JSON")
	queryCmd.Flags().Bool("csv", false, "Output format as CSV")
	queryCmd.Flags().Bool("plain", false, "Output format as plain text")
}
