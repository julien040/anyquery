package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/julien040/anyquery/controller/config/registry"
	"github.com/spf13/cobra"
)

func PluginsList(cmd *cobra.Command, args []string) error {
	// Open the database
	db, queries, err := requestDatabase(cmd.Flags(), true)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We get the plugins
	ctx := context.Background()
	plugins, err := queries.GetPlugins(ctx)
	if err != nil {
		return fmt.Errorf("could not get the plugins: %w", err)
	}

	// We print the plugins
	output := outputTable{
		Writer: os.Stdout,
		Columns: []string{
			"name",
			"description",
			"homepage",
			"version",
			"registry",
			"tablename",
			"author",
			"library",
		},
	}

	output.InferFlags(cmd.Flags())
	for _, plugin := range plugins {
		output.AddRow(plugin.Name, plugin.Description.String, plugin.Homepage.String, plugin.Version, plugin.Registry, plugin.Tablename, plugin.Author.String, plugin.Issharedextension)
	}

	err = output.Close()
	if err != nil {
		return fmt.Errorf("could not close the output: %w", err)
	}

	return nil
}

// Prompt the user for the plugin to install
// and return the plugin name and the registry name
func searchPlugin(querier *model.Queries) (string, string, error) {
	if !isSTDinAtty() || !isSTDoutAtty() {
		return "", "", fmt.Errorf("no tty detected")
	}

	// List the installable plugins
	plugins, err := registry.ListInstallablePluginsForPlatform(querier, runtime.GOOS+"/"+runtime.GOARCH)
	if err != nil {
		return "", "", fmt.Errorf("could not list the installable plugins: %w", err)
	}

	// Prompt the user for the plugin
	selectOption := make([]huh.Option[int], len(plugins))
	for i, plugin := range plugins {
		selectOption[i] = huh.Option[int]{
			Key:   plugin.Name,
			Value: i,
		}
	}

	var selectedIndex int

	prompt := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().Options(selectOption...).Title("Select the plugin to install").Value(&selectedIndex),
		),
	)

	err = prompt.Run()
	if err != nil {
		return "", "", fmt.Errorf("could not prompt the user: %w", err)
	}

	// Get the plugin name and the registry name
	if selectedIndex >= len(plugins) || selectedIndex < 0 {
		return "", "", fmt.Errorf("invalid index returned by the select")
	}

	return plugins[selectedIndex].Name, plugins[selectedIndex].Registry, nil
}

// Ensure the default registry is loaded, and refresh the registry if they haven't been updated in the last 7 days
//
// If the env variable ANYQUERY_SKIP_REGISTRY_CHECK is set, the function will return without doing anything
func checkRegistries(querier *model.Queries) error {
	// The number of seconds in a week
	const timeDuration = 7 * 24 * 60 * 60

	ctx := context.Background()

	// Check if the registry check is skipped
	if os.Getenv("ANYQUERY_SKIP_REGISTRY_CHECK") != "" {
		return nil
	}

	// Check if the default registry is loaded
	_, err := querier.GetRegistry(ctx, "default")
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			err := registry.AddDefaultRegistry(querier)
			if err != nil {
				return fmt.Errorf("could not load the default registry: %w", err)
			}
		} else {
			return fmt.Errorf("could not get the registry: %w", err)
		}
	}

	// Get the registries
	registries, err := querier.GetRegistries(ctx)
	if err != nil {
		return fmt.Errorf("could not get the registries: %w", err)
	}

	for _, registryDB := range registries {
		// Check if the registry has been updated in the last week
		if registryDB.Lastfetched < time.Now().Unix()-timeDuration {
			err := registry.UpdateRegistry(querier, registryDB.Name)
			if err != nil {
				return fmt.Errorf("could not update the registry: %w", err)
			}
		}
	}

	return nil
}

func PluginInstall(cmd *cobra.Command, args []string) error {
	// Request the database
	db, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// Update the registries if needed
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "Updating registries "
	if isSTDoutAtty() {
		s.Start()
	}
	// Ensure the database is up to date with the registries
	err = checkRegistries(queries)
	s.Stop() // no-op if not started
	if err != nil {
		return fmt.Errorf("could not update the registries: %w", err)
	}

	// Get the plugin to install
	var pluginName string
	var registryName string
	if len(args) == 0 {
		pluginName, registryName, err = searchPlugin(queries)
		if err != nil {
			return fmt.Errorf("could not search the plugin: %w", err)
		}
	} else if len(args) == 1 {
		pluginName = args[0]
		registryName = "default"
	} else {
		pluginName = args[1]
		registryName = args[0]
	}
	// Ensure the plugin is not already installed
	_, err = queries.GetPlugin(context.Background(), model.GetPluginParams{
		Name:     pluginName,
		Registry: registryName,
	})
	if err == nil {
		return fmt.Errorf("the plugin is already installed")
	}

	// Get the plugin
	s.Prefix = "Installing plugin " + pluginName + " "
	if isSTDoutAtty() {
		s.Start()
	}
	_, err = registry.InstallPlugin(queries, registryName, pluginName)
	s.Stop() // no-op if not started
	if err != nil {
		return fmt.Errorf("could not install the plugin: %w", err)
	}

	fmt.Println("✅ Successfully installed the plugin", pluginName)

	// Create the default profile for the plugin
	err = createOrUpdateProfile(queries, registryName, pluginName, "default")
	if err != nil {
		return err
	}

	// Print a nice message to explain what tables are available

	plugin, err := queries.GetPlugin(context.Background(), model.GetPluginParams{
		Name:     pluginName,
		Registry: registryName,
	})
	if err != nil {
		return fmt.Errorf("could not get the plugin: %w", err)
	}

	tables := []string{}
	err = json.Unmarshal([]byte(plugin.Tablename), &tables)
	if err != nil {
		return fmt.Errorf("could not unmarshal the tables: %w", err)
	}
	// If the plugin has tables (so it's not a shared extension)
	// We print a message to explain how to query the tables
	// Otherwise, we don't print anything
	if len(tables) == 0 {
		return nil
	}

	tableNamedFormatted := make([]string, len(tables))

	fmt.Println("You can now start querying these tables:")
	for i := 0; i < len(tables); i++ {
		// Modify the table so that they fit the format used by anyquery
		// <profile_name>_<plugin_name>_<table_name> unless the profile is default
		// in which case we remove the profile name (in our case, it's always default)
		tableNamedFormatted[i] = pluginName + "_" + tables[i]
		fmt.Println("	- " + tableNamedFormatted[i])
	}
	fmt.Println("By running the following command:")
	fmt.Println("	anyquery -q \"SELECT * FROM " + tableNamedFormatted[0] + ";\"")
	fmt.Println("You can access at anytime the list of tables by running:")
	fmt.Println("	anyquery -q \"SHOW TABLES;\"")

	return nil

}

func PluginUninstall(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("the plugin name is required")
	}

	// Request the database
	db, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	var pluginName string
	var registryName string

	if len(args) == 1 {
		pluginName = args[0]
		registryName = "default"
	} else {
		pluginName = args[1]
		registryName = args[0]
	}

	// Get the plugin
	_, err = queries.GetPlugin(context.Background(), model.GetPluginParams{
		Name:     pluginName,
		Registry: registryName,
	})
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("the plugin is not installed")
		}
		return fmt.Errorf("could not get the plugin: %w", err)
	}

	// Ensure the plugin is not linked to any profile
	linkedProfiles, err := queries.GetProfilesOfPlugin(context.Background(), model.GetProfilesOfPluginParams{
		Pluginname: pluginName,
		Registry:   registryName,
	})
	if err != nil {
		return fmt.Errorf("could not get the linked profiles: %w", err)
	}

	if len(linkedProfiles) > 0 {
		fmt.Println("The plugin is linked to the following profiles:")
		for _, profile := range linkedProfiles {
			fmt.Println(profile.Name)
		}
		fmt.Println("Please delete the profiles before uninstalling the plugin")
		fmt.Println("by running the following command(s):")
		for _, profile := range linkedProfiles {
			fmt.Println("	anyquery profile delete", registryName, profile.Pluginname, profile.Name)
		}
		return nil
	}

	// Uninstall the plugin
	err = registry.UninstallPlugin(queries, registryName, pluginName)
	if err != nil {
		return fmt.Errorf("could not uninstall the plugin: %w", err)
	}
	fmt.Println("✅ Successfully uninstalled the plugin", pluginName)

	return nil

}

func updateOnePlugin(queries *model.Queries, registryName string, plugin string) error {
	// Get the plugin
	pluginInfo, err := queries.GetPlugin(context.Background(), model.GetPluginParams{
		Name:     plugin,
		Registry: registryName,
	})

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("the plugin is not installed")
		}
		return fmt.Errorf("could not get the plugin from the database: %w", err)
	}

	// Find the plugin
	_, plugins, err := registry.LoadRegistry(queries, registryName)
	if err != nil {
		return err
	}

	var pluginInfoRegistry *registry.Plugin
	for _, p := range plugins.Plugins {
		if p.Name == plugin {
			pluginInfoRegistry = &p
		}
	}

	if pluginInfoRegistry == nil {
		return fmt.Errorf("the plugin is not in the registry")
	}

	// Check if the plugin is up to date
	_, version, err := registry.FindPluginVersionCandidate(*pluginInfoRegistry)
	if err != nil {
		return fmt.Errorf("could not find the plugin version candidate: %w", err)
	}
	if pluginInfo.Version == version.Version {
		fmt.Println("Plugin", plugin, "is already up to date")
		return nil
	}

	// Otherwise, update the plugin
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "Updating plugin " + plugin
	if isSTDoutAtty() {
		s.Start()
	}
	err = registry.UpdatePlugin(queries, registryName, plugin)
	s.Stop() // no-op if not started
	if err != nil {
		return fmt.Errorf("could not update the plugin: %w", err)
	}

	fmt.Println("✅ Successfully updated the plugin", plugin)

	return nil

}

func PluginUpdate(cmd *cobra.Command, args []string) error {
	// Request the database
	db, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// Update the registries if needed
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "Updating registries "
	if isSTDoutAtty() {
		s.Start()
	}
	err = checkRegistries(queries)
	s.Stop() // no-op if not started
	if err != nil {
		return fmt.Errorf("could not update the registries: %w", err)
	}

	// If a registry and a plugin are specified, we update the plugin
	if len(args) == 2 {
		return updateOnePlugin(queries, args[0], args[1])
	}

	// If only a plugin is specified, we update the plugin from the default registry
	if len(args) == 1 {
		return updateOnePlugin(queries, "default", args[0])
	}

	// Otherwise, we update all the plugins
	plugins, err := queries.GetPlugins(context.Background())
	if err != nil {
		return fmt.Errorf("could not get the plugins installed: %w", err)
	}

	for _, plugin := range plugins {
		err = updateOnePlugin(queries, plugin.Registry, plugin.Name)
		if err != nil {
			fmt.Println("Could not update the plugin", plugin.Name, "from the registry", plugin.Registry)
			fmt.Println(err)
		}
	}

	return nil
}
