package cmd

import (
	"os"

	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "anyquery [database] [query]",
	Short: "A tool to query any data source",
	Long: `Anyquery allows you to query any data source
by writing SQL queries. It can be extended with plugins`,
	// Avoid writing help when an error occurs
	// Thanks https://github.com/spf13/cobra/issues/340#issuecomment-243790200
	SilenceUsage: true,
	RunE:         controller.Query,
	Example: `-- Run a one-off query
anyquery -d mydatabase.db -q "SELECT * FROM mytable"

-- Open the interactive shell
anyquery -d mydatabase.db

-- Open a database in memory
anyquery -d ":memory:"
`,
}

func Execute(version string) {
	rootCmd.Version = version
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().Bool("no-input", false, "Do not launch an interactive input")
	addFlag_commandModifiesConfiguration(rootCmd)
	addFlag_commandPrintsData(rootCmd)
	rootCmd.Flags().StringP("database", "d", "", "Database to connect to (a path or :memory:)")
	rootCmd.Flags().Bool("in-memory", false, "Use an in-memory database")
	rootCmd.Flags().Bool("readonly", false, "Start the server in read-only mode")
	rootCmd.Flags().Bool("read-only", false, "Start the server in read-only mode")
	rootCmd.Flags().StringArray("init", []string{}, "Run SQL commands in a file before the query. You can specify multiple files.")
	rootCmd.Flags().Bool("dev", false, "Run the program in developer mode")
	rootCmd.Flags().StringSlice("extension", []string{}, "Load one or more extensions by specifying their path. Separate multiple extensions with a comma.")

	// Query flags
	rootCmd.Flags().StringP("query", "q", "", "Query to run")

	// Log flags
	rootCmd.Flags().String("log-file", "", "Log file")
	rootCmd.Flags().String("log-level", "info", "Log level (trace, debug, info, warn, error, off)")
	rootCmd.Flags().String("log-format", "text", "Log format (text, json)")

	// Alternative language flags
	rootCmd.Flags().String("language", "", "Alternative language to use")
	rootCmd.Flags().Bool("prql", false, "Use the PRQL language")
	rootCmd.Flags().Bool("pql", false, "Use the PQL language")
}
