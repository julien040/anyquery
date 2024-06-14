-- SQLite schema for anyquery
CREATE TABLE IF NOT EXISTS registry (
    -- The unique name of the registry
    name TEXT PRIMARY KEY,
    -- The HTTPS URL to fetch to get the list of plugins
    url TEXT NOT NULL,
    -- Last time the registry was updated (unix timestamp)
    lastUpdated INTEGER NOT NULL,
    -- Last time the registry was fetched (unix timestamp)
    lastFetched INTEGER NOT NULL,
    -- Checksum of the last fetched registry
    checksumRegistry TEXT NOT NULL,
    -- JSON string of the last fetched registry
    registryJSON TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS plugin_installed (
    -- The unique name of the plugin
    name TEXT NOT NULL,
    -- Description of the plugin
    description TEXT,
    -- The path to the directory containing the plugin
    path TEXT NOT NULL,
    -- The path to the executable file from the directory (path column)
    executablePath TEXT NOT NULL,
    -- A semver version of the plugin installed currently
    version TEXT NOT NULL,
    -- The homepage of the plugin
    homepage TEXT,
    -- The name of the registry from which the plugin was installed
    registry TEXT NOT NULL,
    -- The required configuration for the plugin as a JSON string
    config TEXT NOT NULL,
    -- Checksum of the directory containing the plugin
    checksumDir TEXT,
    -- Dev is a boolean to indicate if the plugin is in development mode
    -- If so, we don't check the checksum. Registry, homepage, and version are empty
    dev INTEGER DEFAULT 0 NOT NULL,
    -- Author of the plugin
    author TEXT,
    -- Tablename is a JSON serialized array of the tables names the plugin provides
    tablename TEXT NOT NULL DEFAULT '[]',
    -- IsSharedExtension specifies if the plugin must be load as an anyquery extension or a SQLite extension
    isSharedExtension INTEGER DEFAULT 0 NOT NULL,
    FOREIGN KEY (registry) REFERENCES registry(name),
    PRIMARY KEY (registry, name)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS profile (
    -- The unique name of the profile
    name TEXT NOT NULL,
    -- The linked plugin
    pluginName TEXT NOT NULL,
    -- The linked registry of the plugin
    registry TEXT NOT NULL,
    -- The configuration for the profile as a JSON string
    config TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (registry, pluginName) REFERENCES plugin_installed(registry, name),
    PRIMARY KEY (name, pluginName, registry)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS alias (
    tableName TEXT NOT NULL,
    alias TEXT NOT NULL,
    PRIMARY KEY (tableName, alias)
) WITHOUT ROWID;