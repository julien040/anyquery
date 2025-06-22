package duckdb

import (
	"bufio"
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
)

// rows implements the driver.Rows interface.
type rows struct {
	cols         []string
	firstRow     []driver.Value // Store the first row to determine columns
	file         *os.File
	fileReader   *bufio.Reader
	stillRunning chan error
}

func (r *rows) computeDriverValues(line map[string]interface{}, dest []driver.Value) {
	for i, col := range r.cols {
		if val, ok := line[col]; ok {
			dest[i] = val
		} else {
			dest[i] = nil // If the column is missing, set it to nil.
		}
	}
}

// Columns returns the names of the columns.
func (r *rows) Columns() []string {
	fmt.Println("Returning columns:", r.cols)
	if r.cols == nil {
		fmt.Println("Columns are nil, reading first line to determine columns")
		// If cols is nil, we need to read the first line to get the columns
		line, err := r.fileReader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			fmt.Printf("Error reading first line for columns: %v\n", err)
			return nil
		}
		if err == io.EOF {
			fmt.Println("No data found in file, returning nil columns")
			return nil
		}

		fmt.Printf("Reading column names from first line: %s\n", string(line))
		var unknown map[string]interface{}
		if err := json.Unmarshal(line, &unknown); err != nil {
			fmt.Printf("Error unmarshalling first line for columns: %v (%s)\n", err, string(line))
			return nil
		}
		r.cols = make([]string, 0, len(unknown))

		// While a JSON object has keys in an undefined order, duckdb returns them in the order they were defined.
		// We will read the json object and extract the keys by using a JSON decoder.
		//
		// We read every token in the JSON object, expecting the first token to be an object start '{'.
		tokensParser := json.NewDecoder(bytes.NewReader(line))
		for {
			tok, err := tokensParser.Token()
			if err != nil {
				if err == io.EOF {
					break // End of the JSON object
				}
				fmt.Printf("Error reading tokens for columns: %v\n", err)
				return nil
			}
			if key, ok := tok.(string); ok {
				// The next token should be the value, we skip it
				if _, err := tokensParser.Token(); err != nil {
					fmt.Printf("Error skipping value for column %s: %v\n", key, err)
					return nil
				}
				r.cols = append(r.cols, key) // Add the column name to cols
			} else {
				fmt.Printf("Unexpected token type for column name: %T\n", tok)
			}

		}
	}
	fmt.Printf("Columns determined: %v\n", r.cols)
	return r.cols
}

// Close closes the rows iterator.
func (r *rows) Close() error {
	fmt.Println("Closing rows")
	// Cleanup resources if necessary.

	if r.stillRunning != nil {
		close(r.stillRunning)
		r.stillRunning = nil
	}
	// Delete the temporary file if it exists.
	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return fmt.Errorf("failed to close file: %v", err)
		}
		if err := os.Remove(r.file.Name()); err != nil {
			return fmt.Errorf("failed to remove file: %v", err)
		}
		r.file = nil
	}
	fmt.Println("Rows closed successfully")
	return nil
}

var regexEndOfResults = regexp.MustCompile(`changes:\s*\d+\s*total_changes:\s*\d+`)

// Next copies the next row's column values into dest.
// It returns io.EOF when there are no more rows.
func (r *rows) Next(dest []driver.Value) error {
	fmt.Println("Next called on rows")
	select {
	case err := <-r.stillRunning:
		fmt.Println("Received stillRunning error:", err)
		if err != nil {
			return err
		} else {
			r.stillRunning = nil
			return io.EOF
		}
	default:
		fmt.Println("No stillRunning error, continuing to read rows")
		// Check if cols are set, if not, read the first line to get them.
		if r.cols == nil {
			r.Columns() // This will read the first line and set r.cols
		}

		if r.firstRow != nil {
			// If we have a first row, we can use it directly.
			if len(dest) < len(r.firstRow) {
				return fmt.Errorf("destination slice is too small, expected %d, got %d", len(r.firstRow), len(dest))
			}
			copy(r.firstRow, dest) // Copy the first row values to dest
			r.firstRow = nil       // Clear the first row after using it
			return nil
		}

		// We read the next line from the file.
		fmt.Println("Reading next line from file")
		line, err := r.fileReader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		if len(line) == 0 {
			return nil // No more lines to read
		}

		// Check if the line indicates the end of results.
		if regexEndOfResults.Match(line) {
			fmt.Println("End of results detected, closing rows")
			return io.EOF
		}

		// Decode the JSON line into a map.
		fmt.Printf("Read line: %s\n", string(line))
		var row map[string]interface{}
		if err := json.Unmarshal(line, &row); err != nil {
			return err
		}

		fmt.Printf("Decoded row: %v\n", row)

		// Fill the dest slice with values from the row map.
		for i, col := range r.cols {
			if val, ok := row[col]; ok {
				dest[i] = val
			} else {
				dest[i] = nil // If the column is missing, set it to nil.
			}
		}

	}

	return nil

}
