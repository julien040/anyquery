package main

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func internetCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &internetTable{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "id",
				Type:        rpc.ColumnTypeInt,
				Description: "The ID of the row",
			},
			{
				Name:        "url",
				Type:        rpc.ColumnTypeString,
				Description: "A random URL",
			},
			{
				Name:        "domain_name",
				Type:        rpc.ColumnTypeString,
				Description: "A random domain name not linked to the URL",
			},
			{
				Name:        "domain_extension",
				Type:        rpc.ColumnTypeString,
				Description: "A random domain extension not linked to the URL",
			},
			{
				Name:        "ipv4",
				Type:        rpc.ColumnTypeString,
				Description: "A random IPv4 address",
			},
			{
				Name:        "ipv6",
				Type:        rpc.ColumnTypeString,
				Description: "A random IPv6 address",
			},
			{
				Name:        "mac_address",
				Type:        rpc.ColumnTypeString,
				Description: "A random MAC address",
			},
			{
				Name:        "user_agent",
				Type:        rpc.ColumnTypeString,
				Description: "A random user agent",
			},
		},
	}, nil
}

type internetTable struct {
}

type internetCursor struct {
	rowID int64
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *internetCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Return 1000 rows per call

	rows := make([][]interface{}, 0, 1000)

	for i := 0; i < 1000; i++ {
		rows = append(rows, []interface{}{
			t.rowID,
			gofakeit.URL(),
			gofakeit.DomainName(),
			gofakeit.DomainSuffix(),
			gofakeit.IPv4Address(),
			gofakeit.IPv6Address(),
			gofakeit.MacAddress(),
			gofakeit.UserAgent(),
		})
		t.rowID++
	}

	return rows, false, nil
}

// Create a new cursor that will be used to read rows
func (t *internetTable) CreateReader() rpc.ReaderInterface {
	return &internetCursor{
		rowID: 0,
	}
}

// A destructor to clean up resources
func (t *internetTable) Close() error {
	return nil
}
