package main

import (
	"fmt"
	"time"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
	"github.com/shirou/gopsutil/v4/process"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func processesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &processesTable{}, &rpc.DatabaseSchema{
		// HandlesDelete: true,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "pid",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "parent_pid",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "exe",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cmdline",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cwd",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "gid",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "uids",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "nice",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "created_at",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type processesTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from processesTable, an offset, a cursor, etc.)
type processesCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *processesTable) CreateReader() rpc.ReaderInterface {
	return &processesCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *processesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, true, fmt.Errorf("failed to get processes: %w", err)
	}
	rows := make([][]interface{}, 0, len(processes))

	for _, p := range processes {
		row := make([]interface{}, 0, 7)
		row = append(row, p.Pid)

		// The name of the process
		var name interface{}
		if pName, err := p.Name(); err == nil {
			name = pName
		}
		row = append(row, name)

		// The parent pid of the process
		var parentPid interface{}
		if pParentPid, err := p.Ppid(); err == nil {
			parentPid = pParentPid
		}
		row = append(row, parentPid)

		// The executable path of the process
		var exe interface{}
		if pExe, err := p.Exe(); err == nil {
			exe = pExe
		}
		row = append(row, exe)

		// The command line of the process
		var cmdline interface{}
		if pCmdline, err := p.Cmdline(); err == nil {
			cmdline = pCmdline
		}
		row = append(row, cmdline)

		// The current working directory of the process
		var cwd interface{}
		if pCwd, err := p.Cwd(); err == nil {
			cwd = pCwd
		}
		row = append(row, cwd)

		// The group id of the process
		var gid interface{}
		if pGid, err := p.Gids(); err == nil {
			gid = helper.Serialize(pGid)
		}
		row = append(row, gid)

		// The user id of the process
		var uids interface{}
		if pUids, err := p.Uids(); err == nil {
			uids = helper.Serialize(pUids)
		}
		row = append(row, uids)

		// The nice value of the process
		var nice interface{}
		if pNice, err := p.Nice(); err == nil {
			nice = pNice
		}
		row = append(row, nice)

		p.CreateTime()

		// The created time of the process
		var createdAt interface{}
		if pCreatedAt, err := p.CreateTime(); err == nil {
			parsed := time.UnixMilli(int64(pCreatedAt))
			createdAt = parsed.Format(time.RFC3339)
		}
		row = append(row, createdAt)

		rows = append(rows, row)
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *processesTable) Insert(rows [][]interface{}) error {
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
func (t *processesTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *processesTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *processesTable) Close() error {
	return nil
}
