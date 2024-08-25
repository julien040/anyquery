package controller

import (
	"bytes"
	"testing"

	"github.com/julien040/anyquery/namespace"
	"github.com/stretchr/testify/require"
)

func TestOutputJSON(t *testing.T) {

	// Create a variable buffer to store the output
	buf := bytes.Buffer{}

	output := outputTable{
		Type: outputTableTypeJson,
		Columns: []string{
			"id",
			"name",
			"age",
			"weight",
			"alive",
		},

		Writer: &buf,
	}

	err := output.WriteRows([][]interface{}{
		{1, "John", 25, 80.5, true},
		{2, "Jane", 23, 60.5, false},
	})
	require.NoError(t, err, "The output should be written without errors")

	err = output.Close()
	require.NoError(t, err, "The output should be closed without errors")

	expected := `[{"age":25,"alive":true,"id":1,"name":"John","weight":80.5}
 ,{"age":23,"alive":false,"id":2,"name":"Jane","weight":60.5}
]
`

	require.Equal(t, expected, buf.String(), "The output should be correct for ugly JSON")

	// Test the pretty print
	buf.Reset()

	output = outputTable{
		Type:    outputTableTypeJsonPretty,
		Columns: output.Columns,
		Writer:  &buf,
	}

	err = output.WriteRows([][]interface{}{
		{1, "John", 25, 80.5, true},
		{2, "Jane", 23, 60.5, false},
	})

	require.NoError(t, err, "The output should be written without errors")

	err = output.Close()
	require.NoError(t, err, "The output should be closed without errors")

	expected = `[
  {
    "age": 25,
    "alive": true,
    "id": 1,
    "name": "John",
    "weight": 80.5
  }
 ,{
    "age": 23,
    "alive": false,
    "id": 2,
    "name": "Jane",
    "weight": 60.5
  }
]
`

	require.Equal(t, expected, buf.String(), "The output should be correct for pretty JSON")

}

func TestOutputJSONLines(t *testing.T) {
	buf := bytes.Buffer{}
	output := newOutputTable(
		[]string{"id", "name", "age", "weight", "alive"},
		outputTableTypeJsonLines,
		&buf,
	)

	err := output.WriteRows([][]interface{}{
		{1, "John", 25, 80.5, true},
		{2, "Jane", 23, 60.5, false},
	})

	require.NoError(t, err, "The output should be written without errors")

	err = output.Close()

	require.NoError(t, err, "The output should be closed without errors")

	expected := `{"age":25,"alive":true,"id":1,"name":"John","weight":80.5}
{"age":23,"alive":false,"id":2,"name":"Jane","weight":60.5}` + "\n"

	require.Equal(t, expected, buf.String(), "The output should be correct")

}

func TestOutputCSV(t *testing.T) {
	buf := bytes.Buffer{}
	output := newOutputTable(
		[]string{"id", "name", "age", "weight", "alive"},
		outputTableTypeCsv,
		&buf,
	)

	err := output.WriteRows([][]interface{}{
		{1, "John", 25, 80.5, true},
		{2, "Jane", 23, 60.5, false},
	})

	require.NoError(t, err, "The output should be written without errors")

	err = output.Close()

	require.NoError(t, err, "The output should be closed without errors")

	expected := "id,name,age,weight,alive\n1,John,25,80.500000,true\n2,Jane,23,60.500000,false\n"

	require.Equal(t, expected, buf.String(), "The output should be correct")

	buf.Reset()

	output = newOutputTable(
		[]string{"id", "name", "age", "weight", "alive"},
		outputTableTypePlainWithHeader,
		&buf,
	)

	err = output.WriteRows([][]interface{}{
		{1, "John", 25, 80.5, true},
		{2, "Jane", 23, 60.5, false},
	})

	require.NoError(t, err, "The output should be written without errors")

	err = output.Close()

	require.NoError(t, err, "The output should be closed without errors")

	expected = "id\tname\tage\tweight\talive\n1\tJohn\t25\t80.500000\ttrue\n2\tJane\t23\t60.500000\tfalse\n"

	require.Equal(t, expected, buf.String(), "The output should be correct")

	buf.Reset()

	output = newOutputTable(
		[]string{"id", "name", "age", "weight", "alive"},
		outputTableTypePlain,
		&buf,
	)

	err = output.WriteRows([][]interface{}{
		{1, "John", 25, 80.5, true},
		{2, "Jane", 23, 60.5, false},
	})

	require.NoError(t, err, "The output should be written without errors")

	err = output.Close()

	require.NoError(t, err, "The output should be closed without errors")

	expected = "1\tJohn\t25\t80.500000\ttrue\n2\tJane\t23\t60.500000\tfalse\n"

	require.Equal(t, expected, buf.String(), "The output should be correct")
}

func TestOutputPretty(t *testing.T) {
	rows := [][]interface{}{
		{1, "John\n is the best", 25, 80.5, true},
		{2, "Jane", 23, 60.5, false},
	}

	buf := bytes.Buffer{}
	output := newOutputTable(
		[]string{"id", "name", "age", "weight", "alive"},
		outputTableTypeMarkdown,
		&buf,
	)

	err := output.WriteRows(rows)
	require.NoError(t, err, "The output should be written without errors")

	err = output.Close()
	require.NoError(t, err, "The output should be closed without errors")

}

func TestOutputSQL(t *testing.T) {
	// Create a namespaceSQL
	namespaceSQL := namespace.Namespace{}
	namespaceSQL.Init(namespace.NamespaceConfig{
		InMemory: true,
	})

	db, err := namespaceSQL.Register("")
	require.NoError(t, err, "The namespace should be registered without errors")

	// Create a table
	_, err = db.Exec("CREATE TABLE test (id INT, name VARCHAR(255), age INT, weight FLOAT, alive INT)")
	require.NoError(t, err, "The table should be created without errors")

	// Insert some rows
	_, err = db.Exec("INSERT INTO test VALUES (1, 'John', 25, 80.5, 1)")
	require.NoError(t, err, "The row should be inserted without errors")
	_, err = db.Exec("INSERT INTO test VALUES (2, 'Jane', 23, 60.5, 0)")
	require.NoError(t, err, "The row should be inserted without errors")

	buf := bytes.Buffer{}
	output := outputTable{
		Writer: &buf,
		Type:   outputTableTypePlainWithHeader,
	}

	rows, err := db.Query("SELECT * FROM test")
	require.NoError(t, err, "The query should be executed without errors")
	err = output.WriteSQLRows(rows)
	require.NoError(t, err, "The output should be written without errors")

	output.Close()

}
