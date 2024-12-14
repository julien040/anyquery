package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
	"github.com/shirou/gopsutil/v4/net"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func network_interfacesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &network_interfacesTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "index",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "mtu",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "hardware_addr",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "flags",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "addresses",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type network_interfacesTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from network_interfacesTable, an offset, a cursor, etc.)
type network_interfacesCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *network_interfacesTable) CreateReader() rpc.ReaderInterface {
	return &network_interfacesCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *network_interfacesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, true, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	rows := make([][]interface{}, 0, len(interfaces))
	for _, iface := range interfaces {
		addrs := make([]string, 0, len(iface.Addrs))
		for _, addr := range iface.Addrs {
			addrs = append(addrs, addr.Addr)
		}

		rows = append(rows, []interface{}{
			iface.Index,
			iface.MTU,
			iface.Name,
			iface.HardwareAddr,
			helper.Serialize(iface.Flags),
			addrs,
		})
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *network_interfacesTable) Insert(rows [][]interface{}) error {
	// Example: insert the rows in a database
	// for _, row := range rows {
	// 	err := db.Insert(row[0], row[1], row[2])
	// 	if err != nil {
	// 		return err
	// 	}
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *network_interfacesTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *network_interfacesTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *network_interfacesTable) Close() error {
	return nil
}
