package namespace

import (
	"database/sql"
	"errors"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/julien040/anyquery/module"
	"github.com/julien040/anyquery/rpc"
	"github.com/mattn/go-sqlite3"
)

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
func (n *Namespace) LoadAnyqueryPlugin(path string, manifest rpc.PluginManifest, userConfig map[string]string) error {
	if path == "" {
		return errors.New("the path of the plugin cannot be empty")
	}
	if n.registered {
		return errors.New("the namespace is already registered. Anyquery plugin must be loaded before registering the namespace")
	}

	// Load the plugin
	for index, table := range manifest.Tables {
		plugin := &module.SQLiteModule{
			PluginPath:     path,
			PluginManifest: manifest,
			TableIndex:     index,
			UserConfig:     userConfig,
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
