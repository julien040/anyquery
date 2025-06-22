package duckdb

import (
	"bufio"
	"database/sql/driver"
	"fmt"
	"os"
)

// stmt implements driver.Stmt.
type stmt struct {
	file   *os.File
	stdout *bufio.Reader
}

func (s *stmt) Close() error {
	if s.file != nil {
		fmt.Println("Closing file:", s.file.Name())
		s.file.Close()
	}
	// Close any resources if needed.
	return nil
}

func (s *stmt) NumInput() int {
	// Returning -1 indicates an unknown number of placeholders.
	return -1
}

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	// Execute the statement using args.
	return &result{}, nil
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	fmt.Println("stmt.Query called with args:", args)
	fileReader := bufio.NewReader(s.file)

	return &rows{
		file:       s.file,
		fileReader: fileReader,
	}, nil
}
