package module

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/edsrzf/mmap-go"
	"github.com/mattn/go-sqlite3"
	"github.com/vjeantet/grok"
)

//go:embed template.grok
var grokTemplate string

type LogModule struct {
}

type LogTable struct {
	file        []byte
	mmap        mmap.MMap
	colPosition map[string]int
	pattern     string
	filePattern string // A file that contains the custom grok patterns
}

type LogCursor struct {
	reader      *bufio.Reader
	eof         bool
	currentRow  map[int]interface{}
	colPosition map[string]int
	parser      *grok.Grok
	rowID       int64
	pattern     string
}

func createGrokParser(filePattern string) (*grok.Grok, error) {
	// Read the template file
	patterns := map[string]string{}
	lines := strings.Split(grokTemplate, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		patterns[parts[0]] = parts[1]
	}

	parser, err := grok.NewWithConfig(&grok.Config{
		NamedCapturesOnly: true,
		Patterns:          patterns,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create the grok parser: %s", err)
	}
	if filePattern != "" {
		err = parser.AddPatternsFromPath(filePattern)
		if err != nil {
			return nil, fmt.Errorf("failed to add custom patterns: %s", err)
		}
	}

	return parser, nil
}

func (m *LogModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {

	return m.Connect(c, args)
}

func (v *LogModule) DestroyModule() {}

func (m *LogModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	pattern := "%{GREEDYDATA:log}"
	fileName := ""
	patternFile := ""

	// Parse arguments
	if len(args) >= 4 {
		fileName = strings.Trim(args[3], "' \"")
	}

	if len(args) >= 5 {
		pattern = strings.Trim(args[4], "' \"")
	}

	if len(args) >= 6 {
		patternFile = strings.Trim(args[5], "' \"")
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
		{"pattern", &pattern},
		{"regex", &pattern},
		{"regexp", &pattern},
		{"grok", &pattern},
		{"grok_pattern", &pattern},
		{"format", &pattern},
		{"log_format", &pattern},
		{"log_pattern", &pattern},
		{"pattern_file", &patternFile},
		{"patternfile", &patternFile},
		{"file_pattern", &patternFile},
		{"filepattern", &patternFile},
		{"grok_file", &patternFile},
		{"custom_pattern", &patternFile},
		{"custom_grok", &patternFile},
	}

	parseArgs(params, args)

	if fileName == "" {
		return nil, fmt.Errorf("missing file argument. Pass the file path as the first argument")
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

	// Read the first 1000 lines to determine the schema
	i := 0
	err = nil
	parser, err := createGrokParser(patternFile)
	if err != nil {
		return nil, err
	}
	colsType := map[string]string{}
	buffer := bufio.NewReader(bytes.NewReader(file))
	for i < 1000 && err == nil {
		var line []byte
		line, err = buffer.ReadBytes('\n')
		if err != nil {
			break
		}

		elems, err := parser.Parse(pattern, string(line))
		if err != nil {
			continue
		}
		for fieldName, _ := range elems {
			if _, ok := colsType[fieldName]; ok {
				continue
			}
			// Later we'll parse the type from the string
			colsType[fieldName] = "TEXT"
		}

		i++
	}

	if len(colsType) == 0 {
		return nil, fmt.Errorf("failed to determine the schema. Make sure the pattern is correct")
	}

	// Create the table
	builder := strings.Builder{}
	builder.WriteString("CREATE TABLE log (")
	i = 0
	colPosition := map[string]int{}
	for key, colType := range colsType {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("`" + key + "`")
		builder.WriteString(" ")
		builder.WriteString(colType)
		colPosition[key] = i
		i++
	}
	builder.WriteString(");")

	err = c.DeclareVTab(builder.String())

	if err != nil {
		return nil, fmt.Errorf("failed to declare the table: %s", err)
	}

	// Return the table
	return &LogTable{
		file:        file,
		mmap:        mmap,
		colPosition: colPosition,
		pattern:     pattern,
		filePattern: patternFile,
	}, nil
}

func (t *LogTable) Open() (sqlite3.VTabCursor, error) {

	// Create a new parser
	parser, err := createGrokParser(t.filePattern)
	if err != nil {
		return nil, err
	}

	return &LogCursor{
		reader:      bufio.NewReader(bytes.NewReader(t.file)),
		parser:      parser,
		eof:         false,
		colPosition: t.colPosition,
		pattern:     t.pattern,
		rowID:       0,
	}, nil
}

func (t *LogTable) Disconnect() error {
	return nil
}

func (t *LogTable) Destroy() error {
	return nil
}

func (t *LogTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		Used: make([]bool, len(cst)),
	}, nil
}

func (t *LogCursor) fillCurrentRow() error {
	// Fail safe
	if t.eof {
		return nil
	}

	// Read the next line
	line, err := t.reader.ReadBytes('\n')
	if err == io.EOF {
		t.eof = true
	} else if err != nil {
		return fmt.Errorf("failed to read the next line: %s", err)
	}

	// Parse the line
	elems, err := t.parser.Parse(t.pattern, string(line))
	if err != nil {
		return fmt.Errorf("failed to parse the line: %s", err)
	}

	// Fill the current row
	t.currentRow = map[int]interface{}{}
	for fieldName, fieldValue := range elems {
		if pos, ok := t.colPosition[fieldName]; ok {
			t.currentRow[pos] = fieldValue
		}
	}

	t.rowID++

	return nil
}

func (t *LogCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	return t.fillCurrentRow()
}

func (t *LogCursor) Next() error {
	return t.fillCurrentRow()
}

func (t *LogCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if val, ok := t.currentRow[col]; ok {
		switch identified := val.(type) {
		case string:
			context.ResultText(identified)
		case int:
			context.ResultInt(identified)
		case int64:
			context.ResultInt64(identified)
		case float64:
			context.ResultDouble(identified)
		default:
			context.ResultText(fmt.Sprintf("%v", identified))
		}
	} else {
		context.ResultNull()
	}

	return nil
}

func (t *LogCursor) EOF() bool {
	return t.eof || t.currentRow == nil
}

func (t *LogCursor) Rowid() (int64, error) {
	return t.rowID, nil
}

func (t *LogCursor) Close() error {
	return nil
}
