package main

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func passwordCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &passwordTable{}, &rpc.DatabaseSchema{
		PrimaryKey: -1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "id",
				Type:        rpc.ColumnTypeInt,
				Description: "The ID of the row",
			},
			{
				Name:        "username",
				Type:        rpc.ColumnTypeString,
				Description: "A random username",
			},
			{
				Name:        "password_lower",
				Type:        rpc.ColumnTypeString,
				Description: "A random password in lowercase",
			},
			{
				Name:        "password_lower_upper",
				Type:        rpc.ColumnTypeString,
				Description: "A random password in lowercase and uppercase",
			},
			{
				Name:        "password_with_special",
				Type:        rpc.ColumnTypeString,
				Description: "A random password with special characters",
			},
			{
				Name:        "password_with_special_number",
				Type:        rpc.ColumnTypeString,
				Description: "A random password with special characters and numbers",
			},
		},
	}, nil
}

type passwordTable struct {
}

type passwordCursor struct {
	rowID int64
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *passwordCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Return 1000 rows per call
	rows := make([][]interface{}, 0, 1000)

	for i := 0; i < 1000; i++ {
		rows = append(rows, []interface{}{
			t.rowID,
			gofakeit.Username(),
			gofakeit.Password(true, false, false, false, false, 12),
			gofakeit.Password(true, true, false, false, false, 12),
			gofakeit.Password(true, true, false, true, false, 12),
			gofakeit.Password(true, true, true, true, false, 12),
		})

		t.rowID++
	}

	return rows, false, nil
}

// Create a new cursor that will be used to read rows
func (t *passwordTable) CreateReader() rpc.ReaderInterface {
	return &passwordCursor{}
}

// A destructor to clean up resources
func (t *passwordTable) Close() error {
	return nil
}
