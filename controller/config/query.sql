-- name: AddRegistry :exec
INSERT INTO
    registry (
        name,
        url,
        lastUpdated,
        checksumRegistry,
        registryJSON
    )
VALUES
    (?, ?, ?, ?, ?);

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
    profile (
        name,
        pluginName,
        registry,
        config
    )
VALUES
    (?, ?, ?, ?);

-- name: AddAlias :exec
INSERT INTO
    alias (tableName, alias)
VALUES
    (?, ?);

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
UPDATE
    registry
SET
    url = ?,
    lastUpdated = ?,
    checksumRegistry = ?,
    registryJSON = ?
WHERE
    name = ?;