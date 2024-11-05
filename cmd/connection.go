package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var connectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Manage connections to other databases",
	Long: `Anyquery can connect to other databases such as MySQL, PostgreSQL, SQLite, etc.
You can add, list, and delete connections.

Each connection has a name, a type, and a connection string. You can also define a small CEL script to filter which tables to import.
The connection name will be used as the schema name in the queries. 
For example, if you have a connection named "mydb", a schema named "information_schema" and a table named "tables", you can query it with "SELECT * FROM mydb.information_schema_tables".
`,
	Aliases: []string{"conn", "connections"},
	RunE:    controller.ConnectionList,
	Example: `# List the connections
anyquery connection list

# Add a connection
anyquery connection add mydb mysql mysql://user:password@localhost:3306/dbname "table.schema == 'public'"

# Remove a connection
anyquery connection remove mydb
`,
}

var connectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the connections",
	RunE:  controller.ConnectionList,
}

var connectionAddCmd = &cobra.Command{
	Use:     "add <name> <type> <connection_string> [filter]",
	Short:   "Add a connection",
	Aliases: []string{"create", "new"},
	RunE:    controller.ConnectionAdd,
}

var connectionRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Short:   "Remove a connection",
	Aliases: []string{"rm", "delete"},
	RunE:    controller.ConnectionRemove,
}

func init() {
	rootCmd.AddCommand(connectionCmd)
	connectionCmd.AddCommand(connectionListCmd)
	connectionCmd.AddCommand(connectionAddCmd)
	connectionCmd.AddCommand(connectionRemoveCmd)

	addFlag_commandPrintsData(connectionCmd)
	addPersistentFlag_commandModifiesConfiguration(connectionAddCmd)
	addFlag_commandPrintsData(connectionListCmd)
	addPersistentFlag_commandModifiesConfiguration(connectionRemoveCmd)

}
