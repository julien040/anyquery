package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"

	"github.com/julien040/anyquery/controller/config"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// This file defines a few functions to help commands to run

// requestDatabase requests a database connection from the config package
//
// Its main purpose is to avoid code duplication in the commands.
// It parses the flags to see if the user wants to use a custom database (or the default one)
func requestDatabase(flags *pflag.FlagSet, readOnly bool) (*sql.DB, *model.Queries, error) {
	// We get the path to the database
	path, err := flags.GetString("config")
	if err != nil {
		path = ""
	}

	// We get an absolute path if we have one
	if path != "" {
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, nil, err
		}
	}

	// We open the database
	return config.OpenDatabaseConnection(path, readOnly)

}

func isHttpsURL(s string) bool {
	// parse the string as a URL
	url, err := url.Parse(s)
	if err != nil {
		return false
	}

	return url.Hostname() != "" && url.Scheme == "https"
}

func isSTDinAtty() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func isSTDoutAtty() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Check if --no-input is passed to the command
func isNoInputFlagSet(flags *pflag.FlagSet) bool {
	noInput, err := flags.GetBool("no-input")
	if err != nil {
		return false
	}

	return noInput
}

// List all the possible table names
// based on the format profileName_pluginName_tableName
func listTableName(queries *model.Queries) ([]string, error) {

	answer := []string{}

	// We get the plugins
	// For each plugin, we get the profiles, and parse the table name
	// We return the cartesian product of those three
	ctx := context.Background()
	plugins, err := queries.GetPlugins(ctx)
	if err != nil {
		return answer, err
	}

	for _, plugin := range plugins {
		// We get the table names
		tables := []string{}
		err = json.Unmarshal([]byte(plugin.Tablename), &tables)
		if err != nil {
			return answer, err
		}

		// We get the profiles
		profiles, err := queries.GetProfilesOfPlugin(ctx, model.GetProfilesOfPluginParams{
			Pluginname: plugin.Name,
			Registry:   plugin.Registry,
		})
		if err != nil {
			return answer, err
		}

		for _, profile := range profiles {
			// If the profile is default, we don't add the prefix
			// Otherwise, the table name is prefixed by the profile name
			var prefix string
			if profile.Name != "default" {
				prefix = profile.Name + "_"
			}
			for _, table := range tables {
				answer = append(answer, prefix+profile.Pluginname+"_"+table)
			}
		}

	}

	return answer, nil
}
