package rpc

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRPCPlugin(t *testing.T) {
	// Build the normal plugin
	os.Mkdir("_test", 0755)
	err := exec.Command("go", "build", "-o", "_test/plugin.out", "../test/normalplugin.go").Run()
	require.NoError(t, err, "The plugin should be built without errors")

	var client *InternalClient

	t.Run("Create a connection to the plugin", func(t *testing.T) {
		client, err = NewClient("_test/plugin.out")
		require.NoError(t, err, "The plugin should be created without errors")
	})

	t.Run("Initialize the plugin", func(t *testing.T) {
		schema, err := client.Plugin.Initialize(0, nil)
		require.NoError(t, err, "The plugin should be initialized without errors")
		require.Equal(t, DatabaseSchema{
			Columns: []DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        ColumnTypeInt,
					IsParameter: false,
				},
				{
					Name:        "name",
					Type:        ColumnTypeString,
					IsParameter: false,
				},
			},
			PrimaryKey:   -1,
			HandleOffset: false,
		}, schema, "The schema should be correct")
	})

	t.Run("Query the plugin", func(t *testing.T) {
		rows, noMoreRows, err := client.Plugin.Query(0, 0, QueryConstraint{})
		require.NoError(t, err, "The plugin should be queried without errors")
		require.Equal(t, [][]interface{}{
			{1, "hello"},
			{2, "world"},
		}, rows, "The rows should be correct")
		require.True(t, noMoreRows, "The noMoreRows should be true")
	})

}
