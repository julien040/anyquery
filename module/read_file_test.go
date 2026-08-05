package module

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
	"github.com/stretchr/testify/require"
)

// TestFileModuleFormatDerivation checks which reader FileModule.Connect would
// pick, without opening anything: no file has to exist, so a compressed name
// (data.csv.gz) is covered here for the format it derives, independently of
// whether the bytes behind it can be decompressed.
func TestFileModuleFormatDerivation(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		format   string
		want     string
		wantErr  []string // substrings the error must contain
	}{
		{name: "csv extension", fileName: "/tmp/data.csv", want: "csv"},
		{name: "tsv extension", fileName: "/tmp/data.tsv", want: "tsv"},
		{name: "jsonl extension", fileName: "/tmp/data.jsonl", want: "jsonl"},
		{name: "ndjson extension", fileName: "/tmp/data.ndjson", want: "ndjson"},
		{name: "toml extension", fileName: "/tmp/config.toml", want: "toml"},
		{name: "yml alias", fileName: "/tmp/config.yml", want: "yml"},
		{name: "htm alias", fileName: "/tmp/page.htm", want: "htm"},
		{name: "pq alias", fileName: "/tmp/data.pq", want: "pq"},
		{name: "uppercase extension", fileName: "/tmp/DATA.JSON", want: "json"},
		{name: "gzip compressed csv", fileName: "/tmp/data.csv.gz", want: "csv"},
		{name: "zstd compressed json", fileName: "/tmp/data.json.zst", want: "json"},
		{name: "zstd long extension", fileName: "/tmp/data.jsonl.zstd", want: "jsonl"},
		{name: "http url", fileName: "https://example.com/dir/data.parquet", want: "parquet"},
		{
			name:     "http url with query",
			fileName: "https://example.com/dir/data.csv?signature=abc",
			want:     "csv",
		},
		{
			name:     "explicit format wins over the extension",
			fileName: "/tmp/data.txt",
			format:   "csv",
			want:     "csv",
		},
		{
			name:     "explicit format is case insensitive",
			fileName: "/tmp/data",
			format:   "CSV",
			want:     "csv",
		},
		{
			// An explicit format must not require the source to be classifiable,
			// otherwise a source ParseSource rejects can never be read.
			name:     "explicit format skips the source classification",
			fileName: "s3://bucket/data",
			format:   "csv",
			want:     "csv",
		},
		{
			name:     "stdin",
			fileName: "stdin",
			wantErr:  []string{"stdin", "format="},
		},
		{
			name:     "dash stdin",
			fileName: "-",
			wantErr:  []string{"stdin", "format="},
		},
		{
			name:     "no extension",
			fileName: "/tmp/data",
			wantErr:  []string{"no file extension", "format=", "csv", "parquet"},
		},
		{
			name:     "unknown extension",
			fileName: "/tmp/data.xlsx",
			wantErr:  []string{"xlsx", "format=", "csv"},
		},
		{
			name:     "unsupported explicit format",
			fileName: "/tmp/data.csv",
			format:   "xlsx",
			wantErr:  []string{"unsupported format", "xlsx", "csv"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := formatForSource(test.fileName, test.format)
			if len(test.wantErr) > 0 {
				require.Error(t, err, "resolving the format must fail")
				for _, substring := range test.wantErr {
					require.Contains(t, err.Error(), substring, "error message must mention it")
				}
				return
			}
			require.NoError(t, err, "resolving the format must not fail")
			require.Equal(t, test.want, got, "resolved format")

			entry, ok := fileFormats[got]
			require.True(t, ok, "resolved format must be a known reader")
			require.NotNil(t, entry.newModule(nil), "reader must be instantiable")
		})
	}
}

// TestFileModule dispatches through SQL: every case creates a virtual table on
// a file of the temp dir and asserts the reader that got the arguments by
// querying the shape it produces (key/value for YAML, the CSV columns, ...).
func TestFileModule(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, content string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600), "writing the fixture must not fail")
		return path
	}

	csvContent := "name,age\nalice,30\nbob,25\n"
	csvFile := write("data.csv", csvContent)
	tsvFile := write("data.tsv", "name\tage\nalice\t30\nbob\t25\n")
	jsonFile := write("data.json", `[{"name":"alice","age":30},{"name":"bob","age":25}]`)
	yamlFile := write("data.yaml", "name: alice\nage: 30\n")
	// No extension: the format can only come from format=.
	plainFile := write("plain", csvContent)

	sql.Register("sqlite3-read-file", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			// A nil policy means unrestricted, which the temp dir reads need.
			return conn.CreateModule("read_file", &FileModule{Restrictions: nil})
		},
	})

	db, err := sql.Open("sqlite3-read-file", ":memory:")
	require.NoError(t, err, "opening connection must not fail")
	defer db.Close()
	dbx := sqlx.NewDb(db, "sqlite3-read-file")

	tests := []struct {
		name string
		// args is the argument list passed to read_file(...)
		args string
		// wantCreateErr are substrings the CREATE error must contain. When set,
		// assert is not run.
		wantCreateErr []string
		assert        func(t *testing.T, table string)
	}{
		{
			name: "csv extension",
			args: fmt.Sprintf("'%s', header=true", csvFile),
			assert: func(t *testing.T, table string) {
				var rowCount int
				require.NoError(t, dbx.Get(&rowCount, "SELECT count(*) FROM "+table))
				require.Equal(t, 2, rowCount, "row count")

				var age string
				require.NoError(t, dbx.Get(&age, "SELECT age FROM "+table+" WHERE name = 'alice'"))
				require.Equal(t, "30", age, "value of the age column")
			},
		},
		{
			name: "tsv extension splits on tabs",
			args: fmt.Sprintf("'%s', header=true", tsvFile),
			assert: func(t *testing.T, table string) {
				rows, err := dbx.Query("SELECT * FROM " + table + " LIMIT 1")
				require.NoError(t, err, "querying must not fail")
				columns, err := rows.Columns()
				require.NoError(t, err, "getting columns must not fail")
				require.NoError(t, rows.Close())
				// A comma separator would leave a single column holding the whole
				// header line instead of two.
				require.Equal(t, []string{"name", "age"}, columns, "columns of the TSV file")

				var age string
				require.NoError(t, dbx.Get(&age, "SELECT age FROM "+table+" WHERE name = 'bob'"))
				require.Equal(t, "25", age, "value of the age column")
			},
		},
		{
			name: "tsv extension keeps an explicit separator",
			// The file is comma separated despite its name: the separator the
			// caller passed must survive the dispatch.
			args: fmt.Sprintf("'%s', header=true, separator=','", write("comma.tsv", csvContent)),
			assert: func(t *testing.T, table string) {
				var age string
				require.NoError(t, dbx.Get(&age, "SELECT age FROM "+table+" WHERE name = 'alice'"))
				require.Equal(t, "30", age, "value of the age column")
			},
		},
		{
			name: "json extension",
			args: fmt.Sprintf("'%s'", jsonFile),
			assert: func(t *testing.T, table string) {
				var rowCount int
				require.NoError(t, dbx.Get(&rowCount, "SELECT count(*) FROM "+table))
				require.Equal(t, 2, rowCount, "row count")

				var name string
				require.NoError(t, dbx.Get(&name, "SELECT name FROM "+table+" LIMIT 1"))
				require.Equal(t, "alice", name, "value of the name column")
			},
		},
		{
			name: "yaml extension",
			args: fmt.Sprintf("'%s'", yamlFile),
			assert: func(t *testing.T, table string) {
				// The YAML reader flattens the document into key/value rows, so
				// this shape is only reachable through YamlModule.
				var value string
				require.NoError(t, dbx.Get(&value, "SELECT value FROM "+table+" WHERE key = 'name'"))
				require.Equal(t, "alice", value, "flattened value of the name key")
			},
		},
		{
			name: "no extension with an explicit format",
			args: fmt.Sprintf("'%s', format='csv', header=true", plainFile),
			assert: func(t *testing.T, table string) {
				var rowCount int
				require.NoError(t, dbx.Get(&rowCount, "SELECT count(*) FROM "+table))
				require.Equal(t, 2, rowCount, "row count")
			},
		},
		{
			name:          "no extension without a format",
			args:          fmt.Sprintf("'%s'", plainFile),
			wantCreateErr: []string{"no file extension", "format=", supportedFormats()},
		},
		{
			name:          "unknown extension",
			args:          fmt.Sprintf("'%s'", write("data.xlsx", csvContent)),
			wantCreateErr: []string{"xlsx", "format=", supportedFormats()},
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := fmt.Sprintf("t%d", i)
			_, err := db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE %s USING read_file(%s)", table, test.args))
			if len(test.wantCreateErr) > 0 {
				require.Error(t, err, "creating the virtual table must fail")
				for _, substring := range test.wantCreateErr {
					require.Contains(t, err.Error(), substring, "error message must mention it")
				}
				return
			}
			require.NoError(t, err, "creating the virtual table must not fail")
			defer db.Exec("DROP TABLE " + table)
			test.assert(t, table)
		})
	}
}
