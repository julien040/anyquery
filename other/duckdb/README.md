# DuckDB Golang Driver

DuckDB is an in-process SQL OLAP database. While an [official Go driver](https://github.com/marcboeker/go-duckdb) for DuckDB exists, it adds more than 30MB to the binary size. This package exposes a function to run a SQL query against a DuckDB database.

## Usage

To use this driver, you can import it in your Go code as follows:

```bash
go get github.com/julien040/anyquery/other/duckdb
```

Then, you can open a DuckDB database using the `database/sql` package:

```go
package main

func main() {
    rows, errChan := duckdb.RunDuckDBQuery("my.db", "SELECT * FROM my_table")
    for {
        select {
            case row, ok := <-rows:
                if !ok {
                    // Handle end of rows
                    break
                }
                // Process the row
                val, ok := row["my_column"] // val can be one of float64, int64, string, []byte, []interface{}, map[string]interface{}, nil
            case err, ok := <-errChan:
                if err != nil {
                    // Handle error
                    break
                }
                if !ok {
                    // The query is done, but there might rows still in the channel
                    continue
                }   
        }
    }

```

## License

Contrary to the rest of this repository, this driver (/other/duckdb) is licensed under the [MIT License](
./LICENSE.md).
