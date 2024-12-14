package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/shirou/gopsutil/v4/process"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func process_statsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &process_statsTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "pid",
				Type:        rpc.ColumnTypeInt,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "cpu_affinity",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "cpu_percent",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "memory_percent",
				Type: rpc.ColumnTypeFloat,
			},
			// I/O counters
			{
				Name: "io_read_count",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "io_write_count",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "io_read_bytes",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "io_write_bytes",
				Type: rpc.ColumnTypeInt,
			},
			// Other
			{
				Name: "ctx_switches",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "open_files_count",
				Type: rpc.ColumnTypeInt,
			},
			// Page faults
			{
				Name: "minor_page_faults",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "major_page_faults",
				Type: rpc.ColumnTypeInt,
			},
			// CPU times stats
			{
				Name: "cpu_user_time",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "cpu_system_time",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "cpu_idle_time",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "cpu_iowait_time",
				Type: rpc.ColumnTypeFloat,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type process_statsTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from process_statsTable, an offset, a cursor, etc.)
type process_statsCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *process_statsTable) CreateReader() rpc.ReaderInterface {
	return &process_statsCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *process_statsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	pid := constraints.GetColumnConstraint(0).GetIntValue()
	if pid == 0 {
		return nil, true, nil
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, true, fmt.Errorf("process with pid %d not found", pid)
	}

	row := make([]interface{}, 15)

	cpuAffinity, err := proc.CPUAffinity()
	if err == nil {
		row[0] = cpuAffinity
	}

	cpuPercent, err := proc.CPUPercent()
	if err == nil {
		row[1] = cpuPercent
	}

	memoryPercent, err := proc.MemoryPercent()
	if err == nil {
		row[2] = memoryPercent
	}

	ioCounters, err := proc.IOCounters()
	if err == nil {
		row[3] = ioCounters.ReadCount
		row[4] = ioCounters.WriteCount
		row[5] = ioCounters.ReadBytes
		row[6] = ioCounters.WriteBytes
	}

	ctxSwitches, err := proc.NumCtxSwitches()
	if err == nil {
		row[7] = ctxSwitches
	}

	openFiles, err := proc.NumFDs()
	if err == nil {
		row[8] = openFiles
	}

	pageFaults, err := proc.PageFaults()
	if err == nil {
		row[9] = pageFaults.MinorFaults
		row[10] = pageFaults.MajorFaults
	}

	cpuTimes, err := proc.Times()
	if err == nil {
		row[11] = cpuTimes.User
		row[12] = cpuTimes.System
		row[13] = cpuTimes.Idle
		row[14] = cpuTimes.Iowait
	}

	return [][]interface{}{row}, true, nil
}

// A slice of rows to insert
func (t *process_statsTable) Insert(rows [][]interface{}) error {
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
func (t *process_statsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *process_statsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *process_statsTable) Close() error {
	return nil
}
