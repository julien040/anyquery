package duckdb_test

import (
	"database/sql"
	"testing"

	_ "github.com/julien040/anyquery/other/duckdb"
)

func TestDBQuerying(t *testing.T) {

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	db.SetMaxOpenConns(1)

	defer db.Close()

	t.Log("Database opened successfully")

	// Create a table
	rows, err := db.Query("CREATE TABLE test (id INTEGER, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	rows.Close() // Close the result set

	t.Log("Table created successfully")

	// Insert some data
	rows, err = db.Query("INSERT INTO test (id, name) VALUES (1, 'Alice'), (2, 'Bob')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}
	rows.Close() // Close the result set

	t.Log("Data inserted successfully")

	// Query the data
	rows, err = db.Query("SELECT id, name FROM test")
	if err != nil {
		t.Fatalf("Failed to query data: %v", err)
	}

	t.Log("Data queried successfully")

	defer rows.Close()
	var id int
	var name string
	for rows.Next() {
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		t.Logf("Row: id=%d, name=%s", id, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows error: %v", err)
	}
	// Check if the data is correct
	if id != 2 || name != "Bob" {
		t.Errorf("Expected last row to be id=2, name='Bob', got id=%d, name='%s'", id, name)
	}
	// Cleanup
	_, err = db.Query("DROP TABLE test")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if db.Stats().OpenConnections != 0 {
		t.Errorf("Expected no open connections, got %d", db.Stats().OpenConnections)
	}
	if db.Stats().InUse != 0 {
		t.Errorf("Expected no in-use connections, got %d", db.Stats().InUse)
	}

}
