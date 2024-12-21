package module

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/edsrzf/mmap-go"
	"github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
)

type interfaceRow struct {
	key   string
	value interface{}
}

type YamlModule struct {
}

type YamlTable struct {
	rows []interfaceRow
	mmap mmap.MMap
}

type YamlCursor struct {
	rows  []interfaceRow
	rowID int64
}

func exploreInterface(val interface{}, prefix string, rows *[]interfaceRow) {
	switch v := val.(type) {
	case map[string]interface{}:
		for key, value := range v {
			switch value.(type) {
			case map[string]interface{}:
				exploreInterface(value, prefix+key+".", rows)
			default:
				exploreInterface(value, prefix+key, rows)
			}

		}
	case []interface{}:
		for i, value := range v {
			switch value.(type) {
			case map[string]interface{}:
				exploreInterface(value, prefix+fmt.Sprintf("[%d].", i), rows)
			default:
				exploreInterface(value, prefix+fmt.Sprintf("[%d]", i), rows)
			}
		}
	case string:
		*rows = append(*rows, interfaceRow{prefix, v})
	case int:
		*rows = append(*rows, interfaceRow{prefix, v})
	case float64:
		*rows = append(*rows, interfaceRow{prefix, v})
	case bool:
		*rows = append(*rows, interfaceRow{prefix, v})
	case nil:
		*rows = append(*rows, interfaceRow{prefix, nil})
	default:
		// Other types are ignored
	}
}

func (m *YamlModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (v *YamlModule) DestroyModule() {}

func (m *YamlModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	// Parse the file content
	fileName := ""

	if len(args) > 3 {
		fileName = strings.Trim(args[3], "'\" ")
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
		return nil, fmt.Errorf("missing file argument. Specify it as: SELECT * FROM read_yaml('file=https://example.com');")
	}

	content := []byte{}
	mmap := mmap.MMap{}
	var err error

	if fileName == "stdin" || fileName == "/dev/stdin" || fileName == "-" {
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %s", err)
		}
	} else {
		content, err = openMmapedFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %s", err)
		}
		mmap = content
	}

	// Parse the YAML content
	var val interface{}
	err = yaml.Unmarshal(content, &val)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML content: %s", err)
	}

	rows := []interfaceRow{}
	exploreInterface(val, "", &rows)

	c.DeclareVTab("CREATE TABLE x(key TEXT, value TEXT)")

	return &YamlTable{
		rows: rows,
		mmap: mmap,
	}, nil
}

func (t *YamlTable) Open() (sqlite3.VTabCursor, error) {
	return &YamlCursor{
		rows: t.rows,
	}, nil
}

func (t *YamlTable) Disconnect() error {
	if t.mmap != nil {
		t.mmap.Unmap()
	}
	return nil
}

func (t *YamlTable) Destroy() error {
	return nil
}

func (t *YamlTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		Used: make([]bool, len(cst)),
	}, nil
}

func (t *YamlCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	t.rowID = 0
	return nil
}

func (t *YamlCursor) Next() error {
	t.rowID++
	return nil
}

func (t *YamlCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if t.rowID >= int64(len(t.rows)) {
		context.ResultNull()
		return nil
	}

	row := t.rows[t.rowID]
	if col == 0 {
		context.ResultText(row.key)
	} else if col == 1 {
		switch v := row.value.(type) {
		case string:
			context.ResultText(v)
		case int:
			context.ResultInt(v)
		case float64:
			context.ResultDouble(v)
		case bool:
			if v {
				context.ResultInt(1)
			} else {
				context.ResultInt(0)
			}
		case nil:
			context.ResultNull()
		default:
			// We print the JSON representation of the value
			jsonRep, err := json.Marshal(v)
			if err != nil {
				context.ResultNull()
			} else {
				context.ResultText(string(jsonRep))
			}
		}
	}

	return nil
}

func (t *YamlCursor) EOF() bool {
	return t.rowID >= int64(len(t.rows))
}

func (t *YamlCursor) Rowid() (int64, error) {
	return t.rowID, nil
}

func (t *YamlCursor) Close() error {
	return nil
}
