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
	"github.com/trivago/grok"
)

//go:embed template.grok
var grokTemplate string

type LogModule struct {
}

type LogTable struct {
	file        []byte
	mmap        mmap.MMap
	colPosition map[string]int
	parser      *grok.CompiledGrok
}

type LogCursor struct {
	reader      *bufio.Reader
	eof         bool
	currentRow  map[int]interface{}
	colPosition map[string]int
	parser      *grok.CompiledGrok
	rowID       int64
	pattern     string
}

func extractPatternsFromStr(grokTemplate string) map[string]string {
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
	return patterns
}

func createGrokParser(filePattern string) (*grok.Grok, error) {
	// Read the template file
	patterns := extractPatternsFromStr(grokTemplate)

	// Read the custom patterns
	if filePattern != "" {
		file, err := os.ReadFile(filePattern)
		if err != nil {
			return nil, fmt.Errorf("failed to read the custom grok patterns file: %s", err)
		}

		customPatterns := extractPatternsFromStr(string(file))
		// Overwrite the default patterns
		for key, value := range customPatterns {
			patterns[key] = value
		}
	}

	parser, err := grok.New(grok.Config{
		NamedCapturesOnly: true,
		Patterns:          patterns,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create the grok parser: %s", err)
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

	parser, err := createGrokParser(patternFile)
	if err != nil {
		return nil, err
	}

	compiledParser, err := parser.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile the pattern: %s. Make sure the pattern is a valid grok pattern", err)
	}

	// Create the table
	builder := strings.Builder{}
	builder.WriteString("CREATE TABLE log (")
	colPosition := map[string]int{}
	// We incremenent i by ourselves here because we want to skip empty fields
	// At the same time, it allows us to see if we have any fields at all
	i := 0
	for _, colName := range compiledParser.GetFields() {
		if colName == "" {
			continue
		}
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("`" + colName + "`")
		builder.WriteString(" ")
		builder.WriteString("UNKNOWN")
		colPosition[colName] = i
		i++
	}
	builder.WriteString(");")

	// Fail if no fields were found
	if i == 0 {
		return nil, fmt.Errorf("no fields found in the pattern")
	}

	err = c.DeclareVTab(builder.String())

	if err != nil {
		return nil, fmt.Errorf("failed to declare the table: %s", err)
	}

	// Return the table
	return &LogTable{
		file:        file,
		mmap:        mmap,
		colPosition: colPosition,
		parser:      compiledParser,
	}, nil
}

func (t *LogTable) Open() (sqlite3.VTabCursor, error) {
	return &LogCursor{
		reader:      bufio.NewReader(bytes.NewReader(t.file)),
		eof:         false,
		colPosition: t.colPosition,
		parser:      t.parser,
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
		t.currentRow = nil
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read the next line: %s", err)
	}

	// Parse the line
	elems, err := t.parser.ParseTyped(line)
	if err != nil {
		return fmt.Errorf("failed to parse the line: %s", err)
	}

	// Fill the current row
	t.currentRow = map[int]interface{}{}
	for name, val := range elems {
		if pos, ok := t.colPosition[name]; ok {
			t.currentRow[pos] = val
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
