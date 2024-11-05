-- name: AddRegistry :exec
INSERT INTO
    registry (
        name,
        url,
        lastUpdated,
        lastFetched,
        checksumRegistry,
        registryJSON
    )
VALUES
    (?, ?, ?, unixepoch (), ?, ?);

-- name: AddPlugin :exec
INSERT INTO
    plugin_installed (
        name,
        description,
        path,
        executablePath,
        version,
        homepage,
        registry,
        config,
        checksumDir,
        dev,
        author,
        tablename,
        isSharedExtension
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AddProfile :exec
INSERT INTO
    profile (name, pluginName, registry, config)
VALUES
    (?, ?, ?, ?);

-- name: AddAlias :exec
INSERT INTO
    alias (tableName, alias)
VALUES
    (?, ?);

/* -------------------------------------------------------------------------- */
/*                                     Get                                    */
/* -------------------------------------------------------------------------- */
-- name: GetRegistry :one
SELECT
    *
FROM
    registry
WHERE
    name = ?;

-- name: GetPlugin :one
SELECT
    *
FROM
    plugin_installed
WHERE
    name = ?
    AND registry = ?;

-- name: GetProfile :one
SELECT
    *
FROM
    profile
WHERE
    name = ?
    AND pluginName = ?
    AND registry = ?;

-- name: GetProfilesOfPlugin :many
SELECT
    *
FROM
    profile
WHERE
    pluginName = ?
    AND registry = ?;

-- name: GetAlias :one
SELECT
    *
FROM
    alias
WHERE
    tableName = ?;

-- name: GetAliasOf :one
SELECT
    *
FROM
    alias
WHERE
    alias = ?;

-- name: GetRegistries :many
SELECT
    *
FROM
    registry;

-- name: GetPlugins :many
SELECT
    *
FROM
    plugin_installed;

-- name: GetPluginsOfRegistry :many
SELECT
    *
FROM
    plugin_installed
WHERE
    registry = ?;

-- name: GetProfilesOfRegistry :many
SELECT
    *
FROM
    profile
WHERE
    registry = ?;

-- name: GetProfiles :many
SELECT
    *
FROM
    profile;

-- name: GetAliases :many
SELECT
    *
FROM
    alias;

/* -------------------------------------------------------------------------- */
/*                                   Updates                                  */
/* -------------------------------------------------------------------------- */
-- name: UpdateRegistry :exec
UPDATE registry
SET
    url = ?,
    lastUpdated = ?,
    lastFetched = unixepoch (),
    checksumRegistry = ?,
    registryJSON = ?
WHERE
    name = ?;

-- name: UpdateRegistryFetched :exec
UPDATE registry
SET
    lastFetched = unixepoch ()
WHERE
    name = ?;

-- name: UpdatePlugin :exec
UPDATE plugin_installed
SET
    description = ?,
    executablePath = ?,
    version = ?,
    homepage = ?,
    config = ?,
    checksumDir = ?,
    author = ?,
    tablename = ?,
    isSharedExtension = ?
WHERE
    name = ?
    AND registry = ?;

-- name: UpdateProfileConfig :exec
UPDATE profile
SET
    config = ?
WHERE
    name = ?
    AND pluginName = ?
    AND registry = ?;

-- name: UpdateProfileName :exec
UPDATE profile
SET
    name = ?
WHERE
    name = ?
    AND pluginName = ?
    AND registry = ?;

/* -------------------------------------------------------------------------- */
/*                                   DELETE                                   */
/* -------------------------------------------------------------------------- */
-- name: DeleteRegistry :exec
DELETE FROM registry
WHERE
    name = ?;

-- name: DeletePlugin :exec
DELETE FROM plugin_installed
WHERE
    name = ?
    AND registry = ?;

-- name: DeleteProfile :exec
DELETE FROM profile
WHERE
    name = ?
    AND pluginName = ?
    AND registry = ?;

-- name: DeleteAlias :exec
DELETE FROM alias
WHERE
    alias = ?;

/* -------------------------------------------------------------------------- */
/*                                 Connections                                */
/* -------------------------------------------------------------------------- */
-- name: GetConnections :many
SELECT
    *
FROM
    connections;

-- name: GetConnection :one
SELECT
    *
FROM
    connections
WHERE
    connectionName = ?;

-- name: AddConnection :exec
INSERT INTO
    connections (
        databaseType,
        connectionName,
        urn,
        celScript,
        additionalMetadata
    )
VALUES
    (?, ?, ?, ?, ?);

-- name: UpdateConnection :exec
UPDATE connections
SET
    databaseType = ?,
    urn = ?,
    celScript = ?,
    additionalMetadata = ?
WHERE
    connectionName = ?;

-- name: DeleteConnection :exec
DELETE FROM connections
WHERE
    connectionName = ?;

-- name: GetConnectionByURN :one
SELECT
    *
FROM
    connections
WHERE
    urn = ?;