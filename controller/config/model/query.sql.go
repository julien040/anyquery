// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: query.sql

package model

import (
	"context"
	"database/sql"
)

const addAlias = `-- name: AddAlias :exec
INSERT INTO
    alias (tableName, alias)
VALUES
    (?, ?)
`

type AddAliasParams struct {
	Tablename string
	Alias     string
}

func (q *Queries) AddAlias(ctx context.Context, arg AddAliasParams) error {
	_, err := q.db.ExecContext(ctx, addAlias, arg.Tablename, arg.Alias)
	return err
}

const addConnection = `-- name: AddConnection :exec
INSERT INTO
    connections (
        databaseType,
        connectionName,
        urn,
        celScript,
        additionalMetadata
    )
VALUES
    (?, ?, ?, ?, ?)
`

type AddConnectionParams struct {
	Databasetype       string
	Connectionname     string
	Urn                string
	Celscript          string
	Additionalmetadata string
}

func (q *Queries) AddConnection(ctx context.Context, arg AddConnectionParams) error {
	_, err := q.db.ExecContext(ctx, addConnection,
		arg.Databasetype,
		arg.Connectionname,
		arg.Urn,
		arg.Celscript,
		arg.Additionalmetadata,
	)
	return err
}

const addPlugin = `-- name: AddPlugin :exec
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
        tableMetadata,
        isSharedExtension
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

type AddPluginParams struct {
	Name              string
	Description       sql.NullString
	Path              string
	Executablepath    string
	Version           string
	Homepage          sql.NullString
	Registry          string
	Config            string
	Checksumdir       sql.NullString
	Dev               int64
	Author            sql.NullString
	Tablename         string
	Tablemetadata     string
	Issharedextension int64
}

func (q *Queries) AddPlugin(ctx context.Context, arg AddPluginParams) error {
	_, err := q.db.ExecContext(ctx, addPlugin,
		arg.Name,
		arg.Description,
		arg.Path,
		arg.Executablepath,
		arg.Version,
		arg.Homepage,
		arg.Registry,
		arg.Config,
		arg.Checksumdir,
		arg.Dev,
		arg.Author,
		arg.Tablename,
		arg.Tablemetadata,
		arg.Issharedextension,
	)
	return err
}

const addProfile = `-- name: AddProfile :exec
INSERT INTO
    profile (name, pluginName, registry, config)
VALUES
    (?, ?, ?, ?)
`

type AddProfileParams struct {
	Name       string
	Pluginname string
	Registry   string
	Config     string
}

func (q *Queries) AddProfile(ctx context.Context, arg AddProfileParams) error {
	_, err := q.db.ExecContext(ctx, addProfile,
		arg.Name,
		arg.Pluginname,
		arg.Registry,
		arg.Config,
	)
	return err
}

const addRegistry = `-- name: AddRegistry :exec
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
    (?, ?, ?, unixepoch (), ?, ?)
`

type AddRegistryParams struct {
	Name             string
	Url              string
	Lastupdated      int64
	Checksumregistry string
	Registryjson     string
}

func (q *Queries) AddRegistry(ctx context.Context, arg AddRegistryParams) error {
	_, err := q.db.ExecContext(ctx, addRegistry,
		arg.Name,
		arg.Url,
		arg.Lastupdated,
		arg.Checksumregistry,
		arg.Registryjson,
	)
	return err
}

const deleteAlias = `-- name: DeleteAlias :exec
DELETE FROM alias
WHERE
    alias = ?
`

func (q *Queries) DeleteAlias(ctx context.Context, alias string) error {
	_, err := q.db.ExecContext(ctx, deleteAlias, alias)
	return err
}

const deleteConnection = `-- name: DeleteConnection :exec
DELETE FROM connections
WHERE
    connectionName = ?
`

func (q *Queries) DeleteConnection(ctx context.Context, connectionname string) error {
	_, err := q.db.ExecContext(ctx, deleteConnection, connectionname)
	return err
}

const deleteEntityAttributeValue = `-- name: DeleteEntityAttributeValue :exec
DELETE FROM entity_attribute_value
WHERE
    entity = ?
    AND attribute = ?
`

type DeleteEntityAttributeValueParams struct {
	Entity    string
	Attribute string
}

func (q *Queries) DeleteEntityAttributeValue(ctx context.Context, arg DeleteEntityAttributeValueParams) error {
	_, err := q.db.ExecContext(ctx, deleteEntityAttributeValue, arg.Entity, arg.Attribute)
	return err
}

const deletePlugin = `-- name: DeletePlugin :exec
DELETE FROM plugin_installed
WHERE
    name = ?
    AND registry = ?
`

type DeletePluginParams struct {
	Name     string
	Registry string
}

func (q *Queries) DeletePlugin(ctx context.Context, arg DeletePluginParams) error {
	_, err := q.db.ExecContext(ctx, deletePlugin, arg.Name, arg.Registry)
	return err
}

const deleteProfile = `-- name: DeleteProfile :exec
DELETE FROM profile
WHERE
    name = ?
    AND pluginName = ?
    AND registry = ?
`

type DeleteProfileParams struct {
	Name       string
	Pluginname string
	Registry   string
}

func (q *Queries) DeleteProfile(ctx context.Context, arg DeleteProfileParams) error {
	_, err := q.db.ExecContext(ctx, deleteProfile, arg.Name, arg.Pluginname, arg.Registry)
	return err
}

const deleteRegistry = `-- name: DeleteRegistry :exec
DELETE FROM registry
WHERE
    name = ?
`

// --------------------------------------------------------------------------
//
//	DELETE
//
// --------------------------------------------------------------------------
func (q *Queries) DeleteRegistry(ctx context.Context, name string) error {
	_, err := q.db.ExecContext(ctx, deleteRegistry, name)
	return err
}

const getAlias = `-- name: GetAlias :one
SELECT
    tablename, alias
FROM
    alias
WHERE
    tableName = ?
`

func (q *Queries) GetAlias(ctx context.Context, tablename string) (Alias, error) {
	row := q.db.QueryRowContext(ctx, getAlias, tablename)
	var i Alias
	err := row.Scan(&i.Tablename, &i.Alias)
	return i, err
}

const getAliasOf = `-- name: GetAliasOf :one
SELECT
    tablename, alias
FROM
    alias
WHERE
    alias = ?
`

func (q *Queries) GetAliasOf(ctx context.Context, alias string) (Alias, error) {
	row := q.db.QueryRowContext(ctx, getAliasOf, alias)
	var i Alias
	err := row.Scan(&i.Tablename, &i.Alias)
	return i, err
}

const getAliases = `-- name: GetAliases :many
SELECT
    tablename, alias
FROM
    alias
`

func (q *Queries) GetAliases(ctx context.Context) ([]Alias, error) {
	rows, err := q.db.QueryContext(ctx, getAliases)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Alias
	for rows.Next() {
		var i Alias
		if err := rows.Scan(&i.Tablename, &i.Alias); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getConnection = `-- name: GetConnection :one
SELECT
    connectionname, databasetype, urn, celscript, additionalmetadata
FROM
    connections
WHERE
    connectionName = ?
`

func (q *Queries) GetConnection(ctx context.Context, connectionname string) (Connection, error) {
	row := q.db.QueryRowContext(ctx, getConnection, connectionname)
	var i Connection
	err := row.Scan(
		&i.Connectionname,
		&i.Databasetype,
		&i.Urn,
		&i.Celscript,
		&i.Additionalmetadata,
	)
	return i, err
}

const getConnectionByURN = `-- name: GetConnectionByURN :one
SELECT
    connectionname, databasetype, urn, celscript, additionalmetadata
FROM
    connections
WHERE
    urn = ?
`

func (q *Queries) GetConnectionByURN(ctx context.Context, urn string) (Connection, error) {
	row := q.db.QueryRowContext(ctx, getConnectionByURN, urn)
	var i Connection
	err := row.Scan(
		&i.Connectionname,
		&i.Databasetype,
		&i.Urn,
		&i.Celscript,
		&i.Additionalmetadata,
	)
	return i, err
}

const getConnections = `-- name: GetConnections :many
SELECT
    connectionname, databasetype, urn, celscript, additionalmetadata
FROM
    connections
`

// --------------------------------------------------------------------------
//
//	Connections
//
// --------------------------------------------------------------------------
func (q *Queries) GetConnections(ctx context.Context) ([]Connection, error) {
	rows, err := q.db.QueryContext(ctx, getConnections)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Connection
	for rows.Next() {
		var i Connection
		if err := rows.Scan(
			&i.Connectionname,
			&i.Databasetype,
			&i.Urn,
			&i.Celscript,
			&i.Additionalmetadata,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getEntities = `-- name: GetEntities :many
SELECT DISTINCT
    entity
FROM
    entity_attribute_value
`

func (q *Queries) GetEntities(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getEntities)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var entity string
		if err := rows.Scan(&entity); err != nil {
			return nil, err
		}
		items = append(items, entity)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getEntityAttributeValue = `-- name: GetEntityAttributeValue :one
SELECT
    value
FROM
    entity_attribute_value
WHERE
    entity = ?
    AND attribute = ?
`

type GetEntityAttributeValueParams struct {
	Entity    string
	Attribute string
}

// --------------------------------------------------------------------------
//
//	Entity Attribute Value
//
// --------------------------------------------------------------------------
func (q *Queries) GetEntityAttributeValue(ctx context.Context, arg GetEntityAttributeValueParams) (string, error) {
	row := q.db.QueryRowContext(ctx, getEntityAttributeValue, arg.Entity, arg.Attribute)
	var value string
	err := row.Scan(&value)
	return value, err
}

const getEntityAttributes = `-- name: GetEntityAttributes :many
SELECT
    attribute,
    value
FROM
    entity_attribute_value
WHERE
    entity = ?
`

type GetEntityAttributesRow struct {
	Attribute string
	Value     string
}

func (q *Queries) GetEntityAttributes(ctx context.Context, entity string) ([]GetEntityAttributesRow, error) {
	rows, err := q.db.QueryContext(ctx, getEntityAttributes, entity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetEntityAttributesRow
	for rows.Next() {
		var i GetEntityAttributesRow
		if err := rows.Scan(&i.Attribute, &i.Value); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPlugin = `-- name: GetPlugin :one
SELECT
    name, description, path, executablepath, version, homepage, registry, config, checksumdir, dev, author, tablename, issharedextension, tablemetadata
FROM
    plugin_installed
WHERE
    name = ?
    AND registry = ?
`

type GetPluginParams struct {
	Name     string
	Registry string
}

func (q *Queries) GetPlugin(ctx context.Context, arg GetPluginParams) (PluginInstalled, error) {
	row := q.db.QueryRowContext(ctx, getPlugin, arg.Name, arg.Registry)
	var i PluginInstalled
	err := row.Scan(
		&i.Name,
		&i.Description,
		&i.Path,
		&i.Executablepath,
		&i.Version,
		&i.Homepage,
		&i.Registry,
		&i.Config,
		&i.Checksumdir,
		&i.Dev,
		&i.Author,
		&i.Tablename,
		&i.Issharedextension,
		&i.Tablemetadata,
	)
	return i, err
}

const getPlugins = `-- name: GetPlugins :many
SELECT
    name, description, path, executablepath, version, homepage, registry, config, checksumdir, dev, author, tablename, issharedextension, tablemetadata
FROM
    plugin_installed
`

func (q *Queries) GetPlugins(ctx context.Context) ([]PluginInstalled, error) {
	rows, err := q.db.QueryContext(ctx, getPlugins)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PluginInstalled
	for rows.Next() {
		var i PluginInstalled
		if err := rows.Scan(
			&i.Name,
			&i.Description,
			&i.Path,
			&i.Executablepath,
			&i.Version,
			&i.Homepage,
			&i.Registry,
			&i.Config,
			&i.Checksumdir,
			&i.Dev,
			&i.Author,
			&i.Tablename,
			&i.Issharedextension,
			&i.Tablemetadata,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPluginsOfRegistry = `-- name: GetPluginsOfRegistry :many
SELECT
    name, description, path, executablepath, version, homepage, registry, config, checksumdir, dev, author, tablename, issharedextension, tablemetadata
FROM
    plugin_installed
WHERE
    registry = ?
`

func (q *Queries) GetPluginsOfRegistry(ctx context.Context, registry string) ([]PluginInstalled, error) {
	rows, err := q.db.QueryContext(ctx, getPluginsOfRegistry, registry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PluginInstalled
	for rows.Next() {
		var i PluginInstalled
		if err := rows.Scan(
			&i.Name,
			&i.Description,
			&i.Path,
			&i.Executablepath,
			&i.Version,
			&i.Homepage,
			&i.Registry,
			&i.Config,
			&i.Checksumdir,
			&i.Dev,
			&i.Author,
			&i.Tablename,
			&i.Issharedextension,
			&i.Tablemetadata,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProfile = `-- name: GetProfile :one
SELECT
    name, pluginname, registry, config
FROM
    profile
WHERE
    name = ?
    AND pluginName = ?
    AND registry = ?
`

type GetProfileParams struct {
	Name       string
	Pluginname string
	Registry   string
}

func (q *Queries) GetProfile(ctx context.Context, arg GetProfileParams) (Profile, error) {
	row := q.db.QueryRowContext(ctx, getProfile, arg.Name, arg.Pluginname, arg.Registry)
	var i Profile
	err := row.Scan(
		&i.Name,
		&i.Pluginname,
		&i.Registry,
		&i.Config,
	)
	return i, err
}

const getProfiles = `-- name: GetProfiles :many
SELECT
    name, pluginname, registry, config
FROM
    profile
`

func (q *Queries) GetProfiles(ctx context.Context) ([]Profile, error) {
	rows, err := q.db.QueryContext(ctx, getProfiles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Profile
	for rows.Next() {
		var i Profile
		if err := rows.Scan(
			&i.Name,
			&i.Pluginname,
			&i.Registry,
			&i.Config,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProfilesOfPlugin = `-- name: GetProfilesOfPlugin :many
SELECT
    name, pluginname, registry, config
FROM
    profile
WHERE
    pluginName = ?
    AND registry = ?
`

type GetProfilesOfPluginParams struct {
	Pluginname string
	Registry   string
}

func (q *Queries) GetProfilesOfPlugin(ctx context.Context, arg GetProfilesOfPluginParams) ([]Profile, error) {
	rows, err := q.db.QueryContext(ctx, getProfilesOfPlugin, arg.Pluginname, arg.Registry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Profile
	for rows.Next() {
		var i Profile
		if err := rows.Scan(
			&i.Name,
			&i.Pluginname,
			&i.Registry,
			&i.Config,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProfilesOfRegistry = `-- name: GetProfilesOfRegistry :many
SELECT
    name, pluginname, registry, config
FROM
    profile
WHERE
    registry = ?
`

func (q *Queries) GetProfilesOfRegistry(ctx context.Context, registry string) ([]Profile, error) {
	rows, err := q.db.QueryContext(ctx, getProfilesOfRegistry, registry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Profile
	for rows.Next() {
		var i Profile
		if err := rows.Scan(
			&i.Name,
			&i.Pluginname,
			&i.Registry,
			&i.Config,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getRegistries = `-- name: GetRegistries :many
SELECT
    name, url, lastupdated, lastfetched, checksumregistry, registryjson
FROM
    registry
`

func (q *Queries) GetRegistries(ctx context.Context) ([]Registry, error) {
	rows, err := q.db.QueryContext(ctx, getRegistries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Registry
	for rows.Next() {
		var i Registry
		if err := rows.Scan(
			&i.Name,
			&i.Url,
			&i.Lastupdated,
			&i.Lastfetched,
			&i.Checksumregistry,
			&i.Registryjson,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getRegistry = `-- name: GetRegistry :one
SELECT
    name, url, lastupdated, lastfetched, checksumregistry, registryjson
FROM
    registry
WHERE
    name = ?
`

// --------------------------------------------------------------------------
//
//	Get
//
// --------------------------------------------------------------------------
func (q *Queries) GetRegistry(ctx context.Context, name string) (Registry, error) {
	row := q.db.QueryRowContext(ctx, getRegistry, name)
	var i Registry
	err := row.Scan(
		&i.Name,
		&i.Url,
		&i.Lastupdated,
		&i.Lastfetched,
		&i.Checksumregistry,
		&i.Registryjson,
	)
	return i, err
}

const setEntityAttributeValue = `-- name: SetEntityAttributeValue :exec
INSERT
OR REPLACE INTO entity_attribute_value (entity, attribute, value)
VALUES
    (?, ?, ?)
`

type SetEntityAttributeValueParams struct {
	Entity    string
	Attribute string
	Value     string
}

func (q *Queries) SetEntityAttributeValue(ctx context.Context, arg SetEntityAttributeValueParams) error {
	_, err := q.db.ExecContext(ctx, setEntityAttributeValue, arg.Entity, arg.Attribute, arg.Value)
	return err
}

const updateConnection = `-- name: UpdateConnection :exec
UPDATE connections
SET
    databaseType = ?,
    urn = ?,
    celScript = ?,
    additionalMetadata = ?
WHERE
    connectionName = ?
`

type UpdateConnectionParams struct {
	Databasetype       string
	Urn                string
	Celscript          string
	Additionalmetadata string
	Connectionname     string
}

func (q *Queries) UpdateConnection(ctx context.Context, arg UpdateConnectionParams) error {
	_, err := q.db.ExecContext(ctx, updateConnection,
		arg.Databasetype,
		arg.Urn,
		arg.Celscript,
		arg.Additionalmetadata,
		arg.Connectionname,
	)
	return err
}

const updatePlugin = `-- name: UpdatePlugin :exec
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
    isSharedExtension = ?,
    tableMetadata = ?
WHERE
    name = ?
    AND registry = ?
`

type UpdatePluginParams struct {
	Description       sql.NullString
	Executablepath    string
	Version           string
	Homepage          sql.NullString
	Config            string
	Checksumdir       sql.NullString
	Author            sql.NullString
	Tablename         string
	Issharedextension int64
	Tablemetadata     string
	Name              string
	Registry          string
}

func (q *Queries) UpdatePlugin(ctx context.Context, arg UpdatePluginParams) error {
	_, err := q.db.ExecContext(ctx, updatePlugin,
		arg.Description,
		arg.Executablepath,
		arg.Version,
		arg.Homepage,
		arg.Config,
		arg.Checksumdir,
		arg.Author,
		arg.Tablename,
		arg.Issharedextension,
		arg.Tablemetadata,
		arg.Name,
		arg.Registry,
	)
	return err
}

const updateProfileConfig = `-- name: UpdateProfileConfig :exec
UPDATE profile
SET
    config = ?
WHERE
    name = ?
    AND pluginName = ?
    AND registry = ?
`

type UpdateProfileConfigParams struct {
	Config     string
	Name       string
	Pluginname string
	Registry   string
}

func (q *Queries) UpdateProfileConfig(ctx context.Context, arg UpdateProfileConfigParams) error {
	_, err := q.db.ExecContext(ctx, updateProfileConfig,
		arg.Config,
		arg.Name,
		arg.Pluginname,
		arg.Registry,
	)
	return err
}

const updateProfileName = `-- name: UpdateProfileName :exec
UPDATE profile
SET
    name = ?
WHERE
    name = ?
    AND pluginName = ?
    AND registry = ?
`

type UpdateProfileNameParams struct {
	Name       string
	Name_2     string
	Pluginname string
	Registry   string
}

func (q *Queries) UpdateProfileName(ctx context.Context, arg UpdateProfileNameParams) error {
	_, err := q.db.ExecContext(ctx, updateProfileName,
		arg.Name,
		arg.Name_2,
		arg.Pluginname,
		arg.Registry,
	)
	return err
}

const updateRegistry = `-- name: UpdateRegistry :exec
UPDATE registry
SET
    url = ?,
    lastUpdated = ?,
    lastFetched = unixepoch (),
    checksumRegistry = ?,
    registryJSON = ?
WHERE
    name = ?
`

type UpdateRegistryParams struct {
	Url              string
	Lastupdated      int64
	Checksumregistry string
	Registryjson     string
	Name             string
}

// --------------------------------------------------------------------------
//
//	Updates
//
// --------------------------------------------------------------------------
func (q *Queries) UpdateRegistry(ctx context.Context, arg UpdateRegistryParams) error {
	_, err := q.db.ExecContext(ctx, updateRegistry,
		arg.Url,
		arg.Lastupdated,
		arg.Checksumregistry,
		arg.Registryjson,
		arg.Name,
	)
	return err
}

const updateRegistryFetched = `-- name: UpdateRegistryFetched :exec
UPDATE registry
SET
    lastFetched = unixepoch ()
WHERE
    name = ?
`

func (q *Queries) UpdateRegistryFetched(ctx context.Context, name string) error {
	_, err := q.db.ExecContext(ctx, updateRegistryFetched, name)
	return err
}
