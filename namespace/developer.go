// Define SQL functions that helps to develop plugins
package namespace

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	pathlib "path"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hjson/hjson-go/v4"
	"github.com/julien040/anyquery/module"
	"github.com/julien040/anyquery/rpc"
	"github.com/mattn/go-sqlite3"
	"github.com/santhosh-tekuri/jsonschema/v5"

	_ "embed"
)

//go:embed dev_manifest.json
var devManifestSchema string

type manifest struct {
	// Fields that are filled by the function and are therefore ignored by the unmarshal
	Name         string   `json:"-"` // Name of the plugin
	FdLog        *os.File `json:"-"` // File descriptor for the log file
	ListOfTables []string `json:"-"` // List of modules that are created by the plugin
	ManifestPath string   `json:"-"` // Path to the manifest file

	// Fields that are filled by the user
	Executable        string                            `json:"executable"`
	UserConfig        map[string]map[string]interface{} `json:"user_config"`
	TableNames        []string                          `json:"tables"`
	IsSharedExtension bool                              `json:"is_shared_extension"`
	LogFile           string                            `json:"log_file"`
	LogLevel          string                            `json:"log_level"`
	BuildCommand      string                            `json:"build_command"`
}

type devFunction struct {
	conn           *sqlite3.SQLiteConn
	manifests      map[string]manifest
	connectionPool *rpc.ConnectionPool
}

func (f *devFunction) LoadDevPlugin(args ...string) string {
	schema, err := jsonschema.CompileString("dev_manifest.json", devManifestSchema)
	if err != nil {
		return fmt.Sprintf("error compiling schema\n%v", err)
	}

	if len(args) < 2 {
		return "error: not enough arguments"
	}

	pluginName := args[0]
	pluginManifestPath := args[1]

	// Validate the manifest and read it
	rawManifest, err := os.ReadFile(pluginManifestPath)
	if err != nil {
		return fmt.Sprintf("error reading manifest\n%v", err)
	}

	var unmarshaled map[string]interface{}
	err = hjson.Unmarshal(rawManifest, &unmarshaled)
	if err != nil {
		return fmt.Sprintf("error reading json manifest\n%v", err)
	}
	err = schema.Validate(unmarshaled)
	if err != nil {
		return fmt.Sprintf("error validating manifest\n%v", err)
	}

	// Load the manifest properly
	var m manifest
	hjson.Unmarshal(rawManifest, &m)
	m.Name = pluginName
	m.ManifestPath = pluginManifestPath

	parsedArgs := parseCommands(m.Executable)
	path := parsedArgs[0]
	args = parsedArgs[1:]

	// Run the build command if it's present
	if m.BuildCommand != "" {
		buildArgs := parseCommands(m.BuildCommand)
		buildCommand := buildArgs[0]
		buildArgs = buildArgs[1:]

		content, err := exec.Command(buildCommand, buildArgs...).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("error running build command\n%v\n%s", err, content)
		}
	}

	outputLog := io.Discard

	if m.LogFile != "" {
		logFile, err := os.OpenFile(m.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Sprintf("error opening log file\n%v", err)
		}
		outputLog = logFile
		m.FdLog = logFile
	}

	if m.LogLevel == "" {
		m.LogLevel = "info"
	}

	// Set the log file and log level
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "dev-plugin " + pluginName,
		Level:  hclog.LevelFromString(m.LogLevel),
		Output: outputLog,
	})

	i := 0
	tableNames := []string{}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("error getting working directory\n%v", err)
	}

	pluginPath := ""
	if pathlib.IsAbs(path) {
		pluginPath = path
	} else {
		pluginPath = pathlib.Join(wd, path)
	}

	for profileName, profile := range m.UserConfig {
		// Table name is <profile_name>_<plugin_name>_<table_name>
		// unless the profile name is "default"
		tableNamePrefix := ""
		if profileName != "default" {
			tableNamePrefix = profileName + "_"
		}

		for tableIndex, tableName := range m.TableNames {
			tableName = tableNamePrefix + pluginName + "_" + tableName
			moduleToLoad := &module.SQLiteModule{
				PluginPath: pluginPath,
				PluginArgs: args,
				UserConfig: profile,
				PluginManifest: rpc.PluginManifest{
					Name:        pluginName,
					Tables:      m.TableNames,
					Description: "Dev plugin " + pluginName,
					Author:      "An awesome developer",
					Version:     "0.0.1",
				},
				TableIndex:      tableIndex,
				ConnectionIndex: i,
				ConnectionPool:  f.connectionPool,
				Stderr:          m.FdLog,
				Logger:          logger,
			}

			err = f.conn.CreateModule(tableName, moduleToLoad)
			if err != nil {
				return fmt.Sprintf("error creating module\n%v", err)
			}

			tableNames = append(tableNames, tableName)
		}
		// Increment the connection index
		i++
	}

	m.ListOfTables = tableNames
	f.manifests[pluginName] = m

	returnMessage := strings.Builder{}
	returnMessage.WriteString("Successfully loaded plugin " + pluginName + "\n")
	returnMessage.WriteString("Tables:\n")
	for _, tableName := range tableNames {
		returnMessage.WriteString(tableName + "\n")
	}
	return returnMessage.String()
}

func (f *devFunction) UnloadDevPlugin(name string) string {
	// Get the manifest
	manifest, ok := f.manifests[name]
	if !ok {
		return "Dev plugin " + name + " not found"
	}

	// Unload the plugin
	for _, tableName := range manifest.ListOfTables {
		err := f.conn.DropModule(tableName)
		if err != nil {
			return err.Error()
		}
	}

	// Close the file opened for stderr
	if manifest.FdLog != nil {
		manifest.FdLog.Close()
	}

	// Remove the manifest from the map
	delete(f.manifests, name)

	// Return the success message
	return "Successfully unloaded plugin " + name
}

func (f *devFunction) ReloadDevPlugin(name string) string {
	manifest := f.manifests[name]
	if manifest.Name == "" {
		return "Dev plugin " + name + " not found"
	}

	manifestPath := manifest.ManifestPath

	// Unload the plugin
	unloadMessage := f.UnloadDevPlugin(name)
	if !strings.Contains(unloadMessage, "Successfully unloaded plugin") {
		return unloadMessage
	}

	// Load the plugin
	loadMessage := f.LoadDevPlugin(name, manifestPath)
	if !strings.Contains(loadMessage, "Successfully loaded plugin") {
		return loadMessage
	}

	return "Successfully reloaded plugin " + name
}

func (f *devFunction) ListDevPlugins() string {
	message := strings.Builder{}
	message.WriteString("List of loaded dev plugins:\n")
	for name := range f.manifests {
		message.WriteString(name + " " + f.manifests[name].ManifestPath + "\n")
	}
	return message.String()
}

// Parse the command line arguments so that the executable and its arguments are separated properly
func parseCommands(executableArg string) []string {
	args := []string{}
	tempArg := strings.Builder{}
	doubleQuoteEscape := false
	mustAppend := false
	for _, c := range executableArg {
		switch c {
		case ' ':
			if doubleQuoteEscape {
				tempArg.WriteRune(c)
			} else {
				mustAppend = true
			}

		case '"':
			doubleQuoteEscape = !doubleQuoteEscape

		default:
			tempArg.WriteRune(c)
		}

		if mustAppend {
			args = append(args, tempArg.String())
			tempArg.Reset()
			mustAppend = false
		}

	}
	if tempArg.Len() > 0 {
		args = append(args, tempArg.String())
	}
	return args
}
