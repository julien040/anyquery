package controller

import (
	"context"

	"github.com/charmbracelet/huh"
	"github.com/julien040/anyquery/controller/config/model"
)

// Create a new huh group that will prompt the user to select a registry
func selectRegistry(queries *model.Queries, val *string) (huh.Field, error) {
	// We get the registries
	ctx := context.Background()
	registries, err := queries.GetRegistries(ctx)
	if err != nil {
		return nil, err
	}

	selectOption := make([]huh.Option[string], len(registries))
	for i, registry := range registries {
		selectOption[i].Key = registry.Name
		selectOption[i].Value = registry.Name
	}

	// We create the group
	selectInput := huh.NewSelect[string]().Options(selectOption...).
		Title("Select a registry").Value(val)

	return selectInput, nil
}

// Create a new huh group that will prompt the user to select a plugin
func selectPlugin(queries *model.Queries, registry string, val *string) (huh.Field, error) {
	// We get the plugins
	ctx := context.Background()
	plugins, err := queries.GetPluginsOfRegistry(ctx, registry)

	if err != nil {
		return nil, err
	}

	selectOption := make([]huh.Option[string], len(plugins))
	for i, plugin := range plugins {
		selectOption[i].Key = plugin.Name
		selectOption[i].Value = plugin.Name
	}

	// We create the group
	selectInput := huh.NewSelect[string]().Options(selectOption...).
		Title("Select a plugin").Value(val)

	return selectInput, nil
}

func selectProfile(queries *model.Queries, registry, plugin string, val *string) (huh.Field, error) {
	// We get the profiles
	ctx := context.Background()
	profiles, err := queries.GetProfilesOfPlugin(ctx, model.GetProfilesOfPluginParams{
		Registry:   registry,
		Pluginname: plugin,
	})

	if err != nil {
		return nil, err
	}

	selectOption := make([]huh.Option[string], len(profiles))
	for i, profile := range profiles {
		selectOption[i].Key = profile.Name
		selectOption[i].Value = profile.Name
	}

	// We create the group
	selectInput := huh.NewSelect[string]().Options(selectOption...).
		Title("Select a profile").Value(val)

	return selectInput, nil
}
