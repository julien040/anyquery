# DuckDB Golang Driver

DuckDB is an in-process SQL OLAP database. While an [official Go driver](https://github.com/marcboeker/go-duckdb) for DuckDB exists, it adds more than 30MB to the binary size. This driver is a minimal wrapper around the DuckDB CLI.

To use this driver, you need to have the DuckDB CLI installed on your system. You can download it from the [DuckDB releases page](https://duckdb.org/docs/installation/).

## Usage

To use this driver, you can import it in your Go code as follows:

```bash
go get github.com/julien040/anyquery/other/duckdb
```

Then, you can open a DuckDB database using the `database/sql` package:

```go
package main
import (
    _ "github.com/julien040/anyquery/other/duckdb"
    "database/sql"
)

func main() {
    db, err := sql.Open("duckdb", "path/to/your/database.duckdb")
    if err != nil {
        panic(err)
    }

    defer db.Close()
}
```

## License

Contrary to the rest of this repository, this driver (/other/duckdb) is licensed under the [MIT License](
./LICENSE.md).
