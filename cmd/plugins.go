package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:     "plugins",
	Short:   "Print the plugins installed on the system",
	Aliases: []string{"plugin", "pl"},

	RunE: controller.ListPlugins,
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
	// Because the command modifies the configuration, we add the flag
	// so that the user can specify which conf database to use
	// rather than using the default one
	addFlag_commandModifiesConfiguration(pluginsCmd)
}
