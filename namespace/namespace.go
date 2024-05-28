package namespace

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/controller/config"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/julien040/anyquery/module"
	"github.com/julien040/anyquery/rpc"
	"github.com/mattn/go-sqlite3"

	"golang.org/x/mod/sumdb/dirhash"
)

func hashDirectory(path string) (string, error) {
	str, err := dirhash.HashDir(path, "", dirhash.Hash1)
	if err != nil {
		return "", err
	}

	// We remove the h1: prefix
	return str[4:], nil

}

type NamespaceConfig struct {
	// If InMemory is set to true, the SQLite database will only be stored in memory
	InMemory bool

	// The path to the SQLite database to open
	//
	// If InMemory is set to true, this field will be ignored
	Path string

	// The connection string to use to connect to the database
	//
	// If set, InMemory and Path will be ignored
	ConnectionString string

	// The page cache size in kilobytes
	//
	// By default, it is set to 50000 KB (50 MB)
	PageCacheSize int

	// Enforce foreign key constraints
	EnforceForeignKeys bool

	// The hclog logger to use from hashicorp/go-hclog
	Logger hclog.Logger
}

type Namespace struct {
	// Unexported fields

	// Check if the namespace was registered (the database/sql package was registered)
	// If so, we cannot register any more plugins
	//
	// It's to prevent registering plugins that won't be used because the db connection is already opened
	registered bool

	// The connection string to use to connect to SQLite
	connectionString string

	// The list of plugins to load
	goPluginToLoad []goPlugin

	// The list of shared objects to load
	sharedObjectToLoad []sharedObjectExtension

	// The logger to use
	logger hclog.Logger

	// The connection pool of the anyquery plugins
	pool *rpc.ConnectionPool
}

type sharedObjectExtension struct {
	// Unexported fields
	path       string
	entryPoint string
}

type goPlugin struct {
	// Unexported fields
	plugin sqlite3.Module
	name   string
}

func (n *Namespace) Init(config NamespaceConfig) error {
	// Construct the connection string
	connectionStringBuilder := strings.Builder{}
	if config.ConnectionString != "" {
		connectionStringBuilder.WriteString(config.ConnectionString)
	} else {
		if config.InMemory || config.Path == "" {
			config.Path = "anyquery.db" // If in memory, we use a default path that will be ignored
		}
		connectionStringBuilder.WriteString("file:")
		connectionStringBuilder.WriteString(config.Path)

		// Set shared cache to true
		connectionStringBuilder.WriteString("?cache=shared")

		// Open the database in memory if needed
		if config.InMemory {
			connectionStringBuilder.WriteString("&mode=memory")
		}

		// Set the page cache size
		connectionStringBuilder.WriteString("&_cache_size=")
		if config.PageCacheSize > 0 {
			// To indicate a value in KB, we have to return a negative value
			connectionStringBuilder.WriteString(strconv.Itoa((-1) * config.PageCacheSize))
		} else {
			connectionStringBuilder.WriteString("-50000")
		}

		// Set the journal mode to WAL and synchronous to NORMAL
		connectionStringBuilder.WriteString("&_journal_mode=WAL")
		connectionStringBuilder.WriteString("&_synchronous=NORMAL")

		// Set the foreign key constraints
		if config.EnforceForeignKeys {
			connectionStringBuilder.WriteString("&_foreign_keys=ON")
		} else {
			connectionStringBuilder.WriteString("&_foreign_keys=OFF")
		}
	}

	result := connectionStringBuilder.String()
	n.connectionString = result

	// Set the logger
	if config.Logger == nil {
		n.logger = hclog.New(&hclog.LoggerOptions{
			Name:   "anyquery",
			Output: hclog.DefaultOutput,
			Level:  hclog.Info,
		})
	} else {
		n.logger = config.Logger
	}

	// Create the connection pool
	n.pool = rpc.NewConnectionPool()

	return nil
}

func NewNamespace(config NamespaceConfig) (*Namespace, error) {
	n := &Namespace{}
	err := n.Init(config)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// Load a plugin written in Go
//
// Note: the plugin will only be loaded once the namespace is registered
func (n *Namespace) LoadGoPlugin(plugin sqlite3.Module, name string) error {
	if n.registered {
		return errors.New("the namespace is already registered. Go plugin must be loaded before registering the namespace")
	}
	n.goPluginToLoad = append(n.goPluginToLoad, goPlugin{plugin: plugin, name: name})
	return nil
}

// Load a SQLite extension built as a shared object (.so)
//
// Note: the plugin will only be loaded once the namespace is registered
func (n *Namespace) LoadSharedExtension(path string, entrypoint string) error {
	/* if entrypoint == "" {
		// https://www.sqlite.org/c3ref/load_extension.html#:~:text=The%20entry%20point%20is%20zProc.%20zProc%20may%20be%200
		entrypoint = "0"
	} */
	if path == "" {
		return errors.New("the path of the shared object cannot be empty")
	}
	if n.registered {
		return errors.New("the namespace is already registered. Shared extension must be loaded before registering the namespace")
	}
	n.sharedObjectToLoad = append(n.sharedObjectToLoad, sharedObjectExtension{entryPoint: entrypoint, path: path})
	return nil
}

// Register a plugin written in Go built for anyquery for each table of the manifest
//
// In the manifest, any zeroed string of table name will be ignored
func (n *Namespace) LoadAnyqueryPlugin(path string, manifest rpc.PluginManifest, userConfig map[string]string, connectionID int) error {
	if path == "" {
		return errors.New("the path of the plugin cannot be empty")
	}
	if n.registered {
		return errors.New("the namespace is already registered. Anyquery plugin must be loaded before registering the namespace")
	}

	// Load the plugin
	for index, table := range manifest.Tables {
		plugin := &module.SQLiteModule{
			ConnectionPool:  n.pool,
			ConnectionIndex: connectionID,
			PluginPath:      path,
			PluginManifest:  manifest,
			TableIndex:      index,
			UserConfig:      userConfig,
			Logger:          n.logger,
		}
		n.LoadGoPlugin(plugin, table)
	}
	return nil
}

// Register registers the namespace to the database/sql package
//
// It takes the name of the connection to register. If not specified, a random name will be generated
func (n *Namespace) Register(registerName string) (*sql.DB, error) {
	if n.registered {
		return nil, errors.New("the namespace is already registered")
	}

	// Check if the connection string is not empty
	if n.connectionString == "" {
		return nil, errors.New("the connection string cannot be empty. You must init the namespace before registering it")
	}

	// Check if the register name is empty
	if registerName == "" {
		registerName = "anyquery_custom" + strconv.Itoa(rand.Int())
	}

	for _, driver := range sql.Drivers() {
		if driver == registerName {
			return nil, errors.New("the connection string is already registered")
		}
	}

	// Register the database/sql package
	sql.Register(registerName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			// Set the limit of attached databases to 32
			// I don't know the performance impact of this
			// The number might be increased in the future
			conn.SetLimit(sqlite3.SQLITE_LIMIT_ATTACHED, 32)

			// We load the shared objects
			for _, sharedObject := range n.sharedObjectToLoad {
				err := conn.LoadExtension(sharedObject.path, sharedObject.entryPoint)
				if err != nil {
					return err
				}
			}

			// We load the Go plugins
			for _, goPlugin := range n.goPluginToLoad {
				err := conn.CreateModule(goPlugin.name, goPlugin.plugin)
				if err != nil {
					return err
				}
			}

			return nil
		},
	})

	// Create the DB connection
	db, err := sql.Open(registerName, n.connectionString)
	if err != nil {
		return nil, err
	}

	// go-sqlite3 is not thread-safe for writing
	db.SetMaxOpenConns(1)

	n.registered = true

	return db, nil

}

func (n *Namespace) GetConnectionString() string {
	return n.connectionString
}

func getManifestFromRow(row model.PluginInstalled) (rpc.PluginManifest, error) {
	// We define a plugin manifest that will be used to load the plugin
	var manifest rpc.PluginManifest

	// Case of a development plugin
	if row.Dev == 1 {
		manifest = rpc.PluginManifest{
			// We fill it with garbage data
			Name:        row.Name,
			Version:     "0.0.0",
			Description: "Development plugin " + row.Name,
		}
	} else {
		// We check if the required fields in the DB are not null

		// We check if the name is not empty
		if row.Name == "" {
			return manifest, errors.New("the plugin has an empty name")
		}

		// Unmarshal the tables
		var tables []string
		err := json.Unmarshal([]byte(row.Tablename), &tables)
		if err != nil {
			return manifest, fmt.Errorf("could not unmarshal the tables: %w", err)
		}

		manifest = rpc.PluginManifest{
			Name:        row.Name,
			Version:     row.Version,
			Description: row.Description.String,
			// We remove the first and last character (the brackets)
			Tables:     tables,
			Author:     row.Author.String,
			UserConfig: nil, // We leave it nil because it's not its job to fill it
		}

	}
	// Unmarshal the plugin config manifes
	err := json.Unmarshal([]byte(row.Config), &manifest.UserConfig)
	if err != nil {
		return manifest, fmt.Errorf("could not unmarshal the plugin config: %w", err)
	}

	return manifest, nil
}

// LoadAsAnyqueryCLI loads the plugins from the configuration of the CLI
//
// It's useful if you want to mimic the behavior of the CLI
// (internally, the CLI uses this function to load the plugins)
//
// The path is the absolute path to the database used by the CLI.
// When a plugin can't be loaded, it will be ignored and logged
func (n *Namespace) LoadAsAnyqueryCLI(path string) error {
	ctx := context.Background()
	logger := n.logger.Named("plugin_loader")
	logger.Debug("opening the database from the namespace", "path", path)
	db, queries, err := config.OpenDatabaseConnection(path)
	if err != nil {
		logger.Error("could not open the database", "error", err)
		return err
	}
	defer db.Close()

	logger.Debug("getting the plugins from the database")
	// We get the plugins
	rows, err := queries.GetPlugins(ctx)
	if err != nil {
		logger.Error("could not get the plugins from the database", "error", err)
		return err
	}

	for _, row := range rows {
		logger.Debug("loading the plugin", "plugin", row.Name, "registry", row.Registry)
		// We define a plugin manifest that will be used to load the plugin
		manifest, err := getManifestFromRow(row)
		if err != nil {
			logger.Error("could not load valid data for the plugin", "plugin", row.Name, "registry", row.Registry, "error", err)
		}

		// Ensure the checksum is correct
		// We remove temporarily the checksum check because I think it has issues (DS_Store files)
		/* hash, err := hashDirectory(row.Path)
		if err != nil {
			logger.Error("could not hash the directory", "plugin", row.Name.String, "registry", row.Registry.String, "error", err)
		}
		if hash != row.Checksumdir.String {
			logger.Error("the checksum of the directory is not correct. The plugin will not be loaded", "plugin", row.Name.String, "registry", row.Registry.String)
			continue
		} */

		// We check if the plugin is a shared object extension (a SQLite extension)
		if row.Issharedextension == 1 {
			// We load it using LoadSharedExtension because it's a SQLite extension
			err := n.LoadSharedExtension(row.Path, "")
			if err != nil {
				logger.Error("could not load the shared extension", "plugin", row.Name, "registry", row.Registry, "error", err)
			}
			continue
		}

		// We find the profiles for the plugin
		profiles, err := queries.GetProfilesOfPlugin(ctx, model.GetProfilesOfPluginParams{
			Pluginname: row.Name,
			Registry:   row.Registry,
		})
		if err != nil {
			logger.Error("could not get the profiles of the plugin", "plugin", row.Name, "error", err)
		}

		// For each profile, we register a new module for the plugin
		// If the profile is not named default, it means the user has a custom profile
		// Therefore, we need to rename the tables with a prefix to avoid conflicts.
		// At the same time, we must ensure an alias has not been defined for the table
		for connectionID, profile := range profiles {
			localManifest := manifest
			// We copy the tables to avoid modifying the original manifest
			localManifest.Tables = make([]string, len(manifest.Tables))
			copy(localManifest.Tables, manifest.Tables)
			prefix := ""
			if profile.Name != "default" {
				// We add a prefix to the tables
				prefix = profile.Name + "_"

				for index, table := range localManifest.Tables {
					// We check if the table is not an alias
					alias, err := queries.GetAlias(ctx, sql.NullString{String: prefix + table, Valid: true})
					if err != nil {
						logger.Error("could not get the alias of the table", "table", table, "error", err)
					}
					if alias.Alias.Valid {
						localManifest.Tables[index] = alias.Alias.String
					} else {
						localManifest.Tables[index] = prefix + table
					}
				}
			}
			// We unmarsal the user config
			var userConfig map[string]string
			err := json.Unmarshal([]byte(profile.Config), &userConfig)
			if err != nil {
				logger.Error("could not unmarshal the user config", "error", err)
			}

			// We load the plugin
			err = n.LoadAnyqueryPlugin(row.Path, localManifest, userConfig, connectionID)
			if err != nil {
				logger.Error("could not load the plugin", "plugin", row.Name, "error", err)
			}

		}

	}

	return nil

}
