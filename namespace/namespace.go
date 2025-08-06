package namespace

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	stdpath "path"
	"slices"
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

	// If ReadOnly is set to true, the database will be opened in read-only mode
	ReadOnly bool

	// If DevMode is set to true, the namespace will be in development mode
	// Some functions will be available to load and unload plugins
	// This can represent a security risk if the server is exposed to the internet
	// Therefore, it's recommended to disable it in production
	DevMode bool
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

	devMode bool

	// Exec statements to run when the namespace is registered
	// and the plugins are loaded
	execStatements []string

	// Arguments to pass to the exec statements
	execArgs [][]driver.Value

	// A map of the plugin table, and their modules
	// This is used to flush the insert/update/delete buffers
	anyqueryPlugins map[string]*module.SQLiteModule
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
		//connectionStringBuilder.WriteString("?cache=shared")

		// Set the page cache size
		connectionStringBuilder.WriteString("?_cache_size=")
		if config.PageCacheSize > 0 {
			// To indicate a value in KB, we have to return a negative value
			connectionStringBuilder.WriteString(strconv.Itoa((-1) * config.PageCacheSize))
		} else {
			connectionStringBuilder.WriteString("-50000")
		}

		// Open the database in memory if needed
		if config.InMemory {
			connectionStringBuilder.WriteString("&mode=memory")
		} else if config.ReadOnly {
			connectionStringBuilder.WriteString("&mode=ro")
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

	// Set the dev mode
	n.devMode = config.DevMode

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
func (n *Namespace) LoadAnyqueryPlugin(path string, manifest rpc.PluginManifest, userConfig rpc.PluginConfig, connectionID int) error {
	if path == "" {
		return errors.New("the path of the plugin cannot be empty")
	}
	if n.registered {
		return errors.New("the namespace is already registered. Anyquery plugin must be loaded before registering the namespace")
	}

	// Load the plugin
	for index, table := range manifest.Tables {
		n.logger.Debug("registering table", "table", table, "plugin", manifest.Name, "connection", connectionID)
		// Try to find the table metadata
		// To do so, we'll use manifest.TablesMetadata
		// But the map key are the table names without the plugin and profile prefix

		tableMetadata := rpc.TableMetadata{}

		if metadata, ok := manifest.TablesMetadata[table]; ok {
			tableMetadata.Description = metadata.Description
			tableMetadata.Examples = metadata.Examples
		}

		plugin := &module.SQLiteModule{
			ConnectionPool:  n.pool,
			ConnectionIndex: connectionID,
			PluginPath:      path,
			PluginManifest:  manifest,
			TableIndex:      index,
			UserConfig:      userConfig,
			Logger:          n.logger,
			Metadata:        tableMetadata,
		}
		n.LoadGoPlugin(plugin, table)
		if n.anyqueryPlugins == nil {
			n.anyqueryPlugins = make(map[string]*module.SQLiteModule)
		}
		n.anyqueryPlugins[table] = plugin
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
			// Set the limit of attached databases to 125
			// (the maximum number of attached databases in SQLite)
			//
			// Therefore, we support up to 125 remote databases
			conn.SetLimit(sqlite3.SQLITE_LIMIT_ATTACHED, 125)

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
			if n.devMode {
				devFunction := &devFunction{
					conn:      conn,
					manifests: make(map[string]manifest),
					// dev plugins get their own connection pool
					// so that they don't interfere with the main connection pool
					connectionPool: rpc.NewConnectionPool(),
				}

				// We load the development functions
				conn.RegisterFunc("unload_dev_plugin", devFunction.UnloadDevPlugin, false)
				conn.RegisterFunc("load_dev_plugin", devFunction.LoadDevPlugin, false)
				conn.RegisterFunc("reload_dev_plugin", devFunction.ReloadDevPlugin, false)
			}

			// Load the clear buffers function
			bufferFlusher := &bufferFlusher{
				modules: &n.anyqueryPlugins,
			}

			conn.RegisterFunc("clear_buffers", bufferFlusher.Clear, false)
			conn.RegisterFunc("flush_buffers", bufferFlusher.Flush, false)

			// Register JSON and CSV modules
			conn.CreateModule("json_reader", &module.JSONModule{})
			conn.CreateModule("csv_reader", &module.CsvModule{})
			conn.CreateModule("parquet_reader", &module.ParquetModule{})
			conn.CreateModule("html_reader", &module.HtmlModule{})
			conn.CreateModule("yaml_reader", &module.YamlModule{})
			conn.CreateModule("toml_reader", &module.TomlModule{})
			conn.CreateModule("jsonl_reader", &module.JSONlModule{})
			conn.CreateModule("log_reader", &module.LogModule{})

			// Register the string functions
			// like position, repeat, replace, etc.
			registerStringFunctions(conn)

			// Register the URL functions
			registerURLFunctions(conn)

			// Register the crypto functions
			registerCryptoFunctions(conn)

			// Register the date functions
			registerDateFunctions(conn)

			// Register the other functions
			registerOtherFunctions(conn)

			// Register the JSON functions
			registerJSONFunctions(conn)

			// Register the collations
			registerCollations(conn)

			// Database related modules
			conn.CreateModule("postgres_reader", &module.PostgresModule{})
			conn.CreateModule("mysql_reader", &module.MySQLModule{})
			conn.CreateModule("clickhouse_reader", &module.ClickHouseModule{})
			conn.CreateModule("duckdb_reader", &module.DuckDBModule{})
			conn.CreateModule("cassandra_reader", &module.CassandraModule{})

			// Run the exec statements
			for i, statement := range n.execStatements {
				_, err := conn.Exec(statement, n.execArgs[i])
				if err != nil {
					n.logger.Error("could not execute the exec statement", "statement", statement, "error", err)
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
	db.SetConnMaxIdleTime(0)
	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(32)

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

		// TODO
		var tablesMetadata map[string]rpc.TableMetadata
		err = json.Unmarshal([]byte(row.Tablemetadata), &tablesMetadata)

		manifest = rpc.PluginManifest{
			Name:        row.Name,
			Version:     row.Version,
			Description: row.Description.String,
			// We remove the first and last character (the brackets)
			Tables:         tables,
			Author:         row.Author.String,
			UserConfig:     nil, // We leave it nil because it's not its job to fill it
			TablesMetadata: tablesMetadata,
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
	db, queries, err := config.OpenDatabaseConnection(path, true)
	if err != nil {
		logger.Error("could not open the config database", "error", err)
		return err
	}
	defer db.Close()

	logger.Debug("getting the plugins from the database")
	// We get the plugins
	plugins, err := queries.GetPlugins(ctx)
	if err != nil {
		logger.Error("could not get the plugins from the database", "error", err)
		return err
	}

	logger.Debug("retrieved the plugins from the database", "count", len(plugins))

	for _, plugin := range plugins {
		logger.Debug("loading the plugin", "plugin", plugin.Name, "registry", plugin.Registry)
		// We define a plugin manifest that will be used to load the plugin
		manifest, err := getManifestFromRow(plugin)
		if err != nil {
			logger.Error("could not load valid data for the plugin", "plugin", plugin.Name, "registry", plugin.Registry, "error", err)
			continue
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

		// We merge the directory path with the name of the executable
		pluginPath := stdpath.Join(plugin.Path, plugin.Executablepath)

		// We check if the plugin is a shared object extension (a SQLite extension)
		if plugin.Issharedextension == 1 {
			// We load it using LoadSharedExtension because it's a SQLite extension
			err := n.LoadSharedExtension(pluginPath, "")
			if err != nil {
				logger.Error("could not load the shared extension", "plugin", plugin.Name, "registry", plugin.Registry, "error", err)
			}
			continue
		}

		// We find the profiles for the plugin
		profiles, err := queries.GetProfilesOfPlugin(ctx, model.GetProfilesOfPluginParams{
			Pluginname: plugin.Name,
			Registry:   plugin.Registry,
		})
		if err != nil {
			logger.Error("could not get the profiles of the plugin", "plugin", plugin.Name, "error", err)
			continue
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

			// The table name format is the following:
			// <profile_name>_<plugin_name>_<table_name>
			// If the profile is default, we don't add a prefix
			// so we have <plugin_name>_<table_name>

			if profile.Name != "default" {
				prefix = profile.Name + "_"
			}
			prefix += plugin.Name + "_"

			for index, table := range localManifest.Tables {
				fullName := prefix + table
				// We check if the table is not an alias
				alias, err := queries.GetAlias(ctx, fullName)
				nameToSet := fullName
				if err == nil && alias.Alias != "" {
					localManifest.Tables[index] = alias.Alias
				}
				// We set the table name
				localManifest.Tables[index] = nameToSet

				// We modify the table metadata key if needed
				if metadata, ok := manifest.TablesMetadata[table]; ok {
					localManifest.TablesMetadata[nameToSet] = metadata
					delete(localManifest.TablesMetadata, table)
				}
			}

			// We unmarsal the user config
			userConfig, err := extractUserConf(profile, localManifest)
			if err != nil {
				logger.Error("could not unmarshal the user config", "error", err)
				continue
			}

			logger.Debug("loading the profile", "profile", profile.Name, "plugin", plugin.Name, "registry", plugin.Registry,
				"table count", len(localManifest.Tables), "plugin path", pluginPath, "connection", connectionID)

			// We load the plugin
			err = n.LoadAnyqueryPlugin(pluginPath, localManifest, userConfig, connectionID)
			if err != nil {
				logger.Error("could not load the plugin", "plugin", plugin.Name, "error", err)
			}

		}

	}

	// Get the external connections
	connections, err := queries.GetConnections(ctx)
	if err != nil {
		logger.Error("could not get the connections from the database", "error", err)
	}

	// For each connection, we register the connection
	for _, connection := range connections {
		// Load the JSON metadata
		var metadata map[string]interface{}
		err := json.Unmarshal([]byte(connection.Additionalmetadata), &metadata)
		if err != nil {
			logger.Error("could not unmarshal the metadata of the connection", "connection", connection.Connectionname, "error", err)
		}

		// We register the connection
		err = n.LoadDatabaseConnection(LoadDatabaseConnectionParams{
			SchemaName:       connection.Connectionname,
			ConnectionString: connection.Urn,
			DatabaseType:     connection.Databasetype,
			Filter:           connection.Celscript,
		})
		if err != nil {
			logger.Error("could not load the connection", "connection", connection.Connectionname, "error", err)
		}
	}

	return nil

}

// Extract the user config from the profile
// and keep only the fields that are in the manifest
func extractUserConf(profile model.Profile, manifest rpc.PluginManifest) (rpc.PluginConfig, error) {
	// The first approach for the user config is to simply unmarshal it
	// Unfortunately, this approach has several issues:
	// - The user config may contain fields that are not in the manifest
	//   if the database is modified by external tools
	// - encoding/json unmarshal arrays as []interface{} but the gob encoding which is used between the CLI and the server
	//   returns gob: type not registered for interface: []interface {} as an error when passing []interface{} to the plugin
	//
	// To solve these issues, we will build the user config from the manifest manually
	// It's less efficient but it's safer
	//
	// If we don't find a required field, we return an error
	// If we find an unrequired field, we ignore it
	var userConfig rpc.PluginConfig = make(map[string]interface{})

	// We unmarshal the user config into a temporary map
	tempUnmarshal := make(map[string]interface{})
	err := json.Unmarshal([]byte(profile.Config), &tempUnmarshal)
	if err != nil {
		return userConfig, err
	}

	// We iterate over the fields of the manifest
	// and for each field, we add it to the user config
	// that we'll return
	for _, field := range manifest.UserConfig {
		var value interface{}
		// Ensure the field is in the temporary unmarshal if required
		if field.Required {
			_, ok := tempUnmarshal[field.Name]
			if !ok {
				return userConfig, fmt.Errorf("the required field %s is not found in the user config", field.Name)
			}
		}

		switch field.Type {
		// If the field is of wrong type, we return an error
		// Otherwise, we leave the zero value
		case "string":
			value = ""
			tempVal, ok := tempUnmarshal[field.Name].(string)
			if ok {
				value = tempVal
			} else if field.Required {
				return userConfig, fmt.Errorf("the field %s is not a string", field.Name)
			}
		case "int":
			value = 0
			// encoding/json unmarshal numbers as float64
			tempVal, ok := tempUnmarshal[field.Name].(float64)
			if ok {
				value = int64(tempVal)
			} else if field.Required {
				return userConfig, fmt.Errorf("the field %s is not an int", field.Name)
			}
		case "float":
			value = 0.0
			tempVal, ok := tempUnmarshal[field.Name].(float64)
			if ok {
				value = tempVal
			} else if field.Required {
				return userConfig, fmt.Errorf("the field %s is not a float", field.Name)
			}
		case "bool":
			value = false
			tempVal, ok := tempUnmarshal[field.Name].(bool)
			if ok {
				value = tempVal
			}
		case "[]string":
			value = []string{}
			tempVal, ok := tempUnmarshal[field.Name].([]interface{})
			if ok {
				for i, v := range tempVal {
					str, ok := v.(string)
					if !ok {
						return userConfig, fmt.Errorf("the field %s at index %d is not a string", field.Name, i)
					}
					value = append(value.([]string), str)
				}
			} else if field.Required {
				return userConfig, fmt.Errorf("the field %s is not an array of strings", field.Name)
			}
		case "[]int":
			value = []int64{}
			tempVal, ok := tempUnmarshal[field.Name].([]interface{})
			if ok {
				for i, v := range tempVal {
					num, ok := v.(float64)
					if !ok {
						return userConfig, fmt.Errorf("the field %s at index %d is not an int", field.Name, i)
					}
					value = append(value.([]int64), int64(num))
				}
			} else if field.Required {
				return userConfig, fmt.Errorf("the field %s is not an array of ints", field.Name)
			}
		case "[]float":
			value = []float64{}
			tempVal, ok := tempUnmarshal[field.Name].([]interface{})
			if ok {
				for i, v := range tempVal {
					num, ok := v.(float64)
					if !ok {
						return userConfig, fmt.Errorf("the field %s at index %d is not a float", field.Name, i)
					}
					value = append(value.([]float64), num)
				}
			} else if field.Required {
				return userConfig, fmt.Errorf("the field %s is not an array of floats", field.Name)
			}

		case "[]bool":
			value = []bool{}
			tempVal, ok := tempUnmarshal[field.Name].([]interface{})
			if ok {
				for i, v := range tempVal {
					b, ok := v.(bool)
					if !ok {
						return userConfig, fmt.Errorf("the field %s at index %d is not a bool", field.Name, i)
					}
					value = append(value.([]bool), b)
				}
			} else if field.Required {
				return userConfig, fmt.Errorf("the field %s is not an array of bools", field.Name)
			}
		default:
			return userConfig, fmt.Errorf("the field %s (type %s) is not recognized by the current version of Anyquery", field.Name, field.Type)

		}

		// We add the value to the user config
		userConfig[field.Name] = value

	}

	return userConfig, err
}

// The list of external connections supported by Anyquery
var SupportedConnections = []string{"MySQL", "PostgreSQL", "SQLite", "ClickHouse", "DuckDB", "Cassandra"}

// A struct to hold all the informations required to import tables from an external database
type LoadDatabaseConnectionParams struct {
	// The prefix to use for all the imported tables
	//
	// For example, if you import information_schema.tables from MySQL, and uses the SchemaName "mydb",
	// the table will be imported as mydb.information_schema_tables
	SchemaName string

	// The type of the database. It must be one of the SupportedConnections
	DatabaseType string

	// The connection string to the database
	//
	// For example, for PostgreSQL, it's postgresql://user:password@localhost:5432/dbname
	ConnectionString string

	// A CEL expression to filter the tables to import
	//
	// The expression must return a boolean
	// For example, to import only the tables named "table1" and "table2", you can use "table.name IN ['table1', 'table2']"
	Filter string

	// Additional options to pass to the connection
	Metadata map[string]interface{}
}

// Import and query tables from an external database
func (n *Namespace) LoadDatabaseConnection(args LoadDatabaseConnectionParams) error {
	// Check if the database type is supported
	if !slices.Contains(SupportedConnections, args.DatabaseType) {
		return fmt.Errorf("unsupported connection type %s. Make sure it's one of %s. Also ensure Anyquery is up to date", args.DatabaseType, strings.Join(SupportedConnections, ", "))
	}

	execStatements := []string{}
	execArgs := [][]driver.Value{}
	var err error
	switch args.DatabaseType {
	case "PostgreSQL":
		execStatements, execArgs, err = registerExternalPostgreSQL(args, n.logger)
	case "MySQL":
		execStatements, execArgs, err = registerExternalMySQL(args, n.logger)
	case "SQLite":
		execStatements, execArgs, err = registerExternalSQLite(args, n.logger)
	case "ClickHouse":
		execStatements, execArgs, err = registerExternalClickHouse(args, n.logger)
	case "DuckDB":
		execStatements, execArgs, err = registerExternalDuckDB(args, n.logger)
	case "Cassandra":
		execStatements, execArgs, err = registerExternalCassandra(args, n.logger)
	}
	if err != nil {
		return fmt.Errorf("could not fetch the tables from the external database %s(connection name: %s): %w", args.DatabaseType, args.SchemaName, err)
	}

	// We add the exec statements to the namespace
	n.execStatements = append(n.execStatements, execStatements...)
	n.execArgs = append(n.execArgs, execArgs...)

	return nil
}

type TableMetadata struct {
	// The name of the table
	Name string
	// The description of the table (from the manifest side)
	Description string

	// The description generated by the plugin.
	// It might contains additional informations
	// from a SaaS API for example
	PluginDescription string

	// The examples of the table
	Examples []string

	// The columns of the table
	// Only populated on a DescribeTable call
	Columns []rpc.DatabaseSchemaColumn

	// Whether the table supports the INSERT statement
	Insert bool

	// Whether the table supports the UPDATE statement
	Update bool

	// Whether the table supports the DELETE statement
	Delete bool
}

// List all registered anyquery plugins
func (n *Namespace) ListPluginsTables() []TableMetadata {
	tables := make([]TableMetadata, 0, len(n.anyqueryPlugins))
	for table, plugin := range n.anyqueryPlugins {
		tables = append(tables, TableMetadata{
			Name:        table,
			Description: plugin.Metadata.Description,
			Examples:    plugin.Metadata.Examples,
		})
	}

	// Sort the tables by name
	slices.SortStableFunc(tables, func(i, j TableMetadata) int {
		return strings.Compare(i.Name, j.Name)
	})
	return tables
}

// Describe a table from an anyquery plugin
//
// To ensure the columns are populated, the plugin must be loaded.
// Therefore, make a call to PRAGMA table_info(table_name) to ensure the plugin is loaded
func (n *Namespace) DescribeTable(tableName string) (TableMetadata, error) {
	// Check if the table exists
	plugin, ok := n.anyqueryPlugins[tableName]
	if !ok {
		return TableMetadata{}, fmt.Errorf("the table %s does not exist", tableName)
	}

	res := TableMetadata{
		Name:        tableName,
		Description: plugin.Metadata.Description,
		Examples:    plugin.Metadata.Examples,
	}

	// Copy the columns if the table is loaded
	if plugin.Table == nil {
		return res, nil
	}

	newCol := make([]rpc.DatabaseSchemaColumn, len(plugin.Table.Schema.Columns))
	copy(newCol, plugin.Table.Schema.Columns)
	res.Columns = newCol

	// Add the plugin description
	res.PluginDescription = plugin.Table.Schema.Description

	// Add the insert, update and delete support
	res.Insert = plugin.Table.Schema.HandlesInsert
	res.Update = plugin.Table.Schema.HandlesUpdate
	res.Delete = plugin.Table.Schema.HandlesDelete

	return res, nil

}
