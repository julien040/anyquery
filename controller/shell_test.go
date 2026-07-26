package controller

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/julien040/anyquery/module"
	"github.com/stretchr/testify/require"
)

func TestQuerySplitter(t *testing.T) {
	t.Parallel()

	test := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "simple query",
			query:    "SELECT * FROM table",
			expected: []string{"SELECT * FROM table"},
		},
		{
			name:     "multiple queries",
			query:    "SELECT * FROM table; SELECT * FROM table2",
			expected: []string{"SELECT * FROM table", "SELECT * FROM table2"},
		},
		{
			name:     "multiple queries with comments",
			query:    "SELECT * FROM table; -- This is a comment\nSELECT * FROM table2",
			expected: []string{"SELECT * FROM table", "-- This is a comment\nSELECT * FROM table2"},
		},
		{
			name:     "multiple queries with a dot command",
			query:    "SELECT * FROM table; .mode\n.tables",
			expected: []string{"SELECT * FROM table", ".mode", ".tables"},
		},
		{
			name:     "multiple queries on multiple lines",
			query:    "SELECT * FROM table;\nSELECT * FROM table2",
			expected: []string{"SELECT * FROM table", "SELECT * FROM table2"},
		},
		{
			name:     "a query with a semi-colon in a string",
			query:    "SELECT * FROM table WHERE name = 'SELECT * FROM table;'",
			expected: []string{"SELECT * FROM table WHERE name = 'SELECT * FROM table;'"},
		},
		{
			name:     "a query with a quote and a semi-colon",
			query:    "SELECT * FROM table WHERE name = 'SELECT * FROM \"table\";';",
			expected: []string{"SELECT * FROM table WHERE name = 'SELECT * FROM \"table\";'"},
		},
		{
			name:     "a query with a double quote escaped",
			query:    "SELECT * FROM table WHERE name = 'Mitchell''s table; and his friends'; .exit",
			expected: []string{"SELECT * FROM table WHERE name = 'Mitchell''s table; and his friends'", ".exit"},
		},
		{
			name:     "a query with slash command and a dot command",
			query:    "\\dt;\n.exit",
			expected: []string{"\\dt", ".exit"},
		},
		{
			name:     "a query with a dot command, a normal command and lot of whitespace",
			query:    "    .mode\n\nSELECT * FROM table;    .exit  ",
			expected: []string{".mode", "SELECT * FROM table", ".exit"},
		},
		{
			name: "a query with a comment and a sql command",
			query: `-- This is a; comment
SELECT * FROM table`,
			expected: []string{"-- This is a; comment\nSELECT * FROM table"},
		},
		{
			name: "a query with a multi-line comment and a sql command",
			query: `/*
This is a multi-line comment; with a semi-colon
Hey
*/
SELECT * FROM table; .exit`,
			expected: []string{"/*\nThis is a multi-line comment; with a semi-colon\nHey\n*/\nSELECT * FROM table", ".exit"},
		},
		{
			name:     "a query with a multi-line comment in the middle of a sql command",
			query:    `SELECT *  /* This is a 'multi-line' comment; with "a" semi-colon*/ FROM table; .exit`,
			expected: []string{"SELECT *  /* This is a 'multi-line' comment; with \"a\" semi-colon*/ FROM table", ".exit"},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, splitMultipleQuery(tt.query))
		})
	}

}

// TestReadCommand covers `.read`, which is handled directly in shell.Run
// before the middleware pipeline runs (see handleReadCommand in shell.go).
// It predates the middleware-based dot-command gate and used to be reachable
// regardless of the "dot-command" config flag or the sandbox policy — this
// locks in the fix.
func TestReadCommand(t *testing.T) {
	t.Parallel()

	t.Run("refused when dot-command is disabled", func(t *testing.T) {
		t.Parallel()

		tmp := filepath.Join(t.TempDir(), "should-not-be-read.sql")
		require.NoError(t, os.WriteFile(tmp, []byte("SELECT 1;"), 0644))

		var buf bytes.Buffer
		sh := &shell{
			Config: middlewareConfiguration{
				"dot-command":       false,
				"doNotModifyOutput": true,
			},
			OutputFileDesc: &buf,
		}

		mustStop := sh.Run(".read " + tmp)

		require.False(t, mustStop)
		require.Contains(t, buf.String(), "not available in this context")
	})

	t.Run("still works when dot-command is enabled", func(t *testing.T) {
		t.Parallel()

		tmp := filepath.Join(t.TempDir(), "read-me.sql")
		require.NoError(t, os.WriteFile(tmp, []byte("SELECT 42;"), 0644))

		var seenQueries []string
		recordingMiddleware := func(qd *QueryData) bool {
			seenQueries = append(seenQueries, qd.SQLQuery)
			return true
		}

		var buf bytes.Buffer
		sh := &shell{
			Config: middlewareConfiguration{
				"dot-command":       true,
				"doNotModifyOutput": true,
			},
			Middlewares:    []middleware{recordingMiddleware},
			OutputFileDesc: &buf,
		}

		mustStop := sh.Run(".read " + tmp)

		require.False(t, mustStop)
		require.Contains(t, seenQueries, "SELECT 42",
			"the file content must still be read and run recursively through p.Run")
	})

	t.Run("mustStop from the recursive run is honoured", func(t *testing.T) {
		t.Parallel()

		tmp := filepath.Join(t.TempDir(), "exit.sql")
		require.NoError(t, os.WriteFile(tmp, []byte(".exit"), 0644))

		var buf bytes.Buffer
		sh := &shell{
			Config: middlewareConfiguration{
				"dot-command":       true,
				"doNotModifyOutput": true,
			},
			OutputFileDesc: &buf,
		}

		mustStop := sh.Run(".read " + tmp)

		require.True(t, mustStop, "a .exit inside the read file must propagate mustStop")
	})

	t.Run("error message does not leak the underlying OS error", func(t *testing.T) {
		t.Parallel()

		missing := filepath.Join(t.TempDir(), "does-not-exist.sql")

		var buf bytes.Buffer
		sh := &shell{
			Config: middlewareConfiguration{
				"dot-command":       true,
				"doNotModifyOutput": true,
			},
			OutputFileDesc: &buf,
		}

		sh.Run(".read " + missing)

		out := buf.String()
		require.Contains(t, out, "Error reading file")
		require.Contains(t, out, missing, "the path is fine to keep for interactive usability")
		require.NotContains(t, out, "no such file",
			"the raw OS error must not be echoed: on a network-facing surface it is an ENOENT/EACCES oracle")
		require.NotContains(t, out, "cannot find the file")
	})

	t.Run("sandbox policy (CheckFileRead) is consulted when dot-command is enabled", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		tmp := filepath.Join(dir, "readable.sql")
		require.NoError(t, os.WriteFile(tmp, []byte("SELECT 1;"), 0644))

		otherDir := t.TempDir()
		outsideFile := filepath.Join(otherDir, "outside.sql")
		require.NoError(t, os.WriteFile(outsideFile, []byte("SELECT 1;"), 0644))

		restrictions := &module.Restrictions{AllowedDirs: []string{dir}}

		var buf bytes.Buffer
		sh := &shell{
			Config: middlewareConfiguration{
				"dot-command":       true,
				"doNotModifyOutput": true,
			},
			Restrictions:   restrictions,
			OutputFileDesc: &buf,
		}

		sh.Run(".read " + outsideFile)
		require.Contains(t, buf.String(), "sandbox",
			"a path outside AllowedDirs must be refused by CheckFileRead")
	})

	t.Run("nil Restrictions (the default) is unrestricted", func(t *testing.T) {
		t.Parallel()

		tmp := filepath.Join(t.TempDir(), "unrestricted.sql")
		require.NoError(t, os.WriteFile(tmp, []byte("SELECT 1;"), 0644))

		var seenQueries []string
		recordingMiddleware := func(qd *QueryData) bool {
			seenQueries = append(seenQueries, qd.SQLQuery)
			return true
		}

		var buf bytes.Buffer
		sh := &shell{
			Config: middlewareConfiguration{
				"dot-command":       true,
				"doNotModifyOutput": true,
			},
			Middlewares:    []middleware{recordingMiddleware},
			OutputFileDesc: &buf,
			// Restrictions left nil: matches the interactive REPL, which
			// must keep working exactly as before this change.
		}

		sh.Run(".read " + tmp)

		require.Contains(t, seenQueries, "SELECT 1")
	})

	// This is the highest-severity real entry point: the LLM/GPT tunnel and
	// MCP paths all funnel through executeQueryLLM, whose Config never sets
	// "dot-command" at all (see llm.go). Every other subtest above sets the
	// key to an explicit false, which is not quite the same code path as a
	// missing key defaulting through GetBool — exercise the actual
	// production entry point directly rather than a synthetic shell.
	t.Run("executeQueryLLM (the unauthenticated GPT-tunnel/MCP path) refuses .read", func(t *testing.T) {
		t.Parallel()

		tmp := filepath.Join(t.TempDir(), "should-not-be-read-via-llm.sql")
		require.NoError(t, os.WriteFile(tmp, []byte("SELECT 'leaked';"), 0644))

		var buf bytes.Buffer
		err := executeQueryLLM(nil, ".read "+tmp, &buf, nil)

		require.NoError(t, err)
		require.Contains(t, buf.String(), "not available in this context")
		require.NotContains(t, buf.String(), "leaked",
			"the file content must never be executed/echoed through the LLM path")
	})
}
