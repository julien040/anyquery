package duckdb

import "database/sql"

func init() {
	sql.Register("duckdb", &Driver{})
}
