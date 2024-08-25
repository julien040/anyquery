package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:     "plugins",
	Short:   "Print the plugins installed on the system",
	Aliases: []string{"plugin", "pl"},

	RunE: controller.PluginsList,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install [registry] [plugin]",
	Short: "Search and install a plugin",
	Long: "Search and install a plugin\nIf a plugin is specified, it will be installed without searching" +
		"\nIf the plugin is already installed, it will fail",
	Aliases:    []string{"i", "add"},
	Args:       cobra.MaximumNArgs(2),
	SuggestFor: []string{"get"},
	RunE:       controller.PluginInstall,
	Example:    `anyquery plugin install github`,
}

var pluginUninstallCmd = &cobra.Command{
	Use:   "uninstall [registry] [plugin]",
	Short: "Uninstall a plugin and delete the linked profiles",
	Long: "Uninstall a plugin and delete the linked profiles" +
		"\nIf launch from a script, no confirmation will be asked",
	Aliases: []string{"rm", "remove", "delete"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    controller.PluginUninstall,
	Example: `anyquery plugin uninstall github`,
}

var pluginUpdateCmd = &cobra.Command{
	Use:     "update [...plugin]",
	Short:   "Update n or all plugins",
	Long:    "Update plugins\nIf no plugin is specified, all plugins will be updated",
	RunE:    controller.PluginUpdate,
	Example: `anyquery registry refresh && anyquery plugin update github`,
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
	addFlag_commandPrintsData(pluginsCmd)
	// Because the command modifies the configuration, we add the flag
	// so that the user can specify which conf database to use
	// rather than using the default one
	addPersistentFlag_commandModifiesConfiguration(pluginsCmd)
	pluginsCmd.AddCommand(pluginInstallCmd)
	addFlag_commandPrintsData(pluginInstallCmd)
	pluginsCmd.AddCommand(pluginUninstallCmd)
	pluginsCmd.AddCommand(pluginUpdateCmd)

	// Install is also a subcommand of the root command
	rootCmd.AddCommand(pluginInstallCmd)
	addFlag_commandModifiesConfiguration(pluginInstallCmd)

}
