package module

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

type HtmlModule struct {
}

type HtmlTable struct {
	table   *html.Node
	file    *os.File
	rows    *goquery.Selection
	isTable bool // Whether the rows node is a table or not
}

type HtmlCursor struct {
	rows      *goquery.Selection
	rowID     int64
	actualRow []string
	isTable   bool // Whether we should search for th, td or directly the children
}

func (m *HtmlModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (v *HtmlModule) DestroyModule() {}

func (m *HtmlModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	fileName := ""
	cssSelector := ""

	if len(args) > 3 {
		fileName = strings.Trim(args[3], "' \"")
	}

	if len(args) > 4 {
		cssSelector = strings.Trim(args[4], "' \"")
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
		{"selector", &cssSelector},
		{"css_selector", &cssSelector},
		{"css", &cssSelector},
	}

	parseArgs(params, args)

	if fileName == "" {
		return nil, fmt.Errorf("missing file argument. Example: SELECT * FROM read_html('file=https://example.com');")
	}

	var file *os.File

	if fileName == "/dev/stdin" || fileName == "-" || fileName == "stdin" {
		// Read from stdin
		file = os.Stdin
	} else {

		// Get the cached path
		filePath, err := findCachedDestination(fileName)
		if err != nil {
			return nil, err
		}

		// Download the file
		err = downloadFile(fileName, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %s", err)
		}

		// Open the file
		file, err = os.OpenFile(filePath, os.O_RDONLY, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %s", err)
		}
	}

	document, err := html.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %s", err)
	}

	allMatches := []*html.Node{}

	// If a css selector is provided, find the node
	// Otherwise, we consider the table is the root node
	if cssSelector != "" {
		query, err := cascadia.ParseWithPseudoElement(cssSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSS selector: %s", err)
		}

		// Find the node
		allMatches = cascadia.QueryAll(document, query)
		if len(allMatches) == 0 {
			return nil, fmt.Errorf("failed to find the node with the CSS selector: %s", cssSelector)
		}
		document = allMatches[0]
		if document == nil {
			return nil, fmt.Errorf("failed to find the node with the CSS selector: %s", cssSelector)
		}
	}
	if document.Type != html.ElementNode {
		return nil, fmt.Errorf("the node found is not an element node")
	}

	rows := &goquery.Selection{}

	// Ensure the node is a table and there is only one match
	// Otherwise, we will just return a table with all the matches
	if document.Data != "table" && len(allMatches) != 1 {
		// Get all the matches
		for _, match := range allMatches {
			rows = rows.AddNodes(match)
		}

		// Declare the columns
		c.DeclareVTab("CREATE TABLE x(tag_name TEXT, content TEXT, attributes TEXT)")

	} else {
		goqueryDoc := goquery.NewDocumentFromNode(document)

		rows = goqueryDoc.Find("tbody tr")

		// Find the headers
		columns := []string{}
		goqueryDoc.Find("thead th").Each(func(i int, s *goquery.Selection) {
			columns = append(columns, s.Text())
		})

		// If no headers are found, try to find the first row
		if len(columns) == 0 {
			rows.First().Find("td, th").Each(func(i int, s *goquery.Selection) {
				columns = append(columns, fmt.Sprintf("col%d", i))
			})
		}

		// If no columns are found, return an error
		if len(columns) == 0 {
			return nil, fmt.Errorf("no columns found in the table")
		}

		// Create the table
		schema := strings.Builder{}
		schema.WriteString("CREATE TABLE x(")
		for i, col := range columns {
			if i > 0 {
				schema.WriteString(", ")
			}
			schema.WriteString("`" + col + "`")
			schema.WriteString(" TEXT")
		}
		schema.WriteString(")")

		c.DeclareVTab(schema.String())
	}

	return &HtmlTable{
		table:   document,
		rows:    rows,
		isTable: document.Data == "table",
		file:    file,
	}, nil
}

func (t *HtmlTable) Open() (sqlite3.VTabCursor, error) {
	return &HtmlCursor{
		rows:    t.rows,
		isTable: t.isTable,
	}, nil
}

func (t *HtmlTable) Disconnect() error {
	if t.file != nil {
		t.file.Close()
	}
	return nil
}

func (t *HtmlTable) Destroy() error {
	return nil
}

func (t *HtmlTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{
		Used: make([]bool, len(cst)),
	}, nil
}

func (t *HtmlCursor) fillBuffer() {
	if t.rows == nil {
		return
	}
	if t.rowID >= int64(t.rows.Length()) {
		return
	}

	t.actualRow = []string{}
	if t.isTable {
		// For each th in the actual tr, fill the buffer with the text of the th
		tr := t.rows.Eq(int(t.rowID))
		if tr == nil {
			return
		}
		tr.Find("th, td").Each(func(i int, s *goquery.Selection) {
			t.actualRow = append(t.actualRow, s.Text())
		})
	} else {
		// For each child of the actual node, fill the buffer with the tag name, the content and the attributes
		node := t.rows.Get(int(t.rowID))
		if node == nil {
			return
		}
		t.actualRow = append(t.actualRow, node.Data)

		// Get the content
		goqueryNode := goquery.NewDocumentFromNode(node)
		text := goqueryNode.Text()
		text = strings.TrimSpace(text)
		t.actualRow = append(t.actualRow, text)

		// JSON encode the attributes
		attrJSON, err := json.Marshal(node.Attr)
		if err != nil {
			t.actualRow = append(t.actualRow, "")
		} else {
			t.actualRow = append(t.actualRow, string(attrJSON))
		}

	}
}

func (t *HtmlCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	t.rowID = 0
	t.fillBuffer()
	return nil
}

func (t *HtmlCursor) Next() error {
	t.rowID++
	t.fillBuffer()
	return nil
}

func (t *HtmlCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if col >= len(t.actualRow) {
		context.ResultNull()
	} else {
		context.ResultText(t.actualRow[col])
	}
	return nil
}

func (t *HtmlCursor) EOF() bool {
	return t.rowID >= int64(t.rows.Length())
}

func (t *HtmlCursor) Rowid() (int64, error) {
	return t.rowID, nil
}

func (t *HtmlCursor) Close() error {
	return nil
}
