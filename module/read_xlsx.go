package module

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
)

type XlsxModule struct{}

type XlsxTable struct {
	rows    [][]string
	columns []columnCsv
	file    *excelize.File
}

type XlsxCursor struct {
	rows    [][]string
	columns []columnCsv
	rowID   int64
}

func (m *XlsxModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (m *XlsxModule) DestroyModule() {}

// resolveSheet returns the sheet name to use given the user's sheet parameter.
// If sheetParam is empty, the first visible sheet is returned.
// If sheetParam is a 1-based integer string, the Nth visible sheet is returned.
// Otherwise an exact name match is tried.
func resolveSheet(f *excelize.File, sheetParam string) (string, error) {
	allSheets := f.GetSheetList()

	// Build the visible sheet list
	visible := []string{}
	for _, name := range allSheets {
		if ok, _ := f.GetSheetVisible(name); ok {
			visible = append(visible, name)
		}
	}
	if len(visible) == 0 {
		// Fall back to all sheets if every sheet is hidden
		visible = allSheets
	}

	if sheetParam == "" {
		return visible[0], nil
	}

	// Try numeric index first (1-based)
	if idx, err := strconv.Atoi(sheetParam); err == nil {
		if idx < 1 || idx > len(visible) {
			return "", fmt.Errorf("sheet index %d out of range: workbook has %d visible sheet(s)", idx, len(visible))
		}
		return visible[idx-1], nil
	}

	// Exact name match
	for _, name := range allSheets {
		if name == sheetParam {
			return name, nil
		}
	}

	return "", fmt.Errorf("sheet %q not found; available sheets: %s", sheetParam, strings.Join(allSheets, ", "))
}

// normalizeHeaders deduplicates and sanitizes a slice of raw header strings,
// returning a slice of unique, SQLite-safe column names.
func normalizeHeaders(raw []string) []string {
	seen := map[string]int{}
	out := make([]string, len(raw))
	for i, h := range raw {
		name := transformSQLiteValidName(h)
		if name == "" {
			name = fmt.Sprintf("col%d", i)
		}
		if count, ok := seen[name]; ok {
			seen[name] = count + 1
			name = fmt.Sprintf("%s_%d", name, count+1)
		} else {
			seen[name] = 1
		}
		out[i] = name
	}
	return out
}

// isEmptyRow returns true when every cell in the row is an empty string.
func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if cell != "" {
			return false
		}
	}
	return true
}

// padRow extends row to length n by appending empty strings.
func padRow(row []string, n int) []string {
	for len(row) < n {
		row = append(row, "")
	}
	return row
}

// isExcelError returns true for Excel error values like #N/A, #VALUE!, etc.
func isExcelError(s string) bool {
	return strings.HasPrefix(s, "#")
}

func (m *XlsxModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	fileName := ""
	sheetParam := ""
	headersStr := "true"
	skipStr := "0"
	rangeParam := ""

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
		{"sheet", &sheetParam},
		{"headers", &headersStr},
		{"header", &headersStr},
		{"skip", &skipStr},
		{"skip_rows", &skipStr},
		{"range", &rangeParam},
	}
	parseArgs(params, args)

	if fileName == "" {
		return nil, fmt.Errorf("missing file argument. Specify it as: SELECT * FROM read_xlsx('file=path/to/file.xlsx')")
	}

	useHeaders, _ := strconv.ParseBool(headersStr)
	skipRows, err := strconv.Atoi(skipStr)
	if err != nil || skipRows < 0 {
		skipRows = 0
	}

	// Parse optional range (e.g. "B2:F100")
	rangeMinCol, rangeMinRow, rangeMaxCol, rangeMaxRow := 0, 0, 0, 0
	hasRange := false
	if rangeParam != "" {
		parts := strings.SplitN(rangeParam, ":", 2)
		if len(parts) == 2 {
			minC, minR, err1 := excelize.CellNameToCoordinates(parts[0])
			maxC, maxR, err2 := excelize.CellNameToCoordinates(parts[1])
			if err1 == nil && err2 == nil {
				rangeMinCol, rangeMinRow = minC, minR
				rangeMaxCol, rangeMaxRow = maxC, maxR
				hasRange = true
			}
		}
		if !hasRange {
			return nil, fmt.Errorf("invalid range %q; expected format like A1:D100", rangeParam)
		}
	}

	// Open the file
	var f *excelize.File
	if fileName == "stdin" || fileName == "/dev/stdin" || fileName == "-" {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %s", err)
		}
		f, err = excelize.OpenReader(bytes.NewReader(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse Excel from stdin: %s", err)
		}
	} else {
		cachedPath, err := findCachedDestination(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve file path: %s", err)
		}
		if err := downloadFile(fileName, cachedPath, 60*60*24); err != nil {
			return nil, fmt.Errorf("failed to open file: %s", err)
		}
		f, err = excelize.OpenFile(cachedPath)
		if err != nil {
			if strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "zip") {
				return nil, fmt.Errorf("failed to open Excel file (note: legacy .xls format is not supported; convert to .xlsx first): %s", err)
			}
			return nil, fmt.Errorf("failed to open Excel file: %s", err)
		}
	}

	sheetName, err := resolveSheet(f, sheetParam)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Stream rows from the sheet
	rowIter, err := f.Rows(sheetName)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to read sheet %q: %s", sheetName, err)
	}

	// Collect all raw rows, applying range and skip filters
	var rawRows [][]string
	physicalRow := 0
	for rowIter.Next() {
		physicalRow++

		// Apply range row filter
		if hasRange && (physicalRow < rangeMinRow || physicalRow > rangeMaxRow) {
			continue
		}

		cols, err := rowIter.Columns(excelize.Options{RawCellValue: false})
		if err != nil {
			continue
		}

		// Apply range column filter
		if hasRange {
			// Trim to rangeMaxCol
			if len(cols) > rangeMaxCol {
				cols = cols[:rangeMaxCol]
			}
			// Remove leading columns before rangeMinCol (1-based → 0-based index: rangeMinCol-1)
			if rangeMinCol > 1 && len(cols) >= rangeMinCol {
				cols = cols[rangeMinCol-1:]
			} else if rangeMinCol > 1 {
				cols = []string{}
			}
		}

		rawRows = append(rawRows, cols)
	}
	rowIter.Close()

	// Apply skip
	if skipRows > len(rawRows) {
		skipRows = len(rawRows)
	}
	rawRows = rawRows[skipRows:]

	if len(rawRows) == 0 {
		f.Close()
		return nil, fmt.Errorf("no data found in sheet %q", sheetName)
	}

	// Determine headers
	var headerNames []string
	var dataRows [][]string

	if useHeaders {
		headerNames = normalizeHeaders(rawRows[0])
		dataRows = rawRows[1:]
		if len(headerNames) == 0 {
			f.Close()
			return nil, fmt.Errorf("header row is empty in sheet %q", sheetName)
		}
	} else {
		// Auto-generate column names based on the width of the first row
		headerNames = make([]string, len(rawRows[0]))
		for i := range headerNames {
			headerNames[i] = fmt.Sprintf("col%d", i)
		}
		dataRows = rawRows
	}

	numCols := len(headerNames)

	// Infer column types by scanning up to maxRowsAnalyse data rows.
	// A column is INTEGER if all non-empty cells parse as int64,
	// REAL if all non-empty cells parse as float64, TEXT otherwise.
	colTypes := make([]string, numCols)
	for i := range colTypes {
		colTypes[i] = "int" // start optimistic
	}

	sampleCount := len(dataRows)
	if sampleCount > maxRowsAnalyse {
		sampleCount = maxRowsAnalyse
	}

	for _, row := range dataRows[:sampleCount] {
		row = padRow(row, numCols)
		for i := 0; i < numCols; i++ {
			cell := row[i]
			if cell == "" || isExcelError(cell) {
				continue // empty/error cells don't affect type inference
			}
			switch colTypes[i] {
			case "int":
				if _, err := strconv.ParseInt(cell, 10, 64); err != nil {
					// Can't parse as int; try float
					if _, err2 := strconv.ParseFloat(cell, 64); err2 != nil {
						colTypes[i] = "string"
					} else {
						colTypes[i] = "float"
					}
				}
			case "float":
				if _, err := strconv.ParseFloat(cell, 64); err != nil {
					colTypes[i] = "string"
				}
			}
		}
	}

	// Build columns metadata
	columns := make([]columnCsv, numCols)
	for i, name := range headerNames {
		columns[i] = columnCsv{name: name, colType: colTypes[i]}
	}

	// Collect and normalize all data rows, skipping fully-empty ones
	var allRows [][]string
	for _, row := range dataRows {
		row = padRow(row, numCols)
		row = row[:numCols] // truncate if wider than headers
		if isEmptyRow(row) {
			continue
		}
		allRows = append(allRows, row)
	}

	// Build DeclareVTab schema
	schema := strings.Builder{}
	schema.WriteString("CREATE TABLE x(")
	for i, col := range columns {
		if i > 0 {
			schema.WriteString(", ")
		}
		schema.WriteRune('`')
		schema.WriteString(col.name)
		schema.WriteRune('`')
		schema.WriteString(" ")
		switch col.colType {
		case "int":
			schema.WriteString("INTEGER")
		case "float":
			schema.WriteString("REAL")
		default:
			schema.WriteString("TEXT")
		}
	}
	schema.WriteString(")")

	c.DeclareVTab(schema.String())

	return &XlsxTable{
		rows:    allRows,
		columns: columns,
		file:    f,
	}, nil
}

func (t *XlsxTable) Open() (sqlite3.VTabCursor, error) {
	return &XlsxCursor{
		rows:    t.rows,
		columns: t.columns,
	}, nil
}

func (t *XlsxTable) Disconnect() error {
	if t.file != nil {
		t.file.Close()
	}
	return nil
}

func (t *XlsxTable) Destroy() error {
	return nil
}

func (t *XlsxTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		Used: make([]bool, len(cst)),
	}, nil
}

func (t *XlsxCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	t.rowID = 0
	return nil
}

func (t *XlsxCursor) Next() error {
	t.rowID++
	return nil
}

func (t *XlsxCursor) EOF() bool {
	return t.rowID >= int64(len(t.rows))
}

func (t *XlsxCursor) Rowid() (int64, error) {
	return t.rowID, nil
}

func (t *XlsxCursor) Close() error {
	return nil
}

func (t *XlsxCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if t.rowID >= int64(len(t.rows)) || col >= len(t.columns) {
		context.ResultNull()
		return nil
	}

	row := t.rows[t.rowID]
	if col >= len(row) {
		context.ResultNull()
		return nil
	}

	cell := row[col]

	// Empty cell or Excel error → NULL
	if cell == "" || isExcelError(cell) {
		context.ResultNull()
		return nil
	}

	switch t.columns[col].colType {
	case "int":
		// Handle booleans returned as TRUE/FALSE by excelize
		if strings.EqualFold(cell, "true") {
			context.ResultInt(1)
		} else if strings.EqualFold(cell, "false") {
			context.ResultInt(0)
		} else if v, err := strconv.ParseInt(cell, 10, 64); err == nil {
			context.ResultInt64(v)
		} else {
			context.ResultNull()
		}
	case "float":
		if v, err := strconv.ParseFloat(cell, 64); err == nil {
			context.ResultDouble(v)
		} else {
			context.ResultNull()
		}
	default:
		context.ResultText(cell)
	}

	return nil
}
