package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles [registry] [plugin]",
	Short: "Print the profiles installed on the system",
	Long: `Print the profiles installed on the system.
Alias to profile list.`,
	Aliases: []string{"profile"},
	RunE:    controller.ProfileList,
	Example: `# List the profiles
anyquery profiles`,
}

var profilesUpdateCmd = &cobra.Command{
	Use:   "update [registry] [plugin] [profile]",
	Short: "Update the profiles configuration",
	Long: `Update the profiles configuration.

If only two arguments are provided, we consider that the registry is the default one.
If no argument is provided, the command will prompt you the registry, the plugin and the profile to update.
Note: This command requires the tty to be interactive.`,
	RunE:    controller.ProfileUpdate,
	Example: `anyquery profiles update default github myprofile`,
}

var profilesNewCmd = &cobra.Command{
	Use:   "new [registry] [plugin] [profile]",
	Short: "Create a new profile",
	Long: `Create a new profile.

If only two arguments are provided, we consider that the registry is the default one.
If no argument is provided, the command will prompt you the registry, the plugin and the profile to create.
Note: This command requires the tty to be interactive.`,
	RunE:    controller.ProfileNew,
	Aliases: []string{"create", "add"},
	Example: `anyquery profiles new default github default`,
}

var profilesListCmd = &cobra.Command{
	Use:   "list [registry] [plugin]",
	Short: "List the profiles",
	Long: `List the profiles.

If no argument is provided, the results will not be filtered.
If only one argument is provided, the results will be filtered by the registry.
If two arguments are provided, the results will be filtered by the registry and the plugin.`,
	Aliases: []string{"ls"},
	RunE:    controller.ProfileList,
	Example: `anyquery profiles list`,
}

var profilesDeleteCmd = &cobra.Command{
	Use:   "delete [registry] [plugin] [profile]",
	Short: "Delete a profile",
	Long: `Delete a profile.

If only two arguments are provided, we consider that the registry is the default one.
If no argument is provided, the command will prompt you the registry, the plugin and the profile to create.`,
	Aliases: []string{"rm", "remove"},
	RunE:    controller.ProfileDelete,
	Example: `anyquery profiles delete default github default`,
}

func init() {
	rootCmd.AddCommand(profilesCmd)
	addPersistentFlag_commandModifiesConfiguration(profilesCmd)
	addFlag_commandPrintsData(profilesCmd)
	profilesCmd.AddCommand(profilesUpdateCmd)
	profilesCmd.AddCommand(profilesNewCmd)
	profilesCmd.AddCommand(profilesListCmd)
	addFlag_commandPrintsData(profilesListCmd)
	profilesCmd.AddCommand(profilesDeleteCmd)
}
