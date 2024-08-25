package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage the aliases",
	Long: `Manage the aliases.
They help you use another name for a table so that you don't have to write profileName_pluginName_tableName every time.`,
	Aliases: []string{"aliases"},
	RunE:    controller.AliasList,
	Example: `# List the aliases
anyquery alias

# Add an alias
anyquery alias add myalias mytable

# Delete an alias
anyquery alias delete myalias`,
}

var aliasAddCmd = &cobra.Command{
	Use:     "add [alias] [table]",
	Aliases: []string{"new", "create"},
	Short:   "Add an alias",
	Long: `Add an alias.
The alias name must be unique and not already used by a table.`,
	RunE:    controller.AliasAdd,
	Example: `anyquery alias add myalias mytable`,
}

var aliasDeleteCmd = &cobra.Command{
	Use:     "delete [alias]",
	Short:   "Delete an alias",
	Aliases: []string{"rm", "remove"},
	Args:    cobra.MaximumNArgs(1),
	RunE:    controller.AliasDelete,
	Example: `anyquery alias delete myalias`,
}

var aliasListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List the aliases",
	Aliases: []string{"ls", "show"},
	RunE:    controller.AliasList,
}

func init() {
	rootCmd.AddCommand(aliasCmd)
	addFlag_commandPrintsData(aliasCmd)
	addPersistentFlag_commandModifiesConfiguration(aliasCmd)
	aliasCmd.AddCommand(aliasAddCmd)
	aliasCmd.AddCommand(aliasDeleteCmd)
	aliasCmd.AddCommand(aliasListCmd)
	addFlag_commandPrintsData(aliasListCmd)

}
