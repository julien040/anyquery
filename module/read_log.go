package module

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/edsrzf/mmap-go"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
	"github.com/trivago/grok"
)

//go:embed template.grok
var grokTemplate string

type LogModule struct {
	Restrictions *Restrictions
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

// createGrokParser builds a parser from the built-in patterns, overridden by
// the user's custom patterns. It takes the already-read content of the custom
// pattern file (empty = none) rather than its path: the caller must read that
// file through the sandbox policy, and handing this function a path would make
// it open the path a second time, after the policy already resolved it.
func createGrokParser(customPatternFile []byte) (*grok.Grok, error) {
	// Read the template file
	patterns := extractPatternsFromStr(grokTemplate)

	// Overwrite the default patterns with the custom ones
	if len(customPatternFile) > 0 {
		for key, value := range extractPatternsFromStr(string(customPatternFile)) {
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
	// Freshness of the remote download cache, in seconds. Ignored for a local
	// file, which is never cached.
	cacheTTL := "86400"
	cacheTTLParsed := int64(86400)

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
		{"cache_ttl", &cacheTTL},
		{"cacheTTL", &cacheTTL},
		{"ttl", &cacheTTL},
		{"cache", &cacheTTL},
	}

	parseArgs(params, args)

	if fileName == "" {
		return nil, fmt.Errorf("missing file argument. Pass the file path as the first argument")
	}

	if cacheTTL != "" {
		var err error
		cacheTTLParsed, err = strconv.ParseInt(cacheTTL, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the cache TTL: %s", err)
		}
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

	// The custom grok pattern file is read directly (it does not go through
	// ParseSource/Fetcher), so it is read through the sandbox policy, which
	// enforces the allowed directories on the same open it reads from.
	// ReadLocalFile also covers the unrestricted (nil policy) case.
	customPatterns := []byte{}
	if patternFile != "" {
		customPatterns, err = m.Restrictions.ReadLocalFile(patternFile)
		if err != nil {
			// A refusal by the policy is returned as-is: wrapping it as a
			// failure to read the file would misattribute a configuration
			// refusal to the filesystem. Anything else really is a read
			// failure, and reads better with that context.
			if strings.HasPrefix(err.Error(), "sandbox: ") {
				return nil, err
			}
			return nil, fmt.Errorf("failed to read the custom grok patterns file: %s", err)
		}
	}

	parser, err := createGrokParser(customPatterns)
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

func (t *LogTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
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
