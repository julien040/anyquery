package module

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgumentParsing(t *testing.T) {
	fileName := ""
	useHeaderStr := ""
	args := []argParam{
		{"file", &fileName},
		{"filepath", &fileName},
		{"header", &useHeaderStr},
		{"file_name", &fileName},
		{"src", &fileName},
	}

	t.Run("Argument without quotes", func(t *testing.T) {
		parseArgs(args, []string{"file=example.csv", "header=true"})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

	t.Run("Argument with double quotes", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{`file="example.csv"`, `header="true"`})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

	t.Run("Argument with single quotes", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{`file='example.csv'`, `header='true'`})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

	t.Run("Argument with spaces", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{`file = example.csv`, `header = true`})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

	t.Run("Argument with mixed quotes", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{`file='example.csv"`, `header="true'`})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

	t.Run("Whole argument with quotes", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{`"file=example.csv"`, `"header=true"`})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

	t.Run("Argument is a URL", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{`filepath="https://microsoftedge.github.io/Demos/json-dummy-data/64KB.json"`})
		require.Equal(t, "https://microsoftedge.github.io/Demos/json-dummy-data/64KB.json", fileName)
	})

	t.Run("Argument is in uppercase", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{`FILE="example.csv"`, `HEADER="true"`})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

	t.Run("Argument is escaped with backticks", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{`file=` + "`example.csv`", `header=` + "`true`"})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

	t.Run("Name and arg is escaped with backticks", func(t *testing.T) {
		fileName = ""
		useHeaderStr = ""
		parseArgs(args, []string{"`file`=`example.csv`", "`header`=\"true\""})
		require.Equal(t, "example.csv", fileName)
		require.Equal(t, "true", useHeaderStr)
	})

}

func TestColumnNameRewriter(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"file", "file"},
		{"file_name", "file_name"},
		{"file-name", "file_name"},
		{"file name", "file_name"},
		{"file-name-1", "file_name_1"},
		{"file_name_1", "file_name_1"},
		{"Notes    ", "Notes"},
		{"Notes/Comments", "Notes_Comments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, transformSQLiteValidName(tt.name))
		})
	}

}
