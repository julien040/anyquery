package module

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/edsrzf/mmap-go"
	"github.com/goccy/go-json"
	"github.com/mattn/go-sqlite3"
)

type JSONlModule struct {
}

type JSONlTable struct {
	fileContent []byte
	mmap        mmap.MMap
	colPosition map[int]string
}

type JSONlCursor struct {
	colPosition map[int]string
	rowID       int64
	reader      *json.Decoder
	tempRow     map[string]interface{}
	eof         bool
}

func (m *JSONlModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (v *JSONlModule) DestroyModule() {}

func findCols(value map[string]interface{}, prefix string, cols map[string]string) {
	for k, v := range value {
		// Replace the special characters
		k = transformSQLiteValidName(k)
		switch v.(type) {
		case map[string]interface{}:
			findCols(v.(map[string]interface{}), prefix+k+".", cols)
		default:
			switch v.(type) {
			case string:
				cols[prefix+k] = "TEXT"
			case int:
				cols[prefix+k] = "INT"
			case float64:
				cols[prefix+k] = "FLOAT"
			case bool:
				cols[prefix+k] = "INT"
			default:
				cols[prefix+k] = "TEXT"
			}
		}
	}
}

func (m *JSONlModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	// Download the file
	if len(args) < 4 {
		return nil, sqlite3.ErrConstraintVTab
	}

	fileName := strings.Trim(args[3], "' \"")

	// Parse the args
	argsAvailable := []argParam{
		{"file", &fileName},
		{"file_name", &fileName},
		{"filename", &fileName},
		{"src", &fileName},
		{"path", &fileName},
		{"file_path", &fileName},
		{"filepath", &fileName},
		{"url", &fileName},
	}
	parseArgs(argsAvailable, args)

	// Open the file
	if fileName == "" {
		return nil, fmt.Errorf("missing file argument. Check the validity of the arguments")
	}

	fileContent := []byte{}
	mmap := mmap.MMap{}
	var err error

	if fileName == "/dev/stdin" || fileName == "-" || fileName == "stdin" {
		fileContent, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %s", err)
		}
	} else {
		file, err := openMmapedFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %s", err)
		}
		mmap = file
		fileContent = file

	}

	if len(fileContent) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	// Map a column name to a type
	mapColnameType := map[string]string{}

	mapColPositionName := map[int]string{}

	// Read the maxRowsAnalyse first values to get the columns
	i := 0
	var tempValInterface interface{}
	jsonReader := json.NewDecoder(bytes.NewReader(fileContent))
	for {
		if i >= maxRowsAnalyse {
			break
		}
		err = jsonReader.Decode(&tempValInterface)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read JSON at iteration %d: %s", i, err)
		}

		switch tempValInterface.(type) {
		case map[string]interface{}:
			findCols(tempValInterface.(map[string]interface{}), "", mapColnameType)
		default:
			// If the row is not an object, we continue to the next row
			continue
		}

		i++
	}
	if len(mapColnameType) == 0 {
		return nil, fmt.Errorf("no column found in the JSON file")
	}

	// Define an order for the columns
	// and define the schema at the same time
	schema := strings.Builder{}
	schema.WriteString("CREATE TABLE x(")
	i = 0
	for k := range mapColnameType {
		if i > 0 {
			schema.WriteString(", ")
		}
		schema.WriteRune('`')
		schema.WriteString(k)
		schema.WriteRune('`')
		schema.WriteString(" ")
		schema.WriteString(mapColnameType[k])
		mapColPositionName[i] = k
		i++
	}

	schema.WriteString(")")
	c.DeclareVTab(schema.String())

	return &JSONlTable{
		mmap:        mmap,
		colPosition: mapColPositionName,
		fileContent: fileContent,
	}, nil

}

func (t *JSONlTable) Open() (sqlite3.VTabCursor, error) {
	return &JSONlCursor{
		colPosition: t.colPosition,
		reader:      json.NewDecoder(bytes.NewReader(t.fileContent)),
	}, nil
}

func (t *JSONlTable) Disconnect() error {
	return nil
}

func (t *JSONlTable) Destroy() error {
	return nil
}

func (t *JSONlTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		Used: make([]bool, len(cst)),
	}, nil
}

func findValuesJSONl(currentValue map[string]interface{}, prefix string, tempRow map[string]interface{}) {
	for k, v := range currentValue {
		switch v.(type) {
		case map[string]interface{}:
			findValuesJSONl(v.(map[string]interface{}), prefix+k+".", tempRow)
		default:
			tempRow[prefix+k] = v
		}
	}

}

// Read the next value, fill the temp buffer
// and set eof to true if there is no more value
func (t *JSONlCursor) fillTempBuffer() error {
	if t.eof {
		return nil
	}

	t.tempRow = map[string]interface{}{}
	var mapVal interface{}
	err := t.reader.Decode(&mapVal)
	if err == io.EOF {
		t.eof = true
	} else if err != nil {
		return fmt.Errorf("failed to read JSON: %s", err)
	}
	switch mapVal.(type) {
	case map[string]interface{}:
		findValuesJSONl(mapVal.(map[string]interface{}), "", t.tempRow)
	default:
		// If the row is not an object, we continue to the next row
		return t.fillTempBuffer()
	}

	return nil

}

func (t *JSONlCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	t.rowID = 0
	t.eof = false
	return t.fillTempBuffer()
}

func (t *JSONlCursor) Next() error {
	t.rowID++
	return t.fillTempBuffer()
}

func (t *JSONlCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	// Find the column name
	colName, ok := t.colPosition[col]
	if !ok {
		context.ResultNull()
	}

	// Find the value
	val, ok := t.tempRow[colName]
	if !ok {
		context.ResultNull()
	}
	switch val.(type) {
	case string:
		context.ResultText(val.(string))
	case int: // Should not happen
		context.ResultInt(val.(int))
	case float64:
		context.ResultDouble(val.(float64))
	case bool:
		if val.(bool) {
			context.ResultInt(1)
		} else {
			context.ResultInt(0)
		}
	default:
		// We print the JSON representation of the value
		jsonRep, err := json.Marshal(val)
		if err != nil {
			context.ResultNull()
		} else {
			context.ResultText(string(jsonRep))
		}
	}
	return nil
}

func (t *JSONlCursor) EOF() bool {
	return t.eof
}

func (t *JSONlCursor) Rowid() (int64, error) {
	return t.rowID, nil
}

func (t *JSONlCursor) Close() error {
	return nil
}
