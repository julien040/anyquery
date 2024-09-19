package controller

import (
	"testing"

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
