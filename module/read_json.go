package module

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/edsrzf/mmap-go"
	"github.com/goccy/go-json"
	"github.com/mattn/go-sqlite3"
)

type JSONModule struct {
	fileContent []byte
	mmap        mmap.MMap
	tableShape  jsonShape
}

type JSONTable struct {
	file       mmap.MMap
	tableShape jsonShape
	columns    map[string]column
	rowCount   int
}

type JSONCursor struct {
	firstRowWritten bool
	rowWritten      int
	rowCount        int
	columns         map[string]column
}

type JSONCursorColShape struct {
}

type jsonShape int

const (
	// { "col1":[5,9,0], "str":["fkndk"] }
	//
	// Notably used by Pandas

	columnJsonShape jsonShape = iota
	// [ {"col1":5, "str":"fkndk"}, {"col1":9, "str":"fkndk"}]
	arrayJsonShape
	// Regular JSON { "col1":5, "str":"fkndk"}
	regularJsonShape
)

type column struct {
	typeCol string
	values  []interface{}
	colPos  int
}

// The number of rows to analyse to get the columns
const maxRowsAnalyse = 20

func (m *JSONModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (v *JSONModule) DestroyModule() {}

func (m *JSONModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	// Download the file
	if len(args) < 4 {
		return nil, sqlite3.ErrConstraintVTab
	}

	filepath := strings.Trim(args[3], "' \"")
	jsonPath := ""

	// Parse the args
	argsAvailable := []argParam{
		// We define the args that are available
		// and some aliases
		{
			name:  "url",
			value: &filepath,
		},
		{
			name:  "file",
			value: &filepath,
		},
		{
			name:  "path",
			value: &filepath,
		},
		{
			name:  "src",
			value: &filepath,
		},
		{
			name:  "file_path",
			value: &filepath,
		},
		{
			name:  "filepath",
			value: &filepath,
		},
		{
			name:  "jsonpath",
			value: &jsonPath,
		},
		{
			name:  "json_path",
			value: &jsonPath,
		},
	}

	parseArgs(argsAvailable, args)

	if filepath == "" {
		return nil, fmt.Errorf("no file path provided")
	}

	if filepath == "/dev/stdin" || filepath == "-" || filepath == "stdin" {
		// Read from stdin
		var err error
		m.fileContent, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
	} else {
		file, err := openMmapedFile(filepath)
		if err != nil {
			return nil, err
		}
		m.fileContent = file
		m.mmap = file
	}

	var unmarshaled interface{}

	if jsonPath != "" {
		jsonPathStruct, err := json.CreatePath(jsonPath)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON path: %s", err)
		}
		err = jsonPathStruct.Unmarshal(m.fileContent, &unmarshaled)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON path: %s", err)
		}
	} else {
		err := json.Unmarshal(m.fileContent, &unmarshaled)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON: %s", err)
		}
	}

	columns := make(map[string]column)

	rowCount := 0

	// Get the shape of the JSON
	// and get the columns from that
	switch val := unmarshaled.(type) {
	case []interface{}:
		m.tableShape = arrayJsonShape
		// Find the columns
		for i, v := range val {
			if i > maxRowsAnalyse {
				break
			}
			switch v.(type) {
			case map[string]interface{}:
				recursivelyFindCol("", columns, v.(map[string]interface{}))
			}
		}
		// Fill the columns
		i := 1
		for _, v := range val {
			if _, ok := v.(map[string]interface{}); !ok {
				continue
			}
			recursivelyFillValue("", columns, v.(map[string]interface{}))
			// We check that all columns have the same number of values
			for k, v := range columns {
				if len(v.values) < i {
					v.values = append(v.values, nil)
				}
				columns[k] = v
			}
			i++
		}
		rowCount = len(unmarshaled.([]interface{}))

	case map[string]interface{}:
		// We ensure that the JSON is in the column shape
		// which means it has a few keys and the values are arrays
		m.tableShape = columnJsonShape
		isColumnShape := true
		for _, v := range val {
			if _, ok := v.([]interface{}); !ok {
				isColumnShape = false
				break
			}
		}
		if !isColumnShape {
			m.tableShape = regularJsonShape
		}

		if m.tableShape == regularJsonShape {
			// We treat it as an array of one element
			recursivelyFindCol("", columns, val)
			recursivelyFillValue("", columns, val)
			rowCount = 1
		} else if m.tableShape == columnJsonShape {
			// Find the columns
			i := 0
			for k, v := range val {
				k = transformSQLiteValidName(k)
				if interfaceSlice, ok := v.([]interface{}); ok {
					col := column{
						colPos: i,
						values: interfaceSlice,
					}
					if len(interfaceSlice) > rowCount {
						rowCount = len(interfaceSlice)
					}
					// We iterate over the values to find the type as long as we don't have a type
					col.typeCol = "null"
					for _, val := range interfaceSlice {
						switch val.(type) {
						case bool:
							col.typeCol = "bool"
							break
						case float64:
							col.typeCol = "float64"
							break
						case string:
							col.typeCol = "string"
							break
						case nil:
							continue
						}
					}
					columns[k] = col
				}
			}

		}

	default:
		// Otherwise, we return only one column
		switch val.(type) {
		case bool:
			columns["value"] = column{
				typeCol: "bool",
				values:  []interface{}{val},
			}
		case float64:
			columns["value"] = column{
				typeCol: "float64",
				values:  []interface{}{val},
			}
		case string:
			columns["value"] = column{
				typeCol: "string",
				values:  []interface{}{val},
			}
		case nil:
			columns["value"] = column{
				typeCol: "null",
				values:  []interface{}{nil},
			}
		}

		return nil, fmt.Errorf("unsupported JSON shape")
	}

	if rowCount == 0 {
		return nil, fmt.Errorf("no rows found")
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns found")
	}

	tableDefinition := strings.Builder{}
	tableDefinition.WriteString("CREATE TABLE x(")

	i := 0
	for k, v := range columns {
		if i > 0 {
			tableDefinition.WriteString(", ")
		}
		colName := strings.ReplaceAll(k, "\x1e", ".")
		colName = strings.ReplaceAll(colName, " ", "_")
		colName = strings.ReplaceAll(colName, "-", "_")
		colName = strings.ReplaceAll(colName, "\"", "")
		tableDefinition.WriteString("`" + colName + "`")
		switch v.typeCol {
		case "bool":
			tableDefinition.WriteString(" BOOLEAN")
		case "float64":
			tableDefinition.WriteString(" REAL")
		case "string":
			tableDefinition.WriteString(" TEXT")
		case "null":
			tableDefinition.WriteString(" NULL")
		case "array":
			tableDefinition.WriteString(" TEXT")
		}
		// We store the position of the column
		v.colPos = i
		columns[k] = v
		i++
	}
	tableDefinition.WriteString(")")

	err := c.DeclareVTab(tableDefinition.String())
	if err != nil {
		return nil, err
	}

	return &JSONTable{
		file:     m.fileContent,
		columns:  columns,
		rowCount: rowCount,
	}, nil
}

func (t *JSONTable) Open() (sqlite3.VTabCursor, error) {
	return &JSONCursor{
		columns:  t.columns,
		rowCount: t.rowCount,
	}, nil
}

func (t *JSONTable) Disconnect() error {
	// Unmap the file
	if t.file != nil {
		t.file.Unmap()
	}

	// Drop the columns
	t.columns = nil
	runtime.GC()
	return nil
}

func (t *JSONTable) Destroy() error {
	t.Disconnect()
	return nil
}

func (t *JSONTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		Used:   make([]bool, len(cst)),
		IdxNum: 0,
	}, nil
}

func (t *JSONCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	return nil
}

func (t *JSONCursor) Next() error {
	t.rowWritten++
	return nil
}

func (t *JSONCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	for _, v := range t.columns {
		if v.colPos == col {
			// Check that the row is not out of bounds
			if t.rowWritten >= t.rowCount {
				context.ResultNull()
				return nil
			}

			if len(v.values) <= t.rowWritten {
				context.ResultNull()
				return nil
			}

			switch v.typeCol {
			case "bool":
				if parsed, ok := v.values[t.rowWritten].(bool); ok {
					context.ResultBool(parsed)
				} else {
					context.ResultNull()
				}
			case "float64":
				if parsed, ok := v.values[t.rowWritten].(float64); ok {
					context.ResultDouble(parsed)
				} else {
					context.ResultNull()
				}
			case "string":
				if parsed, ok := v.values[t.rowWritten].(string); ok {
					context.ResultText(parsed)
				} else {
					context.ResultNull()
				}
			case "null":
				context.ResultNull()
			case "array":
				if parsed, ok := v.values[t.rowWritten].(string); ok {
					context.ResultText(parsed)
				} else {
					context.ResultNull()
				}
			default:
				context.ResultNull()
			}
		}
	}
	return nil
}

func (t *JSONCursor) EOF() bool {
	return t.rowWritten >= t.rowCount
}

func (t *JSONCursor) Rowid() (int64, error) {
	return int64(t.rowWritten), nil
}

func (t *JSONCursor) Close() error {
	return nil
}

func recursivelyFindCol(prefix string, cols map[string]column, mapValue map[string]interface{}) {
	for k, v := range mapValue {
		// If the column already exists, we skip it
		// unless it has a null value
		if alreadyPresent, ok := cols[prefix+k]; ok {
			if alreadyPresent.typeCol != "null" {
				continue
			}
		}

		k = transformSQLiteValidName(k)

		// If the value is a map, we recursively find the columns
		switch v.(type) {
		case map[string]interface{}:
			// We use the ␞ RS character to separate the keys
			newPrefix := prefix + k + "\x1e"
			recursivelyFindCol(newPrefix, cols, v.(map[string]interface{}))
		case bool:
			cols[prefix+k] = column{
				typeCol: "bool",
				values:  []interface{}{},
			}
		case float64:
			cols[prefix+k] = column{
				typeCol: "float64",
				values:  []interface{}{},
			}
		case string:
			cols[prefix+k] = column{
				typeCol: "string",
				values:  []interface{}{},
			}
		case nil:
			cols[prefix+k] = column{
				typeCol: "null",
				values:  []interface{}{},
			}
		case []interface{}:
			cols[prefix+k] = column{
				typeCol: "array",
				values:  []interface{}{},
			}
		default:
		}

	}
}

func recursivelyFillValue(prefix string, cols map[string]column, mapValue map[string]interface{}) {
	for k, v := range mapValue {
		k = transformSQLiteValidName(k)
		switch v.(type) {
		case map[string]interface{}:
			// We use the ␞ RS character to separate the keys
			newPrefix := prefix + k + "\x1e"
			recursivelyFillValue(newPrefix, cols, v.(map[string]interface{}))
		default:
			// We fill the column with the value
			col, ok := cols[prefix+k]
			if ok {
				switch col.typeCol {
				case "bool":
					if parsed, ok := v.(bool); ok {
						col.values = append(col.values, parsed)
					} else {
						col.values = append(col.values, nil)
					}
				case "float64":
					if parsed, ok := v.(float64); ok {
						col.values = append(col.values, parsed)
					} else {
						col.values = append(col.values, nil)
					}
				case "string":
					if parsed, ok := v.(string); ok {
						col.values = append(col.values, parsed)
					} else {
						col.values = append(col.values, nil)
					}
				case "null":
					// We fill the null column with nil
					col.values = append(col.values, nil)
				case "array":
					if parsed, ok := v.([]interface{}); ok {
						marshaled, err := json.Marshal(parsed)
						if err == nil {
							col.values = append(col.values, string(marshaled))
						} else {
							col.values = append(col.values, nil)
						}
					} else {
						col.values = append(col.values, nil)
					}
				}

				cols[prefix+k] = col

			}
		}

	}
}
