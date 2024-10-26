# Stanchion

Column-oriented tables in SQLite

## Why?

Stanchion is a SQLite 3 extension that brings the power of column-oriented storage to SQLite, the most widely deployed database. SQLite exclusively supports row-oriented tables, which means it is not an ideal fit for all workloads. Using the Stanchion plugin brings all of the benefits of column-oriented storage and data warehousing to anywhere that SQLite is already deployed, including your existing tech stack.

There are a number of situations where column-oriented storage outperforms row-oriented storage:
* Storing and processing metric, log, and event data
* Timeseries data storage and analysis
* Analytical queries over many rows and a few columns (e.g. calculating the average temperature over months of hourly weather data)
* Change tracking, history/temporal tables
* Anchor modeling / Datomic-like data models

Stanchion is an ideal fit for analytical queries and wide tables because it only scans data from the columns that are referenced by a given query. It uses compression techniques like run length and bit-packed encodings that significantly reduce the size of stored data, greatly reducing the cost of large data sets. This makes it an ideal solution for storing large, expanding datasets.

## Example

Download the prebuilt dynamic library for your platform from a Release or [build from source](#Build).

```sql
.load /path/to/libstanchion

CREATE VIRTUAL TABLE dnd_monsters
USING stanchion (
    id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    size INTEGER NOT NULL,
    challenge_rating FLOAT NOT NULL,
    SORT KEY (id)
);

INSERT INTO dnd_monsters (id, name, type, size, challenge_rating)
VALUES
    (1, 'Beholder', 'ABERRATION', 4, 13),
    (2, 'Gelatinous Cube', 'OOZE', 4, 2),
    (3, 'Mimic', 'MONSTROSITY', 3, 2),
    (4, 'Lich', 'UNDEAD', 3, 21);

-- Because the `dnd_monsters` table is column-oriented, the following query
-- only reads data from the `name`, `type`, and `challenge_rating` columns.
-- Data in the `id` and `size` columns are not accessed or scanned at all!
SELECT name
FROM dnd_monsters
WHERE type = 'UNDEAD' AND challenge_rating >= 18;
```

## Status

Stanchion is in *alpha*. Things may not be fully working. The storage format may change in backwards incompatible ways. **Do not use this in production**... yet.

## Build

[Install Zig (master)](https://ziglang.org/learn/getting-started/#installing-zig) and clone the `stanchion` repository. Then run:

```shell
zig build ext -Doptimize=ReleaseFast
```

The SQLite extension is the dynamic library named `libstanchion` in the `zig-out` directory.

### Running tests

By default, tests use the system SQLite library. However, stanchion's build can optionally download and compile a specific version of SQLite and use that version when running tests. Pass `-Dsqlite-test-version=$SQLITE_VERSION` to the build for unit and integration tests. For example:

```
zig build test -Dsqlite-test-version=3.38.5
```

It is also possible to launch a SQLite shell for any version of SQLite (a convenience feature for debugging):

```
zig build sqlite-shell -Dsqlite-test-version=3.43.2
```

## Usage

### Load Stanchion

Stanchion is a [Run-Time Loadable Extension](https://sqlite.org/loadext.html) that uses SQLite's virtual table system. To load an extension from the SQLite CLI, use the `.load` command. Check the documentation of the SQLite bindings you are using to see how to load an extension in your application. Here are some examples for different language bindings: [`sqlite3` for Python](https://docs.python.org/3/library/sqlite3.html#sqlite3.Connection.load_extension), [`rusqlite` for Rust](https://docs.rs/rusqlite/latest/rusqlite/struct.Connection.html#method.load_extension), [`sqlite3` for Ruby](https://rubydoc.info/gems/sqlite3/SQLite3/Database#load_extension-instance_method), and [`go-sqlite3` for Go](https://pkg.go.dev/github.com/mattn/go-sqlite3#SQLiteConn.LoadExtension).

Before loading stanchion (or any extension), you may first need to enable extension loading. Here are some examples for different language bindings: [`sqlite3` for Python](https://docs.python.org/3/library/sqlite3.html#sqlite3.Connection.enable_load_extension), [`rusqlite` for Rust](https://docs.rs/rusqlite/latest/rusqlite/struct.LoadExtensionGuard.html), and [`sqlite3` for Ruby](https://rubydoc.info/gems/sqlite3/SQLite3/Database#enable_load_extension-instance_method). Some bindings enable extension loading by default (e.g. [`go-sqlite3` for Go](https://github.com/mattn/go-sqlite3#feature--extension-list)). For more information, see the [SQLite documentation for the C API](https://sqlite.org/c3ref/enable_load_extension.html).

Stanchion is both a [persistent](https://sqlite.org/loadext.html#persistent_loadable_extensions) and [auto](https://sqlite.org/c3ref/auto_extension.html) extension. It only needs to be loaded by one connection in a process and then it will automatically be loaded by all connections in the same process. Loading it from the other connections anyway is harmless. Connections in other processes connecting to the same SQLite database still need to load stanchion.

### Create table

Creating a stanchion table works much like creating any table in SQLite:

```sql
CREATE VIRTUAL TABLE sensor_log USING stanchion (
    sensor_id TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    value FLOAT NULL,
    variance FLOAT NULL,
    severity INTEGER NOT NULL,
    SORT KEY (sensor_id, timestamp)
)
```

The `USING stanchion` phrase tells SQLite to create `sensor_log` as a virtual table that is implemented by stanchion.

The `SORT KEY` is required for all stanchion tables. It defines the clustered index, the order of the records in the table. The `SORT KEY` does not enforce uniqueness. Currently, the `SORT KEY` can only contain columns not expressions.

Stanchion tables do not support foreign keys, primary keys, check, or unique constraints. These constraints are generally less useful in the scenarios that column-oriented tables are more useful and they are not widely supported across column-oriented databases. However, some of or all of these constraints may be introduced to stanchion in the future as options.

#### Data types

The following table shows all stanchion data types. Boolean values are converted to integers when passed through SQLite. This allows them to be used from queries and through the SQLite API, which does not support a dedicated Boolean type. See [Differences from SQLite: `BOOLEAN` type](#boolean-type) for more information about the `BOOLEAN` type.

| Stanchion type  | SQLite type  | Aliases              |
|-----------------|--------------|----------------------|
| `BOOLEAN`       | `INTEGER`    | `BOOL`               |
| `INTEGER`       | `INTEGER`    | `INT`                |
| `FLOAT`         | `REAL`       | `REAL`, `DOUBLE`     |
| `BLOB`          | `BLOB`       |                      |
| `TEXT`          | `TEXT`       | `VARCHAR`<sup>*</sup>|

<sub>* Does not support character count</sub>

There is no `ANY` type and all inserted values must match the declared column type. It is not possible to do dynamic typing in stanchion tables. Stanchion tables are roughly equivalent to SQLite tables declared as `STRICT` (without the any type).

There are a small number of aliases supported for widely used type names. Declaring a column with an alias is no different than declaring it with the canonical type name.

### Add and query data

Inserting and querying data works like any other table in SQLite. Stanchion tables even work with features like the `.import` command for adding records to tables.

```sql
INSERT INTO sensor_log (sensor_id, timestamp, value, variance, severity)
VALUES
    (2064, 12433702443, 74.37, 1.06, 1),
    (2064, 12433703443, 73.12, 0.96, 1)
```

Values being inserted into Stanchion tables must be of the column's declared type. This is equivalent to declaring a SQLite table with the `STRICT` table option.

*Updating and deleting records is not currently supported. Support for update and delete will be added in the future.*

Stanchion tables can be used in all places that native tables can be used in SQLite. Consider the `SORT KEY` as a composite index when writing queries to improve query performance. In the following query, the sort key is used to reduce the amount of data scanned. And of course, only the `sensor_id`, `timestamp`, and `value` columns are read at all.

```sql
SELECT AVG(value)
FROM sensor_log
WHERE sensor_id = 2064 AND timestamp > 12433700000
```

## Differences from SQLite

### `BOOLEAN` type

Stanchion has a dedicated `BOOLEAN` type. Boolean values are used within stanchion as part of each nullable segment and exposed so it can also be used directly.

SQLite uses `INTEGER` to represent booleans. Stanchion converts `BOOLEAN` values to from `INTEGER` values when passed through SQLite. Querying a `BOOLEAN` stanchion column returns `INTEGER` SQLite values.

### Clustered index by `SORT KEY`

In Stanchion, the order of the records in the table (aka the clustered index) is controlled by the `SORT KEY`. Currently, every table in stanchion must have an explicit `SORT KEY` made up of 1 or more columns (currently expresions are not supported). It is declared when the table is created and cannot be changed. Unlike a `PRIMARY KEY`, it does not enforce uniqueness.

This differs from SQLite where tables are sorted by the `ROWID` by default or by the `PRIMARY KEY` if the table is a `WITHOUT ROWID` table.

### No uniqueness (`UNIQUE` or `PRIMARY KEY`) or foreign key constraints

This may change in the future. Implementing these will likely require external indexes. When/if a `PRIMARY KEY` is introduced, it will likley make sense to follow the lead of Clickhouse's `MergeTree` engine and require that the `PRIMARY KEY` [must be a prefix](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree#choosing-a-primary-key-that-differs-from-the-sorting-key) of the `SORT KEY`.

### No external indexes

There is currently no external index mechanism. The only index that is used to optimize queries is the clustered index, declared with `SORT KEY`.

### Column types are enforced

Values being inserted into Stanchion tables must be of the column's declared type. This is equivalent to declaring a SQLite table with the `STRICT` table option.

### No support for updates or deletes (yet)

Support for `UPDATE` and `DELETE` will be added to stanchion in the future.

### Table schemas cannot be altered

This is something that stanchion would like to support, but it may be difficult because SQLite does not have a documented way of supporting schema changes to virtual tables. This is still being investigaed.

## TODO

A high level roadmap (only roughly ordered) of upcoming features:

- [X] Rename table
- [ ] Optimize table
- [ ] Concurrency safety and testing
- [ ] More encodings: RLE, Dictionary, Chimp, etc.
- [ ] Byte level lossless compression (e.g. zstd)
- [ ] Function that converts SQLite native table to stanchion table
- [ ] Publish benchmarks against SQLite, DuckDB, and chDB
- [ ] `DELETE` & `UPDATE`
- [ ] Configuration parameters and tuning
- [ ] Guide for using stanchion within iOS and Android apps
- [ ] Schema changes (not natively support by SQLite virtual tables)
- [ ] List data type

## Data Storage Internals

### Row groups, segments, and stripes

Records in each table are grouped into row groups. Each row group contains up to a fixed number of records. Row groups are logical (not physical) groupings and are indexed by the minimum sort key value within the row group. Currently, row groups are immutable and are constructed from a batch of newly inserted records (and possibly values from an existing row group) when the number of inserted records exceeds a threshold.

A row group is made up of multiple segments. A segment contains data for a single column. Think of each row group as a span of data within a table and each segment as a span of data within a column. A segment is backed by a single SQLite `BLOB` value. Currently, segments are immutable.

Each segment is composed of 1 or more stripes. A stripe is a chunk of data of the same type that is backed by a portion (slice) of the segment's `BLOB` value. A segment can contain the following stripes: present, primary, and length. When a segment contains null values, the primary stripe is used to indicate whether the value in each record is not `NULL`. For `BOOLEAN`, `INTEGER`, and `FLOAT` columns, the values are stored in the primary stripe. For `TEXT` and `BLOB` columns, the bytes of the values are stored in the primary stripe, and the length of each value is stored in the length stripe.

### Pending inserts

When records are added to a stanchion table, they are inserted into a standard a persistent B+ Tree (read: native, row-oriented SQLite table) called the pending inserts table. Because it is a native sqlite table, records are stored row-oriented. They are sorted by the sort key so that they can be efficiently merged into row groups.

The reason that records are first stored in the pending inserts table is because creating a segment requires having all of the data that will go into the segment. When a segment is created, the encoding used depends on the values being stored in the segment, and not all encodings support being appened to efficiently. Additionally, greater compression can be achieved when the data exhibits patterns that can be exploited by encodings and there is more data per segment. The pending inserts table acts as a buffer where records are stored until there are enough records that it makes sense to create thes segments.

When a query filters on sort key columns, Stanchion applies that filter to the pending inserts directly to restrict which pending inserts are accessed. Filtering by sort key is the only indexing mechanism currently supported by Stanchion.

### Row group index

The row group index is a native, row-oriented SQLite table that indexes the row groups by the starting (min) sort key of each row group. When a query filters on sort key columns, Stanchion applies that filter to the primary index to restrict which row groups are accessed. Filtering by sort key is the only indexing mechanism currently supported by Stanchion.

## Contributing

See [CONTRIBUTING.md](.github/CONTRIBUTING.md)
