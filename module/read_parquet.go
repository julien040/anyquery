package module

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/edsrzf/mmap-go"
	"github.com/gammazero/deque"
	"github.com/mattn/go-sqlite3"
	"github.com/parquet-go/parquet-go"
)

type ParquetModule struct {
}

type ParquetTable struct {
	mmap   mmap.MMap
	column map[int]parquetColumn
}

type ParquetCursor struct {
	column    map[int]parquetColumn
	reader    *parquet.GenericReader[any]
	rowBuffer *deque.Deque[map[string]interface{}]
	rowID     int64

	// If cursor.EOF() must return true
	eof bool

	// If the parquet file is exhausted, yet we didn't return all the rows to the user
	noMoreRows bool
}

type parquetColumn struct {
	Name      string
	Type      string
	SubFields map[string]parquetColumn
}

const rowToRequestPerBatch = 16

func (m *ParquetModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (v *ParquetModule) DestroyModule() {}

func (m *ParquetModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {

	// Parse the arguments
	fileName := ""
	if len(args) > 3 {
		fileName = strings.Trim(args[3], "' \"")
	}

	params := []argParam{
		{"file", &fileName},
		{"file_name", &fileName},
		{"filename", &fileName},
		{"src", &fileName},
		{"path", &fileName},
		{"file_path", &fileName},
		{"filepath", &fileName},
		{"url", &fileName},
	}

	parseArgs(params, args)

	// Open the file
	if fileName == "" {
		return nil, fmt.Errorf("missing file to open. Specify it with SELECT * FROM read_parquet('file.parquet')")
	}

	// Open the file
	var mmap mmap.MMap
	var err error

	mmap, err = openMmapedFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open the file: %s", err)
	}

	byteReader := bytes.NewReader(mmap)

	// Read the parquet file
	reader := parquet.NewGenericReader[any](byteReader)

	column := make(map[int]parquetColumn)

	sqlSchema := strings.Builder{}
	sqlSchema.WriteString("CREATE TABLE parquet (")
	for i, field := range reader.Schema().Fields() {
		if i > 0 {
			sqlSchema.WriteString(", ")
		}
		sqlSchema.WriteRune('"')
		sqlSchema.WriteString(transformSQLiteValidName(field.Name()))
		sqlSchema.WriteRune('"')
		sqlSchema.WriteString(" ")
		switch field.Type().String() {
		case "BOOLEAN":
			sqlSchema.WriteString("INTEGER")
		case "INT32", "INT64", "INT96", "INT(64,true)", "INT(64,false)", "INT(96,true)", "INT(96,false)", "DATE":
			sqlSchema.WriteString("INTEGER")
		case "FLOAT", "DOUBLE":
			sqlSchema.WriteString("REAL")
		case "BYTE_ARRAY", "FIXED_LEN_BYTE_ARRAY", "STRING":
			sqlSchema.WriteString("TEXT")
		default:
			sqlSchema.WriteString("TEXT")
		}

		// Save the column name
		col := parquetColumn{
			Name: field.Name(),
			Type: field.Type().String(),
		}

		// Get subfields if the field is a group
		if field.Type().String() == "group" {
			col.SubFields = make(map[string]parquetColumn)
			for _, subField := range field.Fields() {
				col.SubFields[subField.Name()] = parquetColumn{
					Name: subField.Name(),
					Type: subField.Type().String(),
				}
			}
		}

		// Save the column in the map
		column[i] = col
	}

	sqlSchema.WriteString(");")
	c.DeclareVTab(sqlSchema.String())

	return &ParquetTable{mmap: mmap, column: column}, nil
}

func (t *ParquetTable) Open() (sqlite3.VTabCursor, error) {
	// Create a new reader
	reader := parquet.NewGenericReader[any](bytes.NewReader(t.mmap))

	return &ParquetCursor{
		column:    t.column,
		reader:    reader,
		rowBuffer: new(deque.Deque[map[string]interface{}]),
	}, nil
}

func (t *ParquetTable) Disconnect() error {
	// Close the file if it was opened
	if t.mmap != nil {
		return t.mmap.Unmap()
	}
	return nil
}

func (t *ParquetTable) Destroy() error {
	return nil
}

func (t *ParquetTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		IdxNum: 1,
		Used:   make([]bool, len(cst)),
	}, nil
}

func (t *ParquetCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	return t.requestRows()
}

func (t *ParquetCursor) requestRows() error {
	buffer := make([]any, rowToRequestPerBatch)
	rowFound, err := t.reader.Read(buffer)
	if err == io.EOF {
		t.noMoreRows = true
	} else if err != nil {
		return err
	}
	for i := 0; i < rowFound; i++ {
		if mapVal, ok := buffer[i].(map[string]interface{}); ok {
			t.rowBuffer.PushBack(mapVal)
		}
	}

	return nil
}

func (t *ParquetCursor) Next() error {
	if t.rowBuffer.Len() != 0 {
		t.rowBuffer.PopFront()
	}
	if t.rowBuffer.Len() == 0 {
		if t.noMoreRows {
			t.eof = true
			return nil
		}
		err := t.requestRows()
		if err != nil {
			return err
		}
	}

	if t.rowBuffer.Len() == 0 && t.noMoreRows {
		t.eof = true
		return nil
	}

	t.rowID++
	return nil
}

func (t *ParquetCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	colName, ok := t.column[col]
	if !ok {
		context.ResultNull()
		return nil
	}
	val, ok := t.rowBuffer.Front()[colName.Name]
	if !ok {
		context.ResultNull()
		return nil
	}

	switch valParsed := val.(type) {
	case bool:
		if valParsed {
			context.ResultInt(1)
		} else {
			context.ResultInt(0)
		}
	case int:
		context.ResultInt(valParsed)
	case int8:
		context.ResultInt(int(valParsed))
	case int16:
		context.ResultInt(int(valParsed))
	case int32:
		context.ResultInt(int(valParsed))
	case int64:
		context.ResultInt64(valParsed)
	case uint64:
		context.ResultInt64(int64(valParsed))
	case float32:
		context.ResultDouble(float64(valParsed))
	case float64:
		context.ResultDouble(valParsed)
	case string:
		// parquet-go returns BYTE_ARRAY as string resulting in UTF-8 issues
		// When we detect a column that has a BYTE_ARRAY type, we will convert it to a byte slice
		if colName.Type == "BYTE_ARRAY" || colName.Type == "FIXED_LEN_BYTE_ARRAY" {
			// Convert the string to a byte slice
			context.ResultBlob([]byte(valParsed))
		} else {
			context.ResultText(valParsed)
		}
	case []byte:
		context.ResultBlob(valParsed)
	case map[string]interface{}:
		for key, value := range valParsed {

			// Get the subfield type if it exists
			subFieldType, ok := colName.SubFields[key]
			if !ok {
				continue
			}

			// Same as the string case, we need to handle BYTE_ARRAY and FIXED_LEN_BYTE_ARRAY
			if subFieldType.Type == "BYTE_ARRAY" || subFieldType.Type == "FIXED_LEN_BYTE_ARRAY" {
				if strValue, ok := value.(string); ok {
					valParsed[key] = []byte(strValue)
				} else if byteValue, ok := value.([]byte); ok {
					valParsed[key] = byteValue
				} else {
					valParsed[key] = value
				}
			}
		}
		marshaled, err := json.Marshal(valParsed)
		if err != nil {
			context.ResultNull()
		} else {
			context.ResultText(string(marshaled))
		}
	default:
		context.ResultNull()
	}

	return nil
}

func (t *ParquetCursor) EOF() bool {
	return t.eof
}

func (t *ParquetCursor) Rowid() (int64, error) {
	return t.rowID, nil
}

func (t *ParquetCursor) Close() error {
	return t.reader.Close()
}
