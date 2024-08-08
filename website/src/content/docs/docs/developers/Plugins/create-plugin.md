---
title: Create a plugin
description: Learn how to create a plugin for Anyquery
---

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything. One of its strengths is the ability to extend its functionalities by creating plugins. In this guide, we will show you how to create a plugin for Anyquery.

## Prerequisites

Before you start creating a plugin, make sure you have the following installed:

- [Go](https://golang.org/doc/install)
- [Anyquery](/docs/usage/installation)

Plugins are written in Go, so you also need Go knowledge to create a plugin.

## Concepts

Anyquery plugins are Go programs that expose tables to the SQL engine. Each table is a struct that implements the [`Table`](https://pkg.go.dev/github.com/julien040/anyquery/rpc#Table) interface. A plugin can expose multiple tables.

A profile is a configuration for a plugin. It contains the necessary information to connect to the plugin (e.g. API key, URL, etc.). A plugin can have multiple profiles.

## Create a new plugin

Anyquery has a handy command to scaffold a new plugin. To create a new plugin, run:

```bash
anyquery tool dev init [repository-url] [directory]
```

- `repository-url`: The URL of the repository where the plugin will be hosted (e.g. `github.com/username/my-awesome-plugin`).
- `directory`: The directory where the plugin will be created.

This command will create a new plugin in the specified directory. The plugin will contain the following files:

- `.gitignore`: A file to ignore unnecessary files.
- `go.mod`, `go.sum`: Go modules files.
- `main.go`: The main file of the plugin. Unless for initialization purposes, you should not modify this file.
- `<table name>.go`: A file for each table that the plugin will expose. The default name is the last part of the repository URL (e.g. `my-awesome-plugin.go`).
- `Makefile`: A file to build the plugin so that you only have to run `make` to build it (expect Make to be installed).
- `devManifest.json`: A pre-configured manifest file for the plugin. You can modify it to add more tables or change the plugin's behavior. See below for more information.
- `manifest.toml`: A TOML file to configure the plugin for publication. You can modify it to add more tables, add user config, etc.
- `.goreleaser.yaml`: A configuration file for GoReleaser. GoReleaser is a tool to build and release Go binaries. It is used to build the plugin for different platforms and architectures. Running `goreleaser r --snapshot --clean` will build the plugin for all platforms and architectures supported by `anyquery`.

## The manifest file

The manifest file is a HJSON file (a superset of JSON) that describes the plugin for its development. It contains the following fields:

- `executable`: The name of the plugin's executable to load the plugin.
- `build_command`: The command to build the plugin. This command will be run when you load/reload the plugin. Note that only a single command is supported. You cannot pipe, use `&&`, glob, etc. Edit the makefile if you need to run multiple commands.
- `user_config`: A dictionnary of user configuration. Each key is the name of the profile, and the value is a list of key-value pairs. The key is the name of the configuration, and the value is the type of the configuration. The supported types are `string`, `int`, `float`, `bool`, `[]string`, `[]int`, `[]float`, `[]bool`.
- `tables`: A list of strings representing the names of the tables to expose. The tables must be defined in the plugin.
- `log-file`: The path to the log file. All `stderr` from the plugin will be redirected to this file.
- `log-level`: The log level of the plugin. The supported levels are `debug`, `info`, `warn`, `error`.

Here is an example of a manifest file:

```json
{
  "executable": "my-awesome-plugin",
  "build_command": "make",
  "user_config": {
    "default": [
        {
            "token": "string"
        }
    ]},
    "tables": ["my_table"],
    "log-file": "/path/to/log-file",
    "log-level": "debug"
}
```

## Implement a table

Let's explore the table created by the scaffold

```go
package main

import "github.com/julien040/anyquery/rpc"

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func my_tableCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
    return &my_tableTable{}, &rpc.DatabaseSchema{
        HandlesInsert: false,
        HandlesUpdate: false,
        HandlesDelete: false,
        HandleOffset:  false,
        Columns: []rpc.DatabaseSchemaColumn{
            {
                Name: "id",
                Type: rpc.ColumnTypeString,
            },
            {
                // This column is a parameter
                // Therefore, it'll be hidden in SELECT * but will be used in WHERE clauses
                // and SELECT * FROM table(<name>)
                Name:        "name",
                Type:        rpc.ColumnTypeString,
                IsParameter: true,
            },
        },
    }, nil
}

type my_tableTable struct {
}

type my_tableCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *my_tableCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
    return nil, true, nil
}

// Create a new cursor that will be used to read rows
func (t *my_tableTable) CreateReader() rpc.ReaderInterface {
    return &my_tableCursor{}
}

// A slice of rows to insert
func (t *my_tableTable) Insert(rows [][]interface{}) error {
    return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *my_tableTable) Update(rows [][]interface{}) error {
    return nil
}

// A slice of primary keys to delete
func (t *my_tableTable) Delete(primaryKeys []interface{}) error {
    return nil
}

// A destructor to clean up resources
func (t *my_tableTable) Close() error {
    return nil
}
```

Ouhhh, 72 lines of code! Let's break it down:

### `my_tableCreator`

This function is a constructor that creates a new table instance. It is called every time a new connection is made to the plugin and only once per connection.

Its first responsibility is to return a new table instance. It acts as a constructor for the table so that several connections can coexist without interfering with each other. In this example, the table instance is `&my_tableTable{}`. You can pass arguments to the table instance like an API key, a database connection, etc.

The second responsibility is to return the database schema. The database schema describes the table to Anyquery. It contains the following fields:

- `HandlesInsert`: A boolean that indicates if the table can handle insert queries.
- `HandlesUpdate`: A boolean that indicates if the table can handle update queries.
- `HandlesDelete`: A boolean that indicates if the table can handle delete queries.
- `HandleOffset`: A boolean that indicates if the table can handle the `OFFSET` clause. If it does, when the table receives a query with an `OFFSET` clause, it will return the rows starting from the offset. Otherwise, `anyquery` will fetch all the rows and apply the offset itself.
- `Columns`: A list of columns that the table exposes. Each column is a [`rpc.DatabaseSchemaColumn`](https://pkg.go.dev/github.com/julien040/anyquery/rpc#DatabaseSchemaColumn) struct that contains the following fields:
  - `Name`: The name of the column.
  - `Type`: The type of the column. The supported types are `ColumnTypeString`, `ColumnTypeInt`, `ColumnTypeFloat`, `ColumnTypeBool`.
  - `IsParameter`: A boolean that indicates if the column is a parameter. If it is, it will be hidden in `SELECT *` but will be used in `WHERE` clauses and `SELECT * FROM table(<name>)`.
  - `IsRequired`: A boolean that indicates if the column is required. If it is, the column must be present in the `FROM table(<name>)` or in the `WHERE` clause. Otherwise, the query will fail with `constraint failed`.
- `PrimaryKey`: The index (0-based) of the primary key. If the table has no primary key, it should be `-1`.
- `BufferInsert`: An integer that indicates the number of rows to buffer before inserting them. This is useful with APIs that have rate limits and support batch insert. Anyquery will buffer the rows until the buffer is full or the query is finished. If the buffer is full, Anyquery will call your plugin's `Insert` method with the buffered rows. It also flushes the buffer before running a `SELECT` query or when `anyquery` closes.
- `BufferUpdate`: Same as `BufferInsert` but for updates.
- `BufferDelete`: Same as `BufferInsert` but for deletes.

The third responsibility is to return an error if something went wrong. If an error is returned, the table won't be exposed to Anyquery.

### `my_tableTable`

This struct is the table instance. It contains the methods to interact with the table. Here is a breakdown of the methods:

- `CreateReader`: This method creates a new cursor that will be used to read rows with pagination. It should return a new instance of a struct that implements the [`rpc.ReaderInterface`](https://pkg.go.dev/github.com/julien040/anyquery/rpc#ReaderInterface) interface.
- `Insert`: This method inserts rows into the table. The rows are a slice of rows to insert. Each row is a slice of values. The method should return an error if something went wrong (can just return nil if the plugin doesn't support insert).
- `Update`: This method updates rows in the table. The rows are a slice of rows to update. Each row is a slice of values. The first element of each row is the primary key, while the rest are the values to update (the primary key is therefore present twice). The method should return an error if something went wrong (can just return nil if the plugin doesn't support update).
- `Delete`: This method deletes rows from the table. The primaryKeys are a slice of primary keys to delete. The method should return an error if something went wrong (can just return nil if the plugin doesn't support delete).

### `my_tableCursor`

This struct is the cursor instance. Having a cursor-based reader allows Anyquery to paginate the results. The cursor instance should implement the [`rpc.ReaderInterface`](https://pkg.go.dev/github.com/julien040/anyquery/rpc#ReaderInterface)interface. The interface contains the following method:

- `Query`: This method returns a slice of rows that will be returned to Anyquery and filtered. The second return value is true if the cursor has no more rows to return.
Constraints are passed as an argument to the method. They are used for optimization purposes to "pre-filter" the rows. If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out.

For example, to implement a cursor that reads rows from a database, you can add a field `pageID` in the cursor struct. Each time `query` is called, you fetch the page `pageID` from the API and increment `pageID` by one. It can also store an offset to know where to start fetching the next page.

The row slice should be a slice of slices of interface{}. Each row is a slice of values. The values can be of type `string`, `int`, `float64`, `bool`, `nil`, []string, []int, []float64, []bool. The values must be in the same order as the columns in the database schema. Any parameter column MUST NOT BE in the row slice.

### `Close`

This method is a destructor that cleans up resources. It is called when Anyquery closes the connection to the plugin. It should return an error if something went wrong.

## Debugging

To debug a plugin, you can run `anyquery` in development mode.

```bash
anyquery --dev
```

Once launched, load the plugin with the `load_dev_plugin` function.

```sql
SELECT load_dev_plugin('plugin_name', 'relative/path/to/devManifest.json');
```

`anyquery` will build the plugin using the `build_command` specified in the manifest file. If the build fails, `anyquery` will return an error. Then, it will load the plugin and expose the tables specified in the manifest file.

Each table will be prefixed with the plugin name. For example, if the plugin name is `my_plugin`, the table `my_table` will be exposed as `my_plugin_my_table`. If you have multiple user configurations, the table will be prefixed with the profile name unless the profile name is `default`. For example, if the profile name is `profile1`, the table `my_table` will be exposed as `profile1_my_plugin_my_table`.

```sql
SELECT * FROM my_plugin_my_table;
```

All logs from the plugin are redirected to the log file specified in the manifest file.

Now, make some changes in your plugin. Once you are done, run the following query to reload the plugin:

```sql
SELECT reload_dev_plugin('plugin_name');
```

Anyquery will rebuild the plugin and reload it. If the build fails, `anyquery` will return an error.

Note that you can do the exact same thing with the MySQL server. To do so, run `anyquery server --dev` and connect to the server with your favorite MySQL client.

:::caution
You CANNOT log to `stdout`. All logs and print messages should be to `stderr`. Replace `fmt` with `log`.
:::

## Adding a new table

In the plugin directory, run the following command to create a new table:

```bash
anyquery tool dev new-table [table name]
```

This command will create a new file with the table name in the plugin directory. Add the new table creator to the `main.go` file and add the table name to the `tables` field in the manifest file.

```go ins="my_new_tableCreator"
// main.go
package main

import (
    "github.com/julien040/anyquery/rpc"
)

func main() {
    plugin := rpc.NewPlugin(my_tableCreator, my_new_tableCreator) // Add the new table creator here
    plugin.Serve()
}
```

```jsonc ins="my_new_table" title="devManifest.json"
{
  "executable": "my-awesome-plugin",
  "build_command": "make",
  "user_config": {
    "default": [
        {
            "token": "string"
        }
    ]},
    "tables": ["my_table", "my_new_table"], // Add the new table name here
    "log-file": "/path/to/log-file",
    "log-level": "debug"
}
```

## Publishing a plugin

Once you are done developing your plugin, you can publish it to the Anyquery plugin repository. To do so, you need to create a manifest file in the TOML format. The manifest file contains the following fields:

- `name`: The name of the plugin. All its tables will be prefixed with this name.
- `description`: A short description of the plugin for the website.
- `repository`: The URL of the repository where the plugin is hosted.
- `tables`: A list of tables to expose. The tables must be defined in the plugin.
- `userConfig`: An array of TOML objects representing the user configuration. Each object contains the following fields:
  - `name`: The name of the configuration.
  - `type`: The type of the configuration. The supported types are `string`, `int`, `float`, `bool`, `[]string`, `[]int`, `[]float`, `[]bool`.
  - `description`: A short description of the configuration.
  - `required`: A boolean that indicates if the configuration is required.
- `file`: An array of TOML objects representing the executables for different platforms. Each object contains the following fields:
  - `platform`: The platform of the executable. One of `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.
  - `directory`: The directory to zip where the executable and its necessary files are located.
  - `executablePath`: The path to the executable relative to the `directory`.

Here is an example of the manifest of the Notion plugin:

```toml
name = "notion"
version = "0.1.1"
description = "Query your Notion database with SQL"
author = "Julien040"
license = "UNLICENSED"
repository = "https://github.com/julien040/anyquery/tree/main/plugins/notion"
homepage = "https://github.com/julien040/anyquery/tree/main/plugins/notion"
type = "anyquery"
minimumAnyqueryVersion = "0.0.1"

tables = ["database"]

# The user configuration schema
[[userConfig]]
name = "token"
description = "The API token to access the Notion API. Tutorial to get it: https://github.com/julien040/anyquery/tree/main/plugins/notion"
type = "string"
required = true
[[userConfig]]
name = "database_id"
description = "The ID of the database you want to query. Tutorial to get it:https://github.com/julien040/anyquery/tree/main/plugins/notion"
type = "string"
required = true

# Results of GoReleaser

[[file]]
platform = "linux/amd64"
directory = "dist/anyquery_linux_amd64_v1"
executablePath = "anyquery"

[[file]]
platform = "linux/arm64"
directory = "dist/anyquery_linux_arm64"
executablePath = "anyquery"

[[file]]
platform = "darwin/amd64"
directory = "dist/anyquery_darwin_amd64_v1"
executablePath = "anyquery"

[[file]]
platform = "darwin/arm64"
directory = "dist/anyquery_darwin_arm64"
executablePath = "anyquery"

[[file]]
platform = "windows/amd64"
directory = "dist/anyquery_windows_amd64_v1"
executablePath = "anyquery.exe"

[[file]]
platform = "windows/arm64"
directory = "dist/anyquery_windows_arm64"
executablePath = "anyquery.exe"
```

### Registry guidelines

To be able to publish a plugin to the Anyquery plugin repository, you need to follow these guidelines:

- A maintainer must be able to inspect the source code (under an NDAs if necessary). Some exceptions can be made for proprietary plugins. Please contact me for more information (contact at anyquery.dev).
- The plugin cannot download and execute code from the internet.
- The plugin must not log to `stdout`. All logs and print messages should be to `stderr`. Replace `fmt` with `log`.
- The plugin must not use `panic`. All errors should be returned to Anyquery.
- The plugin must not use `os.Exit`. All errors should be returned to Anyquery.
- The plugin must not use `os.Args`. All configurations should be passed as parameters.
- The recommended column names are lowercase with underscores (e.g., `first_name`).
- If the column name is reserved, prefix it with an underscore (e.g., `_type`).
- If the plugin requires a token, the token should be passed as a parameter and not hardcoded in the plugin.
- All plugins must have a README explaining how to install the plugin, the required configurations, and how to use the tables.
- Please try as much as possible to avoid CGO. Using CGO might lead to delay in plugins publication.
- All dates should be formatted as RFC3339 (e.g., `2022-01-01T00:00:00Z`). If the date is in a different format, it should be converted to RFC3339. ISO 8601, unix timestamps `YYYY-MM-DD`and hours are also accepted.
- Other datatypes should be converted to JSON arrays and objects.
- Created time of an item should be named `created_at`. Other time fields should be as much as possible suffixed with `_at` (e.g., `updated_at`, `deleted_at`).
- If necessary, you can read the configuration from the environment variables. But configuration per connection must be passed as parameters.
- Column parameters must never be in the row slice.
- Try as much as possible to store cache in `XDG_CACHE_HOME/anyquery/plugins/plugin_name`. We recommend the use of [xdg](https://github.com/adrg/xdg) to get the cache directory.

Once the manifest file is created, and you have ensured you follow the guidelines, you can publish the plugin by opening a pull request to the [Anyquery plugin repository](https://github.com/julien040/anyquery). Add your plugin to the `plugins` directory and create a pull request.

## Troubleshooting

- The plugin hangs forever without returning any results: Make sure that the cursor returns `true` in the second return value of the `Query` method when there are no more rows to return.
- The plugin returns an error: Check the logs in the log file specified in the manifest file. The logs should contain the error message.
- `anyquery` fails to load the plugin: Ensure you're not logging anything to `stdout`. All logs and print messages should be to `stderr`. Take the habit to replace `fmt` with `log`.
- Don't hesitate to open a GitHub discussion if you have any issues. I will be happy to help you. [Open a discussion](https://github.com/julien040/anyquery/discussions/new?category=plugin-development)
- If you're stuck on an implementation, check one of the many examples in the [plugins directory](https://github.com/julien040/anyquery/tree/main/plugins).
