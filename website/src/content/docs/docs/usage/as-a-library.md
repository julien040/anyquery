---
title: As a library
description: Using AnyQuery as a go library
---


[Namespaces](https://pkg.go.dev/github.com/julien040/anyquery@v0.0.0-20240727154302-ea05a02d7d9b/namespace) in Anyquery represent an instance of Anyquery. Once initialized, it returns a [sql.DB](https://pkg.go.dev/database/sql#DB) ([database/sql](https://pkg.go.dev/database/sql)) instance that can be used to interact with the database.

```go
// Create a new namespace
instance, err := namespace.NewNamespace(namespace.NamespaceConfig{
        InMemory: inMemory,
        ReadOnly: readOnly,
        Path:     path,
        Logger: hclog.FromStandardLogger(loPlugin.StandardLog(), &hclog.LoggerOptions{
            Level:       hclog.LevelFromString(logLevel),
            DisableTime: true,
        }),
        DevMode: dev,
    })
    if err != nil {
        return err
    }

    // Load the configuration file. It's optional
    // and represent the source of plugins for the namespace
    err = instance.LoadAsAnyqueryCLI("path/to/config.db")
    if err != nil {
        return err
    }

    // We register the namespace and get a *sql.DB instance
    // that can be used like any database/sql instance
    db, err := instance.Register("")
    if err != nil {
        lo.Fatal("could not register namespace", "error", err)
    }
```

The `db` instance can be used to interact with the database. For example, to execute a query:

```go
rows, err := db.Query("SELECT name, stargazers_count as stars FROM github_repositories_from_user('cloudflare') ORDER BY stars DESC LIMIT 3")
if err != nil {
    return err
}
defer rows.Close()
for rows.Next() {
    var name string
    var stars int
    err = rows.Scan(&name, &stars)
    if err != nil {
        return err
    }
    fmt.Println(name, stars)
}
// Always check for errors after rows.Next() loop.
// because due to the streaming nature of the rows, the error might happen in the middle of the loop
if err = rows.Err(); err != nil {
    return err
}
```

DDL statements and INSERT/UPDATE/DELETE statements must be executed using `db.Exec`. If run using `db.Query`, it might work; however, it can silently fail and not return any error.

```go
_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
if err != nil {
    return err
}
```
