package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/shirou/gopsutil/v4/net"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func network_statsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &network_statsTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "bytes_sent",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "bytes_received",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "packets_sent",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "packets_received",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "err_in",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "err_out",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "drop_in",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "drop_out",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "fifo_in",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "fifo_out",
				Type: rpc.ColumnTypeInt,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type network_statsTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from network_statsTable, an offset, a cursor, etc.)
type network_statsCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *network_statsTable) CreateReader() rpc.ReaderInterface {
	return &network_statsCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *network_statsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	stats, err := net.IOCounters(true)
	if err != nil {
		return nil, true, fmt.Errorf("failed to get network stats: %w", err)
	}

	rows := make([][]interface{}, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, []interface{}{
			stat.Name,
			stat.BytesSent,
			stat.BytesRecv,
			stat.PacketsSent,
			stat.PacketsRecv,
			stat.Errin,
			stat.Errout,
			stat.Dropin,
			stat.Dropout,
			stat.Fifoin,
			stat.Fifoout,
		})
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *network_statsTable) Insert(rows [][]interface{}) error {
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
func (t *network_statsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *network_statsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *network_statsTable) Close() error {
	return nil
}
