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
	"time"

	"github.com/edsrzf/mmap-go"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
	"github.com/julien040/go-ternary"
	"vitess.io/vitess/go/vt/sqlparser"
)

type CsvModule struct {
	Restrictions *Restrictions
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
	// Get the arguments. The separator and the header are left unset on purpose:
	// an empty value means the caller said nothing about them, and what they
	// stand for is then inferred from the file itself (see sniffCSV).
	useHeader := false
	useHeaderStr := ""
	fieldSeparator := ""
	fileName := ""
	schema := ""
	// Freshness of the remote download cache, in seconds. Ignored for a local
	// file, which is never cached.
	cacheTTL := "86400"
	cacheTTLParsed := int64(86400)

	if len(args) >= 4 {
		fileName = strings.Trim(args[3], "' \"")
	}

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
		{"cache_ttl", &cacheTTL},
		{"cacheTTL", &cacheTTL},
		{"ttl", &cacheTTL},
		{"cache", &cacheTTL},
	}
	parseArgs(params, args)

	// An explicit header= wins over detection, whichever way it points.
	headerSet := useHeaderStr != ""
	if headerSet {
		useHeader, _ = strconv.ParseBool(useHeaderStr)
	}

	if cacheTTL != "" {
		var err error
		cacheTTLParsed, err = strconv.ParseInt(cacheTTL, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the cache TTL: %s", err)
		}
	}

	// Parse the separator
	separatorSet := fieldSeparator != ""
	if fieldSeparator == "tab" || fieldSeparator == "\\t" {
		fieldSeparator = "\t"
	}

	// Open the file
	if fileName == "" {
		return nil, fmt.Errorf("missing file argument. Check the validity of the arguments")
	}

	file := []byte{}
	mmap := mmap.MMap{}
	var err error
	if fileName == "/dev/stdin" || fileName == "-" || fileName == "stdin" {
		if !m.Restrictions.AllowStdin() {
			return nil, fmt.Errorf("sandbox: reading from stdin is not allowed")
		}
		// Read from stdin
		file, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %s", err)
		}
	} else {
		// Open the file and mmap it
		mmap, err = openMmapedFile(fileName, m.Restrictions, time.Duration(cacheTTLParsed)*time.Second)
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

		if createTableStmt.TableSpec == nil {
			return nil, fmt.Errorf("invalid schema provided")
		}

		if createTableStmt.TableSpec == nil {
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
		// Without a schema, the head of the file tells the columns, their types,
		// and whatever the caller left unspecified: the delimiter and whether the
		// first row is a header.
		separator := ','
		if separatorSet {
			separator = rune(fieldSeparator[0])
		}

		detectedSeparator, detectedHeader, detectedColumns :=
			sniffCSV(file, separator, separatorSet, headerSet, useHeader)
		if len(detectedColumns) == 0 {
			return nil, fmt.Errorf("failed to read the first row: the file holds no CSV record")
		}

		columns = detectedColumns
		// The cursor skips the first row of the file on this value, so a detected
		// header must not be read back as data.
		useHeader = detectedHeader
		if !separatorSet {
			fieldSeparator = string(detectedSeparator)
		}
	}

	// The separator is unset when nothing was passed and the schema branch left
	// detection out. The reader splits on a single byte of it, so it can never
	// be empty by the time the table is built.
	if fieldSeparator == "" {
		fieldSeparator = ","
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

		tableStatement.WriteString("`" + col.name + "`")
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

func (t *CsvTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
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
