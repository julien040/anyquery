package module

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSniffCSV covers the inference itself: what a file looks like goes in, the
// delimiter, the header flag and the columns come out. Every case states what
// the caller passed, because an explicit value must turn the matching part of
// the detection off.
func TestSniffCSV(t *testing.T) {
	tests := []struct {
		name    string
		content string
		// separator and separatorSet, headerVal and headerSet are the arguments
		// as CsvModule.Connect resolves them. The zero values mean the caller
		// passed neither.
		separator    rune
		separatorSet bool
		headerVal    bool
		headerSet    bool

		wantSeparator rune
		wantHeader    bool
		wantNames     []string
		wantTypes     []string
	}{
		{
			name:          "semicolon delimited",
			content:       "name;age\nalice;30\nbob;25\n",
			wantSeparator: ';',
			wantHeader:    true,
			wantNames:     []string{"name", "age"},
			wantTypes:     []string{"string", "int"},
		},
		{
			name:          "tab delimited",
			content:       "name\tage\nalice\t30\nbob\t25\n",
			wantSeparator: '\t',
			wantHeader:    true,
			wantNames:     []string{"name", "age"},
			wantTypes:     []string{"string", "int"},
		},
		{
			name:          "pipe delimited",
			content:       "name|age\nalice|30\nbob|25\n",
			wantSeparator: '|',
			wantHeader:    true,
			wantNames:     []string{"name", "age"},
			wantTypes:     []string{"string", "int"},
		},
		{
			// The other candidates only appear inside quoted cells, where they
			// are not delimiters, so they must not win.
			name:          "comma delimited with quoted cells holding other candidates",
			content:       "name,age,note\nalice,30,\"a;b|c\"\nbob,25,\"x,y\"\n",
			wantSeparator: ',',
			wantHeader:    true,
			wantNames:     []string{"name", "age", "note"},
			wantTypes:     []string{"string", "int", "string"},
		},
		{
			// Splitting on commas gives one cell for the first record, so the
			// semicolon is the only delimiter that splits the file at all.
			name:          "uneven comma counts do not beat a consistent semicolon",
			content:       "a;b;c\n1,2;3;4\n5;6;7\n",
			wantSeparator: ';',
			wantHeader:    true,
			wantNames:     []string{"a", "b", "c"},
			wantTypes:     []string{"string", "int", "int"},
		},
		{
			// No candidate splits anything, so the comma is kept as the default.
			name:          "single column file falls back to a comma",
			content:       "name\nalice\nbob\n",
			wantSeparator: ',',
			wantHeader:    false,
			wantNames:     []string{"col0"},
			wantTypes:     []string{"string"},
		},
		{
			name:          "header detected above numeric columns",
			content:       "id,score\n1,2.5\n2,3.5\n",
			wantSeparator: ',',
			wantHeader:    true,
			wantNames:     []string{"id", "score"},
			wantTypes:     []string{"int", "float"},
		},
		{
			name:          "header detected above boolean columns",
			content:       "flag,name\ntrue,alice\nfalse,bob\n",
			wantSeparator: ',',
			wantHeader:    true,
			wantNames:     []string{"flag", "name"},
			wantTypes:     []string{"bool", "string"},
		},
		{
			// The first record holds numbers, so it is data like the ones below.
			name:          "no header when the first row is numeric",
			content:       "1,2\n3,4\n5,6\n",
			wantSeparator: ',',
			wantHeader:    false,
			wantNames:     []string{"col0", "col1"},
			wantTypes:     []string{"int", "int"},
		},
		{
			// Nothing tells the first record apart from the others, and reading
			// it as a header would drop a row of data.
			name:          "no header when every row is text",
			content:       "alpha,beta\ngamma,delta\n",
			wantSeparator: ',',
			wantHeader:    false,
			wantNames:     []string{"col0", "col1"},
			wantTypes:     []string{"string", "string"},
		},
		{
			name:          "single row is data",
			content:       "name,age\n",
			wantSeparator: ',',
			wantHeader:    false,
			wantNames:     []string{"col0", "col1"},
			wantTypes:     []string{"string", "string"},
		},
		{
			// The last record has no line break to end it, and must still be
			// part of the sample.
			name:          "last row without a trailing newline",
			content:       "name,age\nalice,30",
			wantSeparator: ',',
			wantHeader:    true,
			wantNames:     []string{"name", "age"},
			wantTypes:     []string{"string", "int"},
		},
		{
			name:          "integers and decimals in one column give a float",
			content:       "measure\n1\n2.5\n3\n",
			wantSeparator: ',',
			wantHeader:    true,
			wantNames:     []string{"measure"},
			wantTypes:     []string{"float"},
		},
		{
			name:          "empty cells do not widen an integer column",
			content:       "id,name\n1,alice\n,bob\n3,carol\n",
			wantSeparator: ',',
			wantHeader:    true,
			wantNames:     []string{"id", "name"},
			wantTypes:     []string{"int", "string"},
		},
		{
			name:          "a column of empty cells is text",
			content:       "id,note\n1,\n2,\n",
			wantSeparator: ',',
			wantHeader:    true,
			wantNames:     []string{"id", "note"},
			wantTypes:     []string{"int", "string"},
		},
		{
			// The caller says there is no header, so the row that looks like one
			// is read as data and widens the column to text.
			name:          "explicit header=false keeps the first row as data",
			content:       "name,age\nalice,30\nbob,25\n",
			headerSet:     true,
			headerVal:     false,
			wantSeparator: ',',
			wantHeader:    false,
			wantNames:     []string{"col0", "col1"},
			wantTypes:     []string{"string", "string"},
		},
		{
			// A header the detection would have missed, since both rows are text.
			name:          "explicit header=true names the columns",
			content:       "city,country\nparis,france\n",
			headerSet:     true,
			headerVal:     true,
			wantSeparator: ',',
			wantHeader:    true,
			wantNames:     []string{"city", "country"},
			wantTypes:     []string{"string", "string"},
		},
		{
			// The file is semicolon separated, but the caller asked for commas:
			// the whole line is then a single cell and no detection corrects it.
			name:          "explicit separator turns delimiter detection off",
			content:       "name;age\nalice;30\nbob;25\n",
			separator:     ',',
			separatorSet:  true,
			wantSeparator: ',',
			wantHeader:    false,
			wantNames:     []string{"col0"},
			wantTypes:     []string{"string"},
		},
		{
			name:          "explicit tab separator",
			content:       "name\tage\nalice\t30\n",
			separator:     '\t',
			separatorSet:  true,
			wantSeparator: '\t',
			wantHeader:    true,
			wantNames:     []string{"name", "age"},
			wantTypes:     []string{"string", "int"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			separator, hasHeader, columns := sniffCSV([]byte(test.content),
				test.separator, test.separatorSet, test.headerSet, test.headerVal)

			require.Equal(t, string(test.wantSeparator), string(separator), "detected separator")
			require.Equal(t, test.wantHeader, hasHeader, "header row")

			names := make([]string, 0, len(columns))
			types := make([]string, 0, len(columns))
			for _, col := range columns {
				names = append(names, col.name)
				types = append(types, col.colType)
			}
			require.Equal(t, test.wantNames, names, "column names")
			require.Equal(t, test.wantTypes, types, "column types")
		})
	}
}

// TestSniffCSVEmptyFile: nothing can be inferred from a file without a record,
// which is how Connect knows it has to reject it.
func TestSniffCSVEmptyFile(t *testing.T) {
	for _, content := range []string{"", "\n", "\n\n"} {
		_, _, columns := sniffCSV([]byte(content), ',', false, false, false)
		require.Empty(t, columns, "no column can be inferred from %q", content)
	}
}

// TestSniffCSVSampleBounds: the sniffer stops after a bounded head of the file,
// so what comes after it cannot influence the result. It is also what makes
// opening a large file cheap.
func TestSniffCSVSampleBounds(t *testing.T) {
	t.Run("records past the sample are ignored", func(t *testing.T) {
		file := strings.Builder{}
		file.WriteString("id\n")
		for i := 0; i < csvSniffMaxRecords+50; i++ {
			// The values beyond the sample are text, and would widen the column
			// if they were read.
			if i < csvSniffMaxRecords {
				fmt.Fprintf(&file, "%d\n", i)
			} else {
				file.WriteString("not a number\n")
			}
		}

		_, hasHeader, columns := sniffCSV([]byte(file.String()), ',', false, false, false)
		require.True(t, hasHeader, "the header must be detected")
		require.Len(t, columns, 1)
		require.Equal(t, "int", columns[0].colType, "the column type must come from the sample only")
	})

	t.Run("bytes past the sample are ignored", func(t *testing.T) {
		file := strings.Builder{}
		file.WriteString("id;note\n")
		// One long record per line, enough of them to overflow the byte limit
		// before the comma separated tail below is reached.
		filler := strings.Repeat("x", 1024)
		for file.Len() < csvSniffMaxBytes*2 {
			fmt.Fprintf(&file, "1;%s\n", filler)
		}
		file.WriteString("a,b,c,d\n")

		separator, _, columns := sniffCSV([]byte(file.String()), ',', false, false, false)
		require.Equal(t, ";", string(separator), "the separator must come from the sampled head")
		require.Len(t, columns, 2)
	})

	t.Run("the sample never ends in the middle of a record", func(t *testing.T) {
		// A record cut in half would look like a record with fewer fields than
		// the others, which is what the field count is compared on.
		file := strings.Builder{}
		file.WriteString("a;b;c\n")
		for file.Len() < csvSniffMaxBytes+1024 {
			file.WriteString("1;2;3\n")
		}
		sample := csvSniffSample([]byte(file.String()))
		require.Less(t, len(sample), csvSniffMaxBytes+1, "the sample must respect the byte limit")
		require.Equal(t, byte('\n'), sample[len(sample)-1], "the sample must end on a record boundary")

		records := readCsvSample(sample, ';')
		require.NotEmpty(t, records)
		for i, record := range records {
			require.Len(t, record, 3, "record %d of the sample", i)
		}
	})

	t.Run("a file below the limit is sampled whole", func(t *testing.T) {
		// Including its last record, which no line break ends.
		content := []byte("a,b\n1,2")
		require.Equal(t, content, csvSniffSample(content))
	})
}
