package controller

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/adrg/xdg"
	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/namespace"
	"github.com/spf13/cobra"
)

func openUserDatabase(cmd *cobra.Command, args []string) (*namespace.Namespace, *sql.DB, error) {
	// Open the database
	path := "anyquery.db"
	var inMemory, readOnly, devMode bool

	// Get the flags
	path, _ = cmd.Flags().GetString("database")
	inMemory, _ = cmd.Flags().GetBool("in-memory")
	readOnly, _ = cmd.Flags().GetBool("readonly")
	if !readOnly {
		readOnly, _ = cmd.Flags().GetBool("read-only")
	}

	// Get the file path from the args
	// The arg takes precedence over the flag
	if len(args) > 0 {
		path = args[0]
	}

	if path == ":memory:" {
		inMemory = true
	}

	// If the path is empty, we open an in-memory database
	if path == "" {
		inMemory = true
	}

	if inMemory {
		// Does not matter so we set it to a random value
		// We set it a random value because if the database is in memory,
		// the first arg will be the query, and not the database path
		path = "myrandom.db"
	}

	devMode, _ = cmd.Flags().GetBool("dev")

	// Create the logger
	var outputLog io.Writer
	logFile, _ := cmd.Flags().GetString("log-file")
	if logFile != "" {
		outputLog, _ = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		outputLog = io.Discard
	}

	logAsJSON := false
	logFormat, _ := cmd.Flags().GetString("log-format")
	if logFormat == "json" {
		logAsJSON = true
	}

	logLevelFlag, _ := cmd.Flags().GetString("log-level")
	logLevel := hclog.LevelFromString(logLevelFlag)

	namespace, err := namespace.NewNamespace(namespace.NamespaceConfig{
		InMemory: inMemory,
		ReadOnly: readOnly,
		Path:     path,
		Logger: hclog.New(
			&hclog.LoggerOptions{
				Output:     outputLog,
				JSONFormat: logAsJSON,
				Level:      logLevel,
			},
		),
		DevMode: devMode,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	anyqueryConfigPath := ""
	anyqueryConfigPath, err = cmd.Flags().GetString("config")
	if anyqueryConfigPath == "" {
		anyqueryConfigPath, err = xdg.ConfigFile("anyquery/config.db")
		if err != nil {
			return nil, nil, err
		}
	}

	err = namespace.LoadAsAnyqueryCLI(anyqueryConfigPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Get the extensions
	extensions, _ := cmd.Flags().GetStringSlice("extension")
	for _, extension := range extensions {
		err = namespace.LoadSharedExtension(extension, "")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load extension: %w", err)
		}
	}

	// Register the namespace
	db, err := namespace.Register("main")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register namespace: %w", err)
	}

	return namespace, db, nil
}
