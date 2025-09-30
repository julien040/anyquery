package main

import (
	"testing"

	"github.com/julien040/anyquery/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test schema creation
func TestAddressBookSchemaCreation(t *testing.T) {
	server, _ := setupTestServer(t)

	config := map[string]any{
		"url":      server.URL,
		"username": "testuser",
		"password": "testpass",
	}

	args := rpc.TableCreatorArgs{UserConfig: config}
	table, schema, err := addressBooksCreator(args)
	require.NoError(t, err, "Failed to create address books table")
	defer table.Close()

	// Verify schema
	assert.Len(t, schema.Columns, addrBookColCount, "Schema column count mismatch")

	// Verify column names
	expectedCols := []string{"path", "name", "description", "max_resource_size"}
	for i, expectedName := range expectedCols {
		require.Less(t, i, len(schema.Columns), "Column index out of bounds")
		assert.Equal(t, expectedName, schema.Columns[i].Name, "Column %d name mismatch", i)
	}
}

// Test real cursor with HTTP mock discovery
func TestAddressBookCursor(t *testing.T) {
	// Use the HTTP mock server that responds to PROPFIND requests
	server := setupCardDAVMockServer(t)

	// Create a real table with the HTTP mock server
	config := map[string]any{
		"url":      server.URL,
		"username": "testuser",
		"password": "testpass",
	}

	args := rpc.TableCreatorArgs{UserConfig: config}
	table, _, err := addressBooksCreator(args)
	require.NoError(t, err, "Failed to create address books table")
	defer table.Close()

	// Create the actual cursor from the table
	cursor := table.CreateReader()
	require.NotNil(t, cursor, "Cursor should not be nil")

	// Test the REAL cursor implementation with HTTP mock that supports discovery
	rows, eof, err := cursor.Query(rpc.QueryConstraint{})
	require.NoError(t, err, "Failed to query with real cursor using HTTP mock discovery")

	assert.True(t, eof, "Expected EOF to be true")
	assert.Len(t, rows, 2, "Expected 2 address books")

	// Verify specific address book data
	paths := []string{}
	names := []string{}
	for _, row := range rows {
		paths = append(paths, row[addrBookColPath].(string))
		names = append(names, row[addrBookColName].(string))
	}
	assert.Contains(t, paths, "/addressbooks/user/personal/", "Personal address book path not found")
	assert.Contains(t, paths, "/addressbooks/user/work/", "Work address book path not found")
	assert.Contains(t, names, "Personal", "Personal address book name not found")
	assert.Contains(t, names, "Work", "Work address book name not found")
}
