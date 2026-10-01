package duckdb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Executes a DuckDB query using the DuckDB CLI.
//
// It returns a channel of map[string]interface{} for the results, string keys are the column names, and interface{} values are the column values.
// It also returns a channel of errors for any errors that occur during the execution of the query.
// RunDuckDBQuery runs query against the database at path with the DuckDB CLI.
//
// When sandboxed, DuckDB's own file and network access (read_csv, read_text,
// httpfs, extension installs, ...) is turned off and the configuration locked,
// so neither the query nor a view stored in the database can reach files or
// URLs beyond the database itself.
func RunDuckDBQuery(path string, query string, sandboxed bool) (<-chan map[string]interface{}, <-chan error) {
	res := make(chan map[string]interface{}, 8)
	chanErr := make(chan error, 1)

	args := []string{path, "-readonly"}
	if sandboxed {
		args = append(args, "-cmd", "SET enable_external_access = false", "-cmd", "SET lock_configuration = true")
	}
	args = append(args, "-cmd", ".mode jsonlines")
	cmd := exec.Command("duckdb", args...)
	cmd.Stdin = strings.NewReader(query + "\n")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		chanErr <- fmt.Errorf("error creating stdout pipe: %v", err)
		return nil, chanErr
	}

	stderrBuffer := &bytes.Buffer{}
	cmd.Stderr = stderrBuffer

	if err := cmd.Start(); err != nil {
		chanErr <- fmt.Errorf("error starting DuckDB command: %v", err)
		return nil, chanErr
	}

	// We'll read the output from the command line by line
	// If the line starts with "{" it is a JSON object, otherwise it is an error message
	scanner := bufio.NewScanner(stdout)
	go func() {
		defer close(res)
		defer close(chanErr)
		scanner.Split(bufio.ScanLines) // Set the scanner to read line by line
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue // Skip empty lines
			}
			if line[0] == '{' {
				// Parse the JSON object
				var jsonObj map[string]interface{}
				if err := json.Unmarshal(line, &jsonObj); err != nil {
					chanErr <- fmt.Errorf("error unmarshalling JSON object: %v", err)
					continue
				}
				res <- jsonObj
			} else {
				// If the line does not start with '{', it is an error message
				chanErr <- fmt.Errorf("error from DuckDB: %s", string(line))
			}
		}

		if err := scanner.Err(); err != nil {
			chanErr <- fmt.Errorf("error reading from DuckDB stdout: %v", err)
		}

		if err := cmd.Wait(); err != nil {
			// Get the error from stderrBuffer
			if stderrBuffer.Len() > 0 {
				chanErr <- fmt.Errorf("DuckDB command finished with error: %v, stderr: %s", err, stderrBuffer.String())
			} else {
				chanErr <- fmt.Errorf("DuckDB command finished with error: %v", err)
			}
		}
	}()

	return res, chanErr
}
