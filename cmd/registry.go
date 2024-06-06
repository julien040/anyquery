package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:        "registry",
	Short:      "List the registries where plugins can be downloaded",
	RunE:       controller.RegistryList,
	Aliases:    []string{"registries"},
	SuggestFor: []string{"store"},
	Args:       cobra.NoArgs,
}

var registryListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List the registries where plugins can be downloaded",
	RunE:    controller.RegistryList,
	Aliases: []string{"ls"},
}

var registryAddCmd = &cobra.Command{
	Use:     "add [name] [url]",
	Short:   "Add a new registry",
	Args:    cobra.MaximumNArgs(2),
	RunE:    controller.RegistryAdd,
	Aliases: []string{"new", "create", "register", "install"},
}

var registryRemoveCmd = &cobra.Command{
	Use:     "remove",
	Short:   "Remove a registry",
	RunE:    controller.RegistryRemove,
	Aliases: []string{"rm", "delete"},
}

var registryGetCmd = &cobra.Command{
	Use:     "get [name]",
	Short:   "Print informations about a registry",
	Args:    cobra.ExactArgs(1),
	RunE:    controller.RegistryGet,
	Aliases: []string{"info", "show", "inspect"},
}

var registryRefreshCmd = &cobra.Command{
	Use:   "refresh [name]",
	Short: "Keep the registry up to date with the server",
	Long: `This command will fetch the registry and save the available plugins for download in the configuration database.
If a name is provided, only this registry will be refreshed. Otherwise, all registries will be refreshed.`,
	Args:    cobra.MaximumNArgs(1),
	RunE:    controller.RegistryRefresh,
	Aliases: []string{"update", "sync", "fetch", "pull"},
}

func init() {
	rootCmd.AddCommand(registryCmd)
	addFlag_commandPrintsData(registryCmd)
	// Set the --config flag for all subcommands and the command itself
	addPersistentFlag_commandModifiesConfiguration(registryCmd)
	registryCmd.AddCommand(registryListCmd)
	addFlag_commandPrintsData(registryListCmd)
	registryCmd.AddCommand(registryAddCmd)
	registryCmd.AddCommand(registryRemoveCmd)
	registryCmd.AddCommand(registryGetCmd)
	addFlag_commandPrintsData(registryGetCmd)
	registryCmd.AddCommand(registryRefreshCmd)
}
