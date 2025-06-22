package duckdb

import (
	"bufio"
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// Conn is a struct that implements the database/sql/driver.Conn interface.
type Conn struct {
	cmd   *exec.Cmd
	stdin *bufio.Writer

	// A conn support only one query at a time.
	mtx *sync.Mutex

	// Wrapped by bufio
	_stdin io.WriteCloser

	// A channel that sends line from stdout
	stdoutChan chan string
}

// PrepareContext prepares a statement for execution.
func (c *Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	fmt.Printf("Preparing query: %s\n", query)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	// We will create a new FIFO special file where the results of the query will be written.
	// at the temporary folder of the system.
	path := filepath.Join(os.TempDir(), fmt.Sprintf("duckdb-query-%d-%d.temp", os.Getpid(), time.Now().UnixNano()))

	err := unix.Mkfifo(path, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create FIFO file: %v", err)
	}

	fmt.Printf("FIFO file created at: %s\n", path)
	// Ask the CLI to write the query to the file.
	if _, err := c.stdin.WriteString(".mode jsonlines\n"); err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %v", err)
	}
	if _, err := c.stdin.WriteString(fmt.Sprintf(".output %s\n", path)); err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %v", err)
	}

	if err := c.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush stdin for setting mode and output: %v", err)
	}
	fmt.Println("Output set to FIFO file")

	// To force query to return results, we need to add a semicolon at the end of the query.
	if query[len(query)-1] != ';' {
		query += ";"
	}

	if _, err := c.stdin.WriteString(query + "\n"); err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %v", err)
	}
	if err := c.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush stdin for running query: %v", err)
	}

	fmt.Printf("Query written to stdin and flushed: %s\n", query)

	// Open the file for reading.
	file, err := os.OpenFile(path, os.O_RDWR, os.ModeNamedPipe)
	fmt.Printf("Opening temporary file for reading: %s\n", path)
	if err != nil {
		fmt.Printf("Error opening temporary file for reading: %v\n", err)
		return nil, fmt.Errorf("failed to open temporary file: %v", err)
	}

	fmt.Printf("Temporary file opened for reading: %s\n", path)

	return &stmt{
		file: file,
	}, nil
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

// Close closes the connection.
func (c *Conn) Close() error {
	return nil
}

// BeginTx starts a transaction.
func (c *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return nil, nil
}

func (c *Conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
