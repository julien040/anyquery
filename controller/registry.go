package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/julien040/anyquery/controller/config/registry"
	"github.com/spf13/cobra"
)

func RegistryList(cmd *cobra.Command, args []string) error {
	// Open the database
	db, queries, err := requestDatabase(cmd.Flags(), true)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We get the registries
	ctx := context.Background()
	registries, err := queries.GetRegistries(ctx)
	if err != nil {
		return fmt.Errorf("could not get the registries: %w", err)
	}

	// We print the registries
	output := outputTable{
		Writer: os.Stdout,
	}

	output.InferFlags(cmd.Flags())
	output.Columns = []string{"Name", "URL", "Checksum", "Last Update", "Last check for update"}
	for _, registry := range registries {
		output.Write([]interface{}{registry.Name, registry.Url, registry.Checksumregistry, registry.Lastupdated, registry.Lastfetched})
	}

	return output.Close()

}

func RegistryAdd(cmd *cobra.Command, args []string) error {
	// Open the database on read-write mode
	db, querier, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	var name, url string

	// Act whether we have the right number of arguments
	// If we have less than 2 arguments, we will prompt the user for the infos
	// unless --no-input is set or stdin is not a tty
	if len(args) < 2 {
		if isNoInputFlagSet(cmd.Flags()) || !isSTDinAtty() {
			return fmt.Errorf("missing arguments: name or url")
		}

		// We prompt the user for the name
		huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Name").
					Description("How do you want to name this registry? It doesn't really matter, but it should be unique.").
					Value(&name).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("the name cannot be empty")
						}

						// Check if the registry already exists
						ctx := context.Background()
						_, err := querier.GetRegistry(ctx, s)
						if err == nil {
							return fmt.Errorf("a registry with the name %s already exists", s)
						}

						return nil
					})),

			huh.NewGroup(
				huh.NewInput().Title("URL").
					Description("What is the URL of the registry? It must be an HTTPS URL.").
					Value(&url).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("the URL cannot be empty")
						}

						if !isHttpsURL(s) {
							return fmt.Errorf("the URL must be an HTTPS URL")
						}

						return nil
					}),
			).Title("Add a new registry")).Run()
	} else {
		name = args[0]
		url = args[1]

		// We check if the registry already exists
		ctx := context.Background()
		_, err := querier.GetRegistry(ctx, name)
		if err == nil {
			return fmt.Errorf("a registry with the name %s already exists", name)
		}

		// We check if the URL is valid
		if !isHttpsURL(url) {
			return fmt.Errorf("the URL must be an HTTPS URL")
		}
	}

	// Create a spinner
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "Downloading registry "
	// Start the spinner if stdout is a tty
	if isSTDoutAtty() {
		s.Start()
	}

	// Download the registry
	err = registry.AddNewRegistry(querier, name, url)
	s.Stop() // no-op if not started
	if err != nil {
		return fmt.Errorf("could not download the registry: %w", err)
	}

	fmt.Printf("Registry %s added\n", name)

	return nil
}

func RegistryGet(cmd *cobra.Command, args []string) error {
	// Open the database
	db, queries, err := requestDatabase(cmd.Flags(), true)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We check if we have the right number of arguments
	if len(args) < 1 {
		return fmt.Errorf("missing argument: name")
	}

	// We get the registry
	ctx := context.Background()
	registry, err := queries.GetRegistry(ctx, args[0])
	if err != nil {
		return fmt.Errorf("could not get the registry: %w", err)
	}

	// We print the registry
	output := outputTable{
		Writer: os.Stdout,
		Type:   outputTableTypeLineByLine,
	}
	output.InferFlags(cmd.Flags())
	output.Columns = []string{"Name", "URL", "Checksum", "Last Update", "Last check for update"}
	output.Write([]interface{}{registry.Name, registry.Url, registry.Checksumregistry, registry.Lastupdated, registry.Lastfetched})

	return output.Close()
}

func RegistryRemove(cmd *cobra.Command, args []string) error {
	// Open the database on read-write mode
	db, querier, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We check if we have the right number of arguments
	if len(args) < 1 {
		return fmt.Errorf("missing argument: name")
	}

	// Check if the registry exists
	ctx := context.Background()
	_, err = querier.GetRegistry(ctx, args[0])
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("a registry with the name %s does not exist", args[0])
		}
		return fmt.Errorf("could not get the registry: %w", err)
	}

	// We remove the registry
	ctx = context.Background()
	err = querier.DeleteRegistry(ctx, args[0])
	if err != nil {
		return fmt.Errorf("could not remove the registry: %w", err)
	}

	fmt.Printf("✅ Registry %s removed\n", args[0])

	return nil
}

func RegistryRefresh(cmd *cobra.Command, args []string) error {
	// Open the database on read-write mode
	db, querier, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	// Load the default registry if not installed
	// Check if the default registry is loaded
	_, err = querier.GetRegistry(ctx, "default")
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
			s.Prefix = "Loading default registry "
			if isSTDoutAtty() {
				s.Start()
			}
			err := registry.AddDefaultRegistry(querier)
			s.Stop() // no-op if not started
			if err != nil {
				return fmt.Errorf("could not load the default registry: %w", err)
			}
		} else {
			return fmt.Errorf("could not get the registry: %w", err)
		}
	}

	if len(args) > 0 {
		name := args[0]

		// Check if the registry exists
		ctx := context.Background()
		_, err = querier.GetRegistry(ctx, name)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				return fmt.Errorf("a registry with the name %s does not exist", name)
			}
			return fmt.Errorf("could not get the registry: %w", err)
		}

		// We refresh the registry
		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Prefix = "Updating registry " + name + " "

		err := registry.UpdateRegistry(querier, name)
		s.Stop() // no-op if not started
		if err != nil {
			return fmt.Errorf("could not update the registry: %w", err)
		}

		fmt.Printf("✅ Registry %s updated\n", name)
	} else { // Otherwise we refresh all the registries
		// We get the registries
		ctx := context.Background()
		registries, err := querier.GetRegistries(ctx)
		if err != nil {
			return fmt.Errorf("could not get the registries: %w", err)
		}
		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)

		// We refresh the registries
		for _, registryL := range registries {
			s.Prefix = "Updating registry " + registryL.Name + " "

			if isSTDoutAtty() {
				s.Start()
			}
			err := registry.UpdateRegistry(querier, registryL.Name)
			s.Stop() // no-op if not started
			if err != nil {
				return fmt.Errorf("could not update the registry: %w", err)
			}

			fmt.Printf("✅ Registry %s updated\n", registryL.Name)
		}
	}
	return nil
}
