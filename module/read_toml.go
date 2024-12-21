package module

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/edsrzf/mmap-go"
	"github.com/mattn/go-sqlite3"
)

type TomlModule struct {
}

type TomlTable struct {
	rows []interfaceRow
	mmap mmap.MMap
}

type TomlCursor struct {
	rows  []interfaceRow
	rowID int64
}

func (m *TomlModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (v *TomlModule) DestroyModule() {}

func (m *TomlModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
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
		return nil, fmt.Errorf("missing file argument. Specify it as: SELECT * FROM read_toml('file=https://example.com');")
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
	err = toml.Unmarshal(content, &val)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML content: %s", err)
	}

	rows := []interfaceRow{}
	exploreInterface(val, "", &rows)

	c.DeclareVTab("CREATE TABLE x(key TEXT, value TEXT)")

	return &TomlTable{
		rows: rows,
		mmap: mmap,
	}, nil
}

func (t *TomlTable) Open() (sqlite3.VTabCursor, error) {
	return &TomlCursor{
		rows: t.rows,
	}, nil
}

func (t *TomlTable) Disconnect() error {
	if t.mmap != nil {
		t.mmap.Unmap()
	}
	return nil
}

func (t *TomlTable) Destroy() error {
	return nil
}

func (t *TomlTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		Used: make([]bool, len(cst)),
	}, nil
}

func (t *TomlCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	t.rowID = 0
	return nil
}

func (t *TomlCursor) Next() error {
	t.rowID++
	return nil
}

func (t *TomlCursor) Column(context *sqlite3.SQLiteContext, col int) error {
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

func (t *TomlCursor) EOF() bool {
	return t.rowID >= int64(len(t.rows))
}

func (t *TomlCursor) Rowid() (int64, error) {
	return t.rowID, nil
}

func (t *TomlCursor) Close() error {
	return nil
}
