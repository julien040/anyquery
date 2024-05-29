package registry

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/adrg/xdg"
	"github.com/julien040/anyquery/controller/config"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/stretchr/testify/require"
)

// TestValidateSchema tests the validateSchema function.
func TestValidateSchema(t *testing.T) {
	// Test cases
	tests := []struct {
		name     string
		registry []byte
		wantErr  bool
	}{
		{
			name: "Valid registry",
			registry: []byte(`{
				"plugins": [
				  {
					"name": "github",
					"desc": "\u003ch2\u003eI'm a description\u003c/h2\u003e\r\n\u003cp\u003eOf course I am\u003c/p\u003e",
					"author": "",
					"versions": [
					  {
						"version": "0.1.2",
						"files": {
						  "darwin/arm64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/sqlean_darwin64/sqlean_macos_arm64_ET7GxU6Ehd.zip",
							"path": ""
						  },
						  "linux/arm64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/z9g6bbpid9xamna/libspatialite_5_1_49DLzRFvGl.0.zip",
							"path": ""
						  }
						},
						"minimum_required_version": "0.0.1",
						"user_config": [],
						"tables": []
					  },
					  {
						"version": "0.0.2",
						"files": {
						  "darwin/arm64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/sqlean_darwin64/sqlean_macos_arm64_ET7GxU6Ehd.zip",
							"path": ""
						  }
						},
						"minimum_required_version": "0.0.1",
						"user_config": [
						  {
							"Name": "Notion API Key",
							"Required": true,
							"Type": "string"
						  }
						],
						"tables": [
						  "db1",
						  "db2"
						]
					  },
					  {
						"version": "0.0.1",
						"files": {
						  "darwin/arm64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/sqlean_darwin64/sqlean_macos_arm64_ET7GxU6Ehd.zip",
							"path": ""
						  },
						  "linux/amd64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/7pnw3hj474e0lkq/empty_TVTxFD4RQc.zip",
							"path": ""
						  }
						},
						"minimum_required_version": "0.0.1",
						"user_config": [],
						"tables": []
					  }
					],
					"license": "MIT",
					"homepage": "https://google.com",
					"last_version": "0.1.2",
					"type": "anyquery"
				  }
				],
				"title": "Anyquery Official Registry",
				"$schema": "https://registry.anyquery.dev/schema_registry.json"
			  }`),
			wantErr: false,
		},
		{
			name: "Invalid registry",
			registry: []byte(`{
				"plugins": [
				  {
					"name": "github",
					"desc": "\u003ch2\u003eI'm a description\u003c/h2\u003e\r\n\u003cp\u003eOf course I am\u003c/p\u003e",
					"author": "",
					"versions": [
					  {
						"version": "0.1.2",
						"files": {
						  "darwin/arm64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/sqlean_darwin64/sqlean_macos_arm64_ET7GxU6Ehd.zip",
							"path": ""
						  },
						  "linux/arm64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/z9g6bbpid9xamna/libspatialite_5_1_49DLzRFvGl.0.zip",
							"path": ""
						  }
						},
						"minimum_required_version": "0.0.1",
						"user_config": [],
						"tables": []
					  },
					  {
						"version": "0.0.2",
						"files": {
						  "darwin/arm64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/sqlean_darwin64/sqlean_macos_arm64_ET7GxU6Ehd.zip",
							"path": ""
						  }
						},
						"minimum_required_version": "0.0.1",
						"user_config": [
						  {
							"Name": "Notion API Key",
							"Required": true
						  }
						],
						"tables": [
						  "db1",
						  "db2"
						]
					  },
					  {
						"version": "0.0.1",
						"files": {
						  "darwin/arm64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/sqlean_darwin64/sqlean_macos_arm64_ET7GxU6Ehd.zip",
							"path": ""
						  },
						  "linux/amd64": {
							"hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							"url": "http://localhost:8090/api/files/qi4bnlajjtxxt5x/7pnw3hj474e0lkq/empty_TVTxFD4RQc.zip",
							"path": ""
						  }
						},
						"minimum_required_version": "0.0.1",
						"user_config": [],
						"tables": []
					  }
					],
					"license": "MIT",
					"homepage": "https://google.com",
					"last_version": "0.1.2",
					"type": "anyquery"
				  }
				],
				"$schema": "https://registry.anyquery.dev/schema_registry.json"
			  }`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateSchema(tt.registry); (err != nil) != tt.wantErr {
				t.Errorf("validateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// A mock server that returns a registry
const registry = `https://gist.githubusercontent.com/julien040/8a6db37826cbeb6b999463cd49067bcc/raw/5b4110a478106f1a1357ec27c016eae3f1063ad9/registry1.json`

// Test if a registry can be added and updated
func TestRegistry(t *testing.T) {
	db, query, err := config.OpenDatabaseConnection("./test.db", false)
	if err != nil {
		t.Fatal(fmt.Errorf("error opening database: %v", err))
	}
	var updateTime int64
	defer db.Close()
	defer os.Remove("./test.db")
	t.Run("Add registry", func(t *testing.T) {
		err := AddNewRegistry(query, "default", registry)
		if err != nil {
			t.Fatal(fmt.Errorf("error adding registry: %v", err))
		}
		ctx := context.Background()

		testedRegistry, err := query.GetRegistry(ctx, "default")
		require.NoError(t, err)

		require.Equal(t, "default", testedRegistry.Name)
		require.Equal(t, registry, testedRegistry.Url)
		require.Equal(t, "6069c08e689f1a792262d71fac6fcc6d161a7621cd37f1174ef1255bfae7eda8", testedRegistry.Checksumregistry)
		updateTime = testedRegistry.Lastupdated
		require.NotEqual(t, 0, updateTime)

	})

	t.Run("Create an invalid registry", func(t *testing.T) {
		err := AddNewRegistry(query, "invalid", "https://example.com")
		require.Error(t, err)

		// Assert that the registry was not added
		t.Run("Check if an invalid registry was not added", func(t *testing.T) {
			_, err := query.GetRegistry(context.Background(), "invalid")
			require.Error(t, err)
		})

	})

	t.Run("Update registry", func(t *testing.T) {
		err := UpdateRegistry(query, "default")
		require.NoError(t, err)
		_, err = query.GetRegistry(context.Background(), "default")
		require.NoError(t, err)
	})

	t.Run("Load the registry", func(t *testing.T) {
		registry, parsedPlugins, err := LoadRegistry(query, "default")
		require.NoError(t, err)
		require.NotNil(t, registry)
		require.NotNil(t, parsedPlugins)

		// Check if the registry is the same
		require.Equal(t, "Anyquery Official Registry", parsedPlugins.Title)
		require.Equal(t, "sqlean", parsedPlugins.Plugins[0].Name)
		require.Equal(t, "MIT", parsedPlugins.Plugins[0].License)
		require.Equal(t, "https://github.com/nalgeon/sqlean/releases", parsedPlugins.Plugins[0].Homepage)
		require.Equal(t, "sharedObject", parsedPlugins.Plugins[0].Type)
		require.Equal(t, "0.1.2", parsedPlugins.Plugins[0].LastVersion)

		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			file, version, err := FindPluginVersionCandidate(parsedPlugins.Plugins[0])
			require.NoError(t, err)
			require.Equal(t, "3e4065cb3d1cc9cbcdfe9a9cdcfd9d8f60f9f232b6bdbcd3ac913a0dc65351e1", file.Hash)
			require.Equal(t, "https://github.com/nalgeon/sqlean/releases/download/0.22.0/sqlean-macos-arm64.zip", file.URL)
			require.Equal(t, "text.dylib", file.Path)
			require.Equal(t, "0.0.1", version.MinimumRequiredVersion)
			require.Equal(t, 1, len(version.UserConfig))
			require.Equal(t, "Notion API Key", version.UserConfig[0].Name)
			require.Equal(t, true, version.UserConfig[0].Required)
			require.Equal(t, 0, len(version.Tables))

		}

		if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
			file, version, err := FindPluginVersionCandidate(parsedPlugins.Plugins[0])
			require.NoError(t, err)
			require.Equal(t, "62c776a64f442e513a865adcf8b430254cdd80f6eca14d4df7d11c770f94eefd", file.Hash)
			require.Equal(t, "https://github.com/nalgeon/sqlean/releases/download/0.22.0/sqlean-linux-x86.zip", file.URL)
			require.Equal(t, "text.so", file.Path)
			require.Equal(t, "0.0.1", version.MinimumRequiredVersion)
			require.Equal(t, 1, len(version.UserConfig))
			require.Equal(t, "Notion API Key", version.UserConfig[0].Name)
			require.Equal(t, true, version.UserConfig[0].Required)
			require.Equal(t, 0, len(version.Tables))
		}
	})

	t.Run("Install a plugin", func(t *testing.T) {

		err := InstallPlugin(query, "default", "sqlean")
		require.NoError(t, err)

		// Check if the plugin was installed
		pathPlugin := path.Join(xdg.DataHome, "anyquery", "plugins", "default", "sqlean")

		// Check if the folder exists
		_, err = os.Stat(pathPlugin)
		require.NoError(t, err)

		// Check if the file exists
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			_, err = os.Stat(path.Join(pathPlugin, "text.dylib"))
			require.NoError(t, err)
		}

		if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
			_, err = os.Stat(path.Join(pathPlugin, "text.so"))
			require.NoError(t, err)
		}

		// Check if the plugin was added to the database
		pluginData, err := query.GetPlugin(context.Background(), model.GetPluginParams{
			Name:     "sqlean",
			Registry: "default",
		})
		require.NoError(t, err)
		require.Equal(t, "sqlean", pluginData.Name)
		require.Equal(t, "default", pluginData.Registry)
		require.Equal(t, int64(0), pluginData.Dev)
		require.Equal(t, pathPlugin, pluginData.Path)
		require.Equal(t, sql.NullString{
			Valid:  true,
			String: "nalgeon",
		}, pluginData.Author)
		require.Equal(t, "[]", pluginData.Tablename)

		// Delete the plugin
		err = os.RemoveAll(pathPlugin)
		require.NoError(t, err)
	})

}
