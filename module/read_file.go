package module

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
)

// FileModule is a dispatcher: it does not read anything itself, it picks the
// reader matching the source's format and hands the arguments over to it. So
// file_reader('data.csv') behaves exactly like csv_reader('data.csv').
//
// The format comes from the format= argument when it is given, otherwise from
// the file extension of the source.
type FileModule struct {
	Restrictions *Restrictions
}

// fileFormat is one entry of the format -> reader table used by FileModule.
type fileFormat struct {
	newModule func(r *Restrictions) sqlite3.Module

	// tabSeparated marks a format read by CsvModule with a tab delimiter. The
	// delimiter is forwarded as separator=tab because a literal tab in an
	// argument value is eaten by argRegExp (see parseArgs in helper.go), while
	// the "tab" name is normalized to "\t" by CsvModule.Connect.
	tabSeparated bool
}

func csvReader(r *Restrictions) sqlite3.Module     { return &CsvModule{Restrictions: r} }
func jsonReader(r *Restrictions) sqlite3.Module    { return &JSONModule{Restrictions: r} }
func jsonlReader(r *Restrictions) sqlite3.Module   { return &JSONlModule{Restrictions: r} }
func parquetReader(r *Restrictions) sqlite3.Module { return &ParquetModule{Restrictions: r} }
func tomlReader(r *Restrictions) sqlite3.Module    { return &TomlModule{Restrictions: r} }
func yamlReader(r *Restrictions) sqlite3.Module    { return &YamlModule{Restrictions: r} }
func htmlReader(r *Restrictions) sqlite3.Module    { return &HtmlModule{Restrictions: r} }

// fileFormats holds every name accepted by format= and every extension
// FileModule can infer a reader from. Aliases (jsonl/ndjson, yaml/yml,
// html/htm, parquet/pq) are separate entries pointing at the same reader.
var fileFormats = map[string]fileFormat{
	"csv":     {newModule: csvReader},
	"tsv":     {newModule: csvReader, tabSeparated: true},
	"json":    {newModule: jsonReader},
	"jsonl":   {newModule: jsonlReader},
	"ndjson":  {newModule: jsonlReader},
	"parquet": {newModule: parquetReader},
	"pq":      {newModule: parquetReader},
	"toml":    {newModule: tomlReader},
	"yaml":    {newModule: yamlReader},
	"yml":     {newModule: yamlReader},
	"html":    {newModule: htmlReader},
	"htm":     {newModule: htmlReader},
}

// compressionExtensions are dropped before the format extension is read, so
// that data.csv.gz is dispatched to the csv reader.
var compressionExtensions = map[string]bool{
	".gz":   true,
	".zst":  true,
	".zstd": true,
}

// supportedFormats lists the accepted format names, sorted so error messages
// are stable.
func supportedFormats() string {
	names := make([]string, 0, len(fileFormats))
	for name := range fileFormats {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// formatExtension returns the lowercase extension of p without its leading
// dot, skipping a trailing compression extension. ext is path.Ext for URLs and
// filepath.Ext for local paths so that a Windows separator is handled.
func formatExtension(p string, ext func(string) string) string {
	raw := ext(p)
	if compressionExtensions[strings.ToLower(raw)] {
		raw = ext(strings.TrimSuffix(p, raw))
	}
	return strings.TrimPrefix(strings.ToLower(raw), ".")
}

// formatForSource resolves the format name to dispatch on. An explicit format
// wins over the source, and is resolved without classifying the source at all,
// so that format= also rescues a source whose extension is absent or
// misleading.
func formatForSource(fileName string, format string) (string, error) {
	if trimmed := strings.ToLower(strings.TrimSpace(format)); trimmed != "" {
		if _, ok := fileFormats[trimmed]; !ok {
			return "", fmt.Errorf("read_file: unsupported format %q; supported formats are %s",
				trimmed, supportedFormats())
		}
		return trimmed, nil
	}

	source, err := ParseSource(fileName)
	if err != nil {
		return "", err
	}

	extension := ""
	switch source.Kind {
	case KindStdin:
		return "", fmt.Errorf("read_file: stdin has no file name to infer the format from; "+
			"pass format= (one of %s)", supportedFormats())
	case KindHTTP:
		extension = formatExtension(source.URL.Path, path.Ext)
	default:
		extension = formatExtension(source.Path, filepath.Ext)
	}

	if extension == "" {
		return "", fmt.Errorf("read_file: %q has no file extension to infer the format from; "+
			"pass format= (one of %s)", fileName, supportedFormats())
	}

	if _, ok := fileFormats[extension]; !ok {
		return "", fmt.Errorf("read_file: unsupported file extension %q for %q; "+
			"pass format= (one of %s)", extension, fileName, supportedFormats())
	}

	return extension, nil
}

func (m *FileModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (v *FileModule) DestroyModule() {}

func (m *FileModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	fileName := ""
	format := ""
	separator := ""

	if len(args) >= 4 {
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
		{"format", &format},
		{"type", &format},
		// The separator is only read, never forwarded as is: a .tsv source must
		// not overwrite a delimiter the caller passed explicitly. parseArgs is
		// last-wins, so appending separator=tab would win over the caller.
		{"separator", &separator},
		{"field_separator", &separator},
		{"fs", &separator},
		{"delimiter", &separator},
	}
	parseArgs(params, args)

	if fileName == "" {
		return nil, fmt.Errorf("missing file argument. Check the validity of the arguments")
	}

	formatName, err := formatForSource(fileName, format)
	if err != nil {
		return nil, err
	}

	target := fileFormats[formatName]

	// The reader is built per Connect rather than shared: JSONModule keeps the
	// file content of the table it created on the module value itself, so a
	// shared instance would leak state between tables.
	forwarded := args
	if target.tabSeparated && separator == "" {
		forwarded = append(append([]string{}, args...), "separator=tab")
	}

	return target.newModule(m.Restrictions).Connect(c, forwarded)
}
