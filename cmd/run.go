package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [query_id | http_url | local_path | s3_url]",
	Args:  cobra.MinimumNArgs(1),
	Short: "Run a SQL query from the community repository",
	Long: `Run a SQL query from the community repository.
The query can be specified by its ID or by its URL.
If the query is specified by its ID, the query will be downloaded from the repository github.com/julien040/anyquery/tree/queries.
If your query isn't from the repository, you can use the URL to specify the query.`,
	RunE: controller.Run,
	Example: `# Run a query by its ID
anyquery run github_stars_per_day

# Run a query by its URL
anyquery run https://raw.githubusercontent.com/julien040/anyquery/main/queries/github_stars_per_day.sql`,
}

func init() {
	rootCmd.AddCommand(runCmd)
	addFlag_commandModifiesConfiguration(runCmd)
	addFlag_commandPrintsData(runCmd)
	runCmd.Flags().StringP("database", "d", "", "Database to connect to (a path or :memory:)")
	runCmd.Flags().Bool("in-memory", false, "Use an in-memory database")
	runCmd.Flags().Bool("readonly", false, "Start the server in read-only mode")
	runCmd.Flags().Bool("read-only", false, "Start the server in read-only mode")
}
