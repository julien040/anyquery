package controller

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// This file defines a struct that is used by commands to output tables

type outputTableType int

const (
	outputTableTypePlain outputTableType = iota
	outputTableTypeCsv
	outputTableTypeJson
	outputTableTypeJsonPretty
	outputTableTypeJsonLines
	outputTableTypePretty
	outputTableTypePlainWithHeader
	outputTableTypeMarkdown
	outputTableTypeLineByLine
	outputTableTypeHtml
)

type outputTable struct {
	// The columns of the table
	Columns []string

	// The type of the output
	Type outputTableType

	// Where the output should be written
	Writer io.Writer

	// The number of rows that have been written
	rowCount int

	// The encoder that will be used to write the output
	encoder tableEncoder
}

type tableEncoder interface {
	// Write writes a single row to the output
	Write(row []interface{}) error

	// Close flushes the output
	Close() error
}

var formatName map[string]outputTableType = map[string]outputTableType{
	"plain":       outputTableTypePlain,
	"tsv":         outputTableTypePlainWithHeader,
	"csv":         outputTableTypeCsv,
	"json":        outputTableTypeJsonPretty,
	"uglyjson":    outputTableTypeJson,
	"jsonl":       outputTableTypeJsonLines,
	"pretty":      outputTableTypePretty,
	"plainheader": outputTableTypePlainWithHeader,
	"markdown":    outputTableTypeMarkdown,
	"linebyline":  outputTableTypeLineByLine,
	"html":        outputTableTypeHtml,
}

// Create a new output table and force the output type
func newOutputTable(columns []string, outputType outputTableType, writer io.Writer) *outputTable {
	return &outputTable{
		Columns: columns,
		Type:    outputType,
		Writer:  writer,
	}
}

func (o *outputTable) SetEncoder() {
	switch o.Type {
	case outputTableTypeJson:
		o.encoder = &jsonTableEncoder{
			Columns: o.Columns,
			Writer:  o.Writer,
		}

	case outputTableTypeJsonPretty:
		o.encoder = &jsonTableEncoder{
			Columns: o.Columns,
			Writer:  o.Writer,
			Indent:  true,
		}

	case outputTableTypeJsonLines:
		o.encoder = &jsonLinesTableEncoder{
			Columns: o.Columns,
			Writer:  o.Writer,
		}

	case outputTableTypeCsv:
		o.encoder = &csvTableEncoder{
			Columns:       o.Columns,
			Writer:        o.Writer,
			ColumnHeaders: true,
			Separator:     ',',
		}

	case outputTableTypePlainWithHeader:
		o.encoder = &csvTableEncoder{
			Columns:       o.Columns,
			Writer:        o.Writer,
			ColumnHeaders: true,
			Separator:     '\t',
		}

	case outputTableTypePlain:
		o.encoder = &csvTableEncoder{
			Columns:       o.Columns,
			Writer:        o.Writer,
			ColumnHeaders: false,
			Separator:     '\t',
		}

	case outputTableTypePretty:
		o.encoder = &prettyTableEncoder{
			Columns: o.Columns,
			Writer:  o.Writer,
		}
	case outputTableTypeMarkdown:
		o.encoder = &markdownTableEncoder{
			Columns: o.Columns,
			Writer:  o.Writer,
		}
	case outputTableTypeLineByLine:
		o.encoder = &lineByLineTableEncoder{
			Columns: o.Columns,
			Writer:  o.Writer,
		}
	case outputTableTypeHtml:
		o.encoder = &htmlTableEncoder{
			Columns: o.Columns,
			Writer:  o.Writer,
		}

	default:
		// We default to plain with header if the type is unknown
		o.encoder = &csvTableEncoder{
			Columns:       o.Columns,
			Writer:        o.Writer,
			ColumnHeaders: true,
			Separator:     '\t',
		}
	}

}

func (o *outputTable) Write(row []interface{}) error {
	if o.encoder == nil {
		o.SetEncoder()
	}

	if err := o.encoder.Write(row); err != nil {
		return err
	}

	o.rowCount++

	return nil
}

func (o *outputTable) AddRow(row ...interface{}) {
	o.Write(row)
}

func (o *outputTable) WriteRows(rows [][]interface{}) error {
	for _, row := range rows {
		if err := o.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// Convert a database/sql.Rows to the output format
// and close the rows
func (o *outputTable) WriteSQLRows(rows *sql.Rows) error {

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	o.Columns = cols

	// Store the scannedValues of the rows
	scannedValues := make([]interface{}, len(cols))
	for i := range scannedValues {
		var v interface{}
		scannedValues[i] = &v
	}

	// Scan the rows
	for rows.Next() {
		if err := rows.Scan(scannedValues...); err != nil {
			return err
		}

		strRow := make([]interface{}, len(scannedValues))

		for i, val := range scannedValues {
			_, ok := val.(*interface{})
			if !ok {
				strRow[i] = fmt.Sprintf("%v", val)
			}

			parsed := *(val.(*interface{}))
			strRow[i] = parsed
		}

		if err := o.Write(strRow); err != nil {
			return err
		}

	}
	if rows.Err() != nil {
		return rows.Err()
	}

	// Close the rows
	if err := rows.Close(); err != nil {
		return err
	}

	return nil
}

// InferFlags scans the flags and modifies the output configuration accordingly
func (o *outputTable) InferFlags(flag *pflag.FlagSet) {
	// Check if io.writer is a file
	// If so, check if it's a tty
	// Default to the corresponding output type unless one is already set by the caller
	// We need to do that because if a cmd set a default output type, we don't want to override it
	if file, ok := o.Writer.(*os.File); ok && o.Type == 0 {
		if !term.IsTerminal(int(file.Fd())) {
			o.Type = outputTableTypePlain
		} else {
			// If the output is a tty, we default to pretty
			o.Type = outputTableTypePretty
		}
	}

	// Check if the special flags --json, --csv and --plain are set
	if json, _ := flag.GetBool("json"); json {
		o.Type = outputTableTypeJsonPretty
		return
	}

	if csv, _ := flag.GetBool("csv"); csv {
		o.Type = outputTableTypeCsv
		return
	}

	if plain, _ := flag.GetBool("plain"); plain {
		o.Type = outputTableTypePlain
		return
	}

	// Finally, iterate over the format flags
	// If one is set, we set the output type to the corresponding type
	format, err := flag.GetString("format")
	if err == nil {
		if val, ok := formatName[format]; ok {
			o.Type = val
		}
	}

}

// Close flushes the output
func (o *outputTable) Close() error {
	if o.encoder == nil {
		o.SetEncoder()
	}

	return o.encoder.Close()
}

/**********************
 * JSON TABLE ENCODER *
 **********************/

type jsonTableEncoder struct {
	Columns         []string
	Writer          io.Writer
	encoder         *json.Encoder
	Indent          bool
	firstRowWritten bool
}

func (j *jsonTableEncoder) Write(row []interface{}) error {
	if j.encoder == nil {
		j.encoder = json.NewEncoder(j.Writer)
		if j.Indent {
			j.encoder.SetIndent("  ", "  ")
		}
		fmt.Fprint(j.Writer, "[")
		if j.Indent {
			fmt.Fprint(j.Writer, "\n")
		}
	}

	// To separate the rows with a comma
	if j.firstRowWritten {
		fmt.Fprint(j.Writer, " ,")
	} else {
		if j.Indent {
			fmt.Fprint(j.Writer, "  ")
		}
		j.firstRowWritten = true
	}
	mapToPrint := make(map[string]interface{})
	for i, col := range j.Columns {
		if i < len(row) {
			// Sometimes, a column might be returned as a string of its JSON representation
			// We will check if row[i] is a string and if it is, we will check if it's starting with a { or a [
			if str, ok := row[i].(string); ok {
				if strings.HasPrefix(str, "{") || strings.HasPrefix(str, "[") {
					var v interface{}
					if err := json.Unmarshal([]byte(str), &v); err == nil {
						mapToPrint[col] = v
						continue
					}
				}
			}
			mapToPrint[col] = row[i]
		} else { // When the row is shorter than the columns, we set the column to nil
			mapToPrint[col] = nil
		}
	}

	j.encoder.Encode(mapToPrint)

	return nil

}
func (j *jsonTableEncoder) Close() error {
	// Print the closing bracket
	if j.firstRowWritten {
		fmt.Fprint(j.Writer, "]\n")
	} else {
		fmt.Fprint(j.Writer, "[]\n")
	}

	return nil
}

/*****************
 * JSONL ENCODER *
 *****************/

type jsonLinesTableEncoder struct {
	Columns []string
	Writer  io.Writer
	encoder *json.Encoder
}

func (j *jsonLinesTableEncoder) Write(row []interface{}) error {
	if j.encoder == nil {
		j.encoder = json.NewEncoder(j.Writer)
	}

	mapToAppend := make(map[string]interface{})
	for i, col := range j.Columns {
		if i < len(row) {
			// Sometimes, a column might be returned as a string of its JSON representation
			// We will check if row[i] is a string and if it is, we will check if it's starting with a { or a [
			if str, ok := row[i].(string); ok {
				if strings.HasPrefix(str, "{") || strings.HasPrefix(str, "[") {
					var v interface{}
					if err := json.Unmarshal([]byte(str), &v); err == nil {
						mapToAppend[col] = v
						continue
					}
				}
			}
			mapToAppend[col] = row[i]
		} else { // When the row is shorter than the columns, we set the column to nil
			mapToAppend[col] = nil
		}
	}

	return j.encoder.Encode(mapToAppend)
}

func (j *jsonLinesTableEncoder) Close() error {
	return nil
}

/***************
 * CSV ENCODER *
 ***************/

type csvTableEncoder struct {
	Columns       []string
	Writer        io.Writer
	ColumnHeaders bool
	Separator     rune

	// Whether the columns have already been written at the start
	columnWritten bool

	// The encoder that will be used to write the output
	encoder *csv.Writer
}

func (c *csvTableEncoder) Write(row []interface{}) error {
	if c.encoder == nil {
		c.encoder = csv.NewWriter(c.Writer)
		c.encoder.Comma = c.Separator
	}

	// Write the columns if they haven't been written yet
	if !c.columnWritten {
		if c.ColumnHeaders {
			if err := c.encoder.Write(c.Columns); err != nil {
				return err
			}
		}
		c.columnWritten = true
	}

	// Convert the row to strings
	rowStr := convertValueToStrSlice(row)

	// Write the row
	if err := c.encoder.Write(rowStr); err != nil {
		return err
	}

	return nil

}

func convertValueToStrSlice(row []interface{}) []string {
	rowStr := make([]string, len(row))

	for i, val := range row {
		switch v := val.(type) {
		case nil:
			rowStr[i] = ""
		case sql.NullString:
			if v.Valid {
				rowStr[i] = v.String
			} else {
				rowStr[i] = ""
			}
		case sql.NullInt64:
			if v.Valid {
				rowStr[i] = fmt.Sprintf("%d", v.Int64)
			} else {
				rowStr[i] = ""
			}
		case sql.NullFloat64:
			if v.Valid {
				rowStr[i] = fmt.Sprintf("%f", v.Float64)
			} else {
				rowStr[i] = ""
			}
		case sql.NullBool:
			if v.Valid {
				if v.Bool {
					rowStr[i] = "true"
				} else {
					rowStr[i] = "false"
				}
			} else {
				rowStr[i] = ""
			}
		case sql.NullTime:
			if v.Valid {
				rowStr[i] = v.Time.String()
			} else {
				rowStr[i] = ""
			}
		case string:
			rowStr[i] = v
		case int, int8, int16, int32, int64:
			rowStr[i] = fmt.Sprintf("%d", v)
		case float32, float64:
			rowStr[i] = fmt.Sprintf("%f", v)
		case bool:
			if v {
				rowStr[i] = "true"
			} else {
				rowStr[i] = "false"
			}
		default:
			rowStr[i] = fmt.Sprintf("%v", v)
		}

	}
	return rowStr
}

func (c *csvTableEncoder) Close() error {
	if c.encoder == nil {
		c.encoder = csv.NewWriter(c.Writer)
		c.encoder.Comma = c.Separator
	}

	// Write the columns if they haven't been written yet
	if !c.columnWritten {
		if c.ColumnHeaders {
			if err := c.encoder.Write(c.Columns); err != nil {
				return err
			}
		}
		c.columnWritten = true
	}

	if c.encoder != nil {
		c.encoder.Flush()
	}

	return nil
}

/******************
 * PRETTY ENCODER *
 ******************/

type prettyTableEncoder struct {
	Columns       []string           // The columns of the table (may be mutated during setup to drop columns)
	Writer        io.Writer          // The writer where the output will be written
	internalTable *tablewriter.Table // The table that will be rendered
	rowWritten    int                // How many rows have been written
	columnLength  int                // The width of a column slot (content + padding), excluding borders
	contentLength int                // columnLength minus the 2 padding chars
	numOrigKept   int                // How many original columns survive (before the "..." sentinel)
}

// computeColumnLength derives the width of a single column slot (content + its
// 2 chars of padding) so that numCols slots plus the numCols+1 border
// characters between/around them add up to the terminal width.
func computeColumnLength(terminalWidth, numCols int) int {
	numCols = max(1, numCols)
	return (terminalWidth - numCols - 1) / numCols
}

// setup determines column widths, drops columns that would be too narrow, and
// starts the streaming table. It is called lazily on the first Write so that
// any terminal resize between construction and first output is captured.
func (p *prettyTableEncoder) setup() error {
	p.columnLength = 40
	terminalWidth := 0
	if f, ok := p.Writer.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		var err error
		terminalWidth, _, err = term.GetSize(int(f.Fd()))
		if err == nil {
			p.columnLength = computeColumnLength(terminalWidth, len(p.Columns))
		}
	}

	// Drop trailing columns until every remaining column is at least 12 chars wide,
	// keeping a minimum of 2 columns.
	dropped := false
	if terminalWidth > 0 {
		for len(p.Columns) > 2 && p.columnLength < 12 {
			p.Columns = p.Columns[:len(p.Columns)-1]
			p.columnLength = computeColumnLength(terminalWidth, len(p.Columns))
			dropped = true
		}
	}

	p.numOrigKept = len(p.Columns)

	if dropped {
		p.Columns = append(p.Columns, "...")
		if terminalWidth > 0 {
			p.columnLength = computeColumnLength(terminalWidth, len(p.Columns))
		}
	}

	if p.columnLength < 1 {
		p.columnLength = 1
	}
	p.contentLength = max(1, p.columnLength-2)

	widths := tw.NewMapper[int, int]()
	for i := range p.Columns {
		widths.Set(i, p.columnLength)
	}

	p.internalTable = tablewriter.NewTable(p.Writer,
		tablewriter.WithColumnWidths(widths),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithStreaming(tw.StreamConfig{Enable: true}),
	)
	if err := p.internalTable.Start(); err != nil {
		return err
	}
	p.internalTable.Header(p.Columns)
	return nil
}

func (p *prettyTableEncoder) Write(row []interface{}) error {
	if p.internalTable == nil {
		if err := p.setup(); err != nil {
			return err
		}
	}

	rowStr := convertValueToStrSlice(row)

	// Trim or pad to the number of original columns kept, then append "..." if needed.
	if len(rowStr) > p.numOrigKept {
		rowStr = rowStr[:p.numOrigKept]
	}
	for len(rowStr) < p.numOrigKept {
		rowStr = append(rowStr, "")
	}
	if p.numOrigKept < len(p.Columns) {
		rowStr = append(rowStr, "...")
	}

	for i, val := range rowStr {
		rowStr[i] = wordWrap(val, p.contentLength)
	}

	p.internalTable.Append(rowStr)
	p.rowWritten++
	return nil
}

func (p *prettyTableEncoder) Close() error {
	if p.internalTable == nil {
		if err := p.setup(); err != nil {
			return err
		}
	}
	if err := p.internalTable.Close(); err != nil {
		return err
	}
	fmt.Fprintf(p.Writer, "%d results\n", p.rowWritten)
	return nil
}

type markdownTableEncoder struct {
	Columns       []string
	Writer        io.Writer
	internalTable *tablewriter.Table
}

func (m *markdownTableEncoder) init() error {
	m.internalTable = tablewriter.NewTable(m.Writer,
		tablewriter.WithRenderer(renderer.NewMarkdown()),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
		tablewriter.WithStreaming(tw.StreamConfig{Enable: true}),
	)
	if err := m.internalTable.Start(); err != nil {
		return err
	}
	m.internalTable.Header(m.Columns)
	return nil
}

func (m *markdownTableEncoder) Write(row []interface{}) error {
	if m.internalTable == nil {
		if err := m.init(); err != nil {
			return err
		}
	}

	rowStr := convertValueToStrSlice(row)
	for i, val := range rowStr {
		rowStr[i] = strings.ReplaceAll(val, "\n", "<br>")
	}
	m.internalTable.Append(rowStr)
	return nil
}

func (m *markdownTableEncoder) Close() error {
	if m.internalTable == nil {
		if err := m.init(); err != nil {
			return err
		}
	}
	return m.internalTable.Close()
}

// Create a new line every lengthLine characters
//
// It can wrap on spaces, newlines, commas, semi-colons, colons, and slashes
func wordWrap(s string, lengthLine int) string {
	if len(s) <= lengthLine {
		return s
	}

	// We will split the string on spaces, newlines, commas, semi-colons, colons, and slashes
	builder := strings.Builder{}

	currentLineLength := 0
	i := 0
	for i < len(s) {

		switch s[i] {
		case ' ', '\n', ',', ';', ':', '/':
			// We add -5 to take the opportunity to break the line earlier instead of never
			if currentLineLength >= lengthLine-5 {
				if s[i] != '\n' {
					builder.WriteRune('\n')
				}
				currentLineLength = 0
			}
			builder.WriteByte(s[i])
		default:
			// If we don't have a special character,
			// and we are way over the line length, we will break the line
			if currentLineLength >= lengthLine {
				builder.WriteRune('\n')
				currentLineLength = 0
			}
			builder.WriteByte(s[i])

		}
		currentLineLength++
		i++
	}

	return builder.String()
}

/**********************
 * LINEBYLINE ENCODER *
 **********************/

type lineByLineTableEncoder struct {
	Columns         []string
	Writer          io.Writer
	firstRowWritten bool
}

func (l *lineByLineTableEncoder) Write(row []interface{}) error {
	if !l.firstRowWritten {
		l.firstRowWritten = true
	} else {
		// Add a separator between rows
		fmt.Fprintln(l.Writer, "---")
	}
	for i, val := range row {
		fmt.Fprintf(l.Writer, "%s: %v\n", l.Columns[i], val)
	}
	return nil
}

func (l *lineByLineTableEncoder) Close() error {
	return nil
}

type htmlTableEncoder struct {
	Columns         []string
	Writer          io.Writer
	firstRowWritten bool
}

func (h *htmlTableEncoder) Write(row []interface{}) error {
	if !h.firstRowWritten {
		h.firstRowWritten = true

		// Write the header
		fmt.Fprintln(h.Writer, "<table>")
		fmt.Fprintln(h.Writer, "    <thead>")
		fmt.Fprintln(h.Writer, "    <tr>")
		for _, col := range h.Columns {
			fmt.Fprintf(h.Writer, "        <th>%s</th>\n", html.EscapeString(col))
		}
		fmt.Fprintln(h.Writer, "    </tr>")
		fmt.Fprintln(h.Writer, "    </thead>")

		fmt.Fprintln(h.Writer, "    <tbody>")
	}

	// Write the row
	fmt.Fprintln(h.Writer, "    <tr>")
	toPrint := convertValueToStrSlice(row)
	for _, val := range toPrint {
		fmt.Fprintf(h.Writer, "        <td>%s</td>\n", html.EscapeString(val))
	}
	fmt.Fprintln(h.Writer, "    </tr>")
	return nil
}

func (h *htmlTableEncoder) Close() error {
	if h.firstRowWritten {
		fmt.Fprintln(h.Writer, "\t</tbody>")
		fmt.Fprintln(h.Writer, "</table>")
	}
	return nil
}
