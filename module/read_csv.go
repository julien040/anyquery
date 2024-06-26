package module

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/edsrzf/mmap-go"
	"github.com/julien040/go-ternary"
	"github.com/mattn/go-sqlite3"
	"vitess.io/vitess/go/vt/sqlparser"
)

type CsvModule struct {
}

type CsvTable struct {
	fileOpened     *sync.Pool
	useHeader      bool
	fieldSeparator string
	file           []byte
	mmap           mmap.MMap
	columns        []columnCsv
}

type CsvCursor struct {
	useHeader bool
	tempRow   []string
	reader    *csv.Reader
	columns   []columnCsv
	eof       bool
	rowID     int64
}

type columnCsv struct {
	name    string
	colType string
}

var typeEquivalences = map[string]string{
	"integer":  "int",
	"int8":     "int",
	"long":     "int",
	"int":      "int",
	"bigint":   "int",
	"smallint": "int",
	"tinyint":  "int",
	"int16":    "int",
	"int32":    "int",
	"int64":    "int",
	"real":     "float",
	"float":    "float",
	"double":   "float",
	"float32":  "float",
	"float64":  "float",
	"decimal":  "float",
	"text":     "string",
	"string":   "string",
	"varchar":  "string",
	"char":     "string",
	"bool":     "bool",
	"boolean":  "bool",
}

func (m *CsvModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (v *CsvModule) DestroyModule() {}

var alphaNumRegexp *regexp.Regexp = regexp.MustCompile(`[^\p{L}\p{N} ]+`)

func (m *CsvModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	// Get the arguments
	useHeader := false
	useHeaderStr := ""
	fieldSeparator := ","
	fileName := ""
	schema := ""

	params := []argParam{
		{"file", &fileName},
		{"header", &useHeaderStr},
		{"headers", &useHeaderStr},
		{"separator", &fieldSeparator},
		// Alias
		{"use_header", &useHeaderStr},
		{"file_name", &fileName},
		{"filename", &fileName},
		{"src", &fileName},
		{"path", &fileName},
		{"file_path", &fileName},
		{"filepath", &fileName},
		{"url", &fileName},
		{"field_separator", &fieldSeparator},
		{"FS", &fieldSeparator},
		{"delimiter", &fieldSeparator},
		{"schema", &schema},
		{"table", &schema},
	}
	parseArgs(params, args)

	useHeader, _ = strconv.ParseBool(useHeaderStr)

	// Open the file
	if fileName == "" {
		return nil, fmt.Errorf("missing file argument. Check the validity of the arguments")
	}

	file := []byte{}
	mmap := mmap.MMap{}
	var err error
	if fileName == "/dev/stdin" || fileName == "-" || fileName == "stdin" {
		// Read from stdin
		file, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %s", err)
		}
	} else {
		// Open the file and mmap it
		mmap, err = openMmapedFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to open the file: %s", err)
		}
		file = mmap

	}

	columns := []columnCsv{}

	// Try to parse the schema
	if schema != "" {
		parser, err := sqlparser.New(sqlparser.Options{})
		if err != nil {
			return nil, fmt.Errorf("failed to create the parser: %s", err)
		}

		stmt, err := parser.Parse(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the schema: %s", err)
		}

		createTableStmt, ok := stmt.(*sqlparser.CreateTable)
		if !ok {
			return nil, fmt.Errorf("invalid schema provided")
		}

		for i, col := range createTableStmt.TableSpec.Columns {
			lowerCaseType := strings.ToLower(col.Type.Type)
			colType, ok := typeEquivalences[lowerCaseType]
			if !ok {
				return nil, fmt.Errorf("unsupported type: %s for column %s(position %d)", col.Type.Type, col.Name, i)
			}
			// Add the column
			columns = append(columns, columnCsv{
				name:    col.Name.String(),
				colType: colType,
			})
		}
	} else {
		// We read the first row to get the columns and the amount of columns
		reader := csv.NewReader(bytes.NewReader(file))
		reader.Comma = rune(fieldSeparator[0])
		row, err := reader.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to read the first row: %s", err)
		}

		fmt.Println("Header: ", row)

		for i, col := range row {
			colName := fmt.Sprintf("col%d", i)
			if useHeader {
				colName = col
			}
			columns = append(columns, columnCsv{
				name:    colName,
				colType: "string",
			})
		}
	}

	// Create the table
	tableStatement := strings.Builder{}
	tableStatement.WriteString("CREATE TABLE x(")
	for i, col := range columns {
		if i > 0 {
			tableStatement.WriteString(", ")
		}
		// Replace invalid characters
		col.name = transformSQLiteValidName(col.name)

		tableStatement.WriteString(col.name)
		tableStatement.WriteString(" ")
		switch col.colType {
		case "int":
			tableStatement.WriteString("INTEGER")
		case "float":
			tableStatement.WriteString("REAL")
		case "bool":
			tableStatement.WriteString("INTEGER")
		default:
			tableStatement.WriteString("TEXT")
		}
	}
	tableStatement.WriteString(")")

	c.DeclareVTab(tableStatement.String())

	return &CsvTable{
		useHeader:      useHeader,
		columns:        columns,
		mmap:           mmap,
		file:           file,
		fieldSeparator: fieldSeparator,
	}, nil
}

func (t *CsvTable) Open() (sqlite3.VTabCursor, error) {
	// Create a new reader
	reader := csv.NewReader(bytes.NewReader(t.file))
	reader.Comma = rune(t.fieldSeparator[0])
	reader.LazyQuotes = true
	reader.ReuseRecord = true

	return &CsvCursor{
		useHeader: t.useHeader,
		reader:    reader,
		columns:   t.columns,
	}, nil
}

func (t *CsvTable) Disconnect() error {
	// Unmap the file
	if t.mmap != nil {
		t.mmap.Unmap()
	}
	return nil
}

func (t *CsvTable) Destroy() error {
	return nil
}

func (t *CsvTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		Used: make([]bool, len(cst)),
	}, nil
}

func (t *CsvCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	// Skip the first row if we have a header
	if t.useHeader {
		_, err := t.reader.Read()
		if err == io.EOF {
			t.eof = true
			return nil
		} else if err != nil {
			return err
		}
	}

	t.rowID = 0
	t.eof = false
	t.Next()

	return nil
}

func (t *CsvCursor) Next() error {
	row, err := t.reader.Read()
	if err == io.EOF {
		t.eof = true
		return nil
	}
	t.tempRow = row
	t.rowID++
	return err
}

func (t *CsvCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if col >= len(t.tempRow) {
		context.ResultNull()
		return nil
	}

	switch t.columns[col].colType {
	case "int":
		val, err := strconv.ParseInt(t.tempRow[col], 10, 64)
		if err != nil {
			context.ResultNull()
		} else {
			context.ResultInt64(val)
		}
	case "float":
		val, err := strconv.ParseFloat(t.tempRow[col], 64)
		if err != nil {
			context.ResultNull()
		} else {
			context.ResultDouble(val)
		}
	case "bool":
		val, err := strconv.ParseBool(t.tempRow[col])
		if err != nil {
			context.ResultNull()
		} else {
			context.ResultInt(ternary.If(val, 1, 0))
		}
	default:
		context.ResultText(t.tempRow[col])
	}

	return nil
}

func (t *CsvCursor) EOF() bool {
	return t.eof
}

func (t *CsvCursor) Rowid() (int64, error) {
	return t.rowID, nil
}

func (t *CsvCursor) Close() error {
	return nil
}
