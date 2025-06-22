package duckdb

import (
	"bufio"
	"database/sql/driver"
	"fmt"
	"net/url"
	"os/exec"
	"sync"
)

// Driver is a struct that implements the database/sql/driver.Driver interface.
type Driver struct {
}

func (d *Driver) Open(name string) (driver.Conn, error) {
	// Parse the query string to extract options if needed
	parsed, err := url.Parse(name)
	if err != nil { // any error is treated as the direct path to the DuckDB database
		parsed = nil
	}

	cmdArgs := []string{}
	if parsed != nil && parsed.Scheme != "" {
		// If the scheme is present, we can use it to determine how to run DuckDB
		cmdArgs = append(cmdArgs, parsed.Path)

		// Add any query parameters as command line arguments
		for key, values := range parsed.Query() {
			switch key {
			case "read_only", "readOnly":
				if len(values) > 0 && values[0] == "true" {
					cmdArgs = append(cmdArgs, "-readonly")
				}

			case "in_memory", "inMemory":
				if len(values) > 0 && values[0] == "true" {
					cmdArgs[0] = ":memory:"
				}

			case "unsigned":
				if len(values) > 0 && values[0] == "true" {
					cmdArgs = append(cmdArgs, "-unsigned")
				}
			}
		}
	} else {
		cmdArgs = append(cmdArgs, name)
	}

	cmdArgs = append(cmdArgs, "-interactive", "-cmd", ".mode jsonlines")

	conn := &Conn{
		// Initialize the command to run DuckDB
		cmd: exec.Command("duckdb", cmdArgs...),
	}

	fmt.Printf("DuckDB command: %s %s\n", conn.cmd.Path, conn.cmd.Args)

	// Set up the command's standard input and output
	stdin, err := conn.cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}
	conn.stdin = bufio.NewWriter(stdin)
	conn._stdin = stdin
	stdout, err := conn.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stdoutbuf := bufio.NewReader(stdout)

	conn.stdoutChan = make(chan string, 10) // Buffered channel to hold stdout lines
	go func() {
		for {
			line, err := stdoutbuf.ReadString('\n')
			if err != nil {
				close(conn.stdoutChan)
				return
			}
			conn.stdoutChan <- line
		}
	}()

	conn.mtx = &sync.Mutex{}

	// Start the DuckDB command
	if err := conn.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start the DuckDB CLI: %v", err)
	}

	fmt.Println("DuckDB CLI started successfully")

	// Read until the first "D" // to ensure the connection is established
	/* content, err := conn.stdout.ReadString('D')
	if err != nil {
		return nil, fmt.Errorf("failed to read from DuckDB stdout: %v (%s)", err, content)
	}

	fmt.Println("DuckDB CLI output:", content) */
	for i := 0; i < 2; i++ {
		fmt.Println("Waiting for DuckDB CLI to output the first line...")
		line := <-conn.stdoutChan
		fmt.Printf("DuckDB CLI output: %s\n", line)
	}

	// Write that we want to use changes on
	if _, err := conn.stdin.WriteString(".changes on\n"); err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %v", err)
	}
	if err := conn.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush stdin for settings changes on: %v", err)
	}

	fmt.Println("Changes mode set to 'on'")

	return conn, nil
}
