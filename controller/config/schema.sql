-- SQLite schema for anyquery
CREATE TABLE IF NOT EXISTS registry (
    -- The unique name of the registry
    name TEXT PRIMARY KEY,
    -- The HTTPS URL to fetch to get the list of plugins
    url TEXT,
    -- Last time the registry was updated
    lastUpdated INTEGER,
    -- Checksum of the last fetched registry
    checksumRegistry TEXT,
    -- JSON string of the last fetched registry
    registryJSON TEXT
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS plugin_installed (
    -- The unique id of the plugin
    id TEXT,
    -- The unique name of the plugin
    name TEXT,
    -- Description of the plugin
    description TEXT,
    -- The path to the directory containing the plugin
    path TEXT,
    -- The path to the executable file from the directory (path column)
    executablePath TEXT,
    -- A semver version of the plugin installed currently
    version TEXT,
    -- The homepage of the plugin
    homepage TEXT,
    -- The name of the registry from which the plugin was installed
    registry TEXT,
    -- The required configuration for the plugin as a JSON string
    config TEXT,
    -- Checksum of the directory containing the plugin
    checksumDir TEXT,
    -- Dev is a boolean to indicate if the plugin is in development mode
    -- If so, we don't check the checksum. Registry, homepage, and version are empty
    dev INTEGER DEFAULT 0,
    FOREIGN KEY (registry) REFERENCES registry(name),
    PRIMARY KEY (registry, id)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS profile (
    -- The unique name of the profile
    name TEXT,
    -- The linked plugin
    pluginId TEXT,
    -- The configuration for the profile as a JSON string
    config TEXT,
    FOREIGN KEY (pluginId) REFERENCES plugin_installed(id),
    PRIMARY KEY (name, pluginId)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS alias (
    tableName TEXT,
    alias TEXT,
    PRIMARY KEY (tableName, alias)
) WITHOUT ROWID;