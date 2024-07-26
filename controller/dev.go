package controller

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

type templateDevArgs struct {
	ModuleName string
	ModuleURL  string
	TableName  string
	FileName   string
}

var templateMainGo = `package main

import (
	"github.com/julien040/anyquery/rpc"
)

func main() {
	plugin := rpc.NewPlugin({{.TableName}}Creator)
	plugin.Serve()
}
`

var templateTableGo = `package main

import "github.com/julien040/anyquery/rpc"

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func {{.TableName}}Creator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &{{.TableName}}Table{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "id",
				Type: rpc.ColumnTypeString,
			},
			{
				// This column is a parameter
				// Therefore, it'll be hidden in SELECT * but will be used in WHERE clauses
				// and SELECT * FROM table(<name>)
				Name:        "name",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
			},
		},
	}, nil
}

type {{.TableName}}Table struct {
}

type {{.TableName}}Cursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *{{.TableName}}Cursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	return nil, true, nil
}

// Create a new cursor that will be used to read rows
func (t *{{.TableName}}Table) CreateReader() rpc.ReaderInterface {
	return &{{.TableName}}Cursor{}
}

// A slice of rows to insert
func (t *{{.TableName}}Table) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *{{.TableName}}Table) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *{{.TableName}}Table) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *{{.TableName}}Table) Close() error {
	return nil
}
`

var templateMakefile = `
files := $(wildcard *.go)

all: $(files)
	go build -o {{.ModuleName}}.out $(files)

prod: $(files)
	go build -o {{.ModuleName}}.out -ldflags "-s -w" $(files)

clean:
	rm -f {{.ModuleName}}.out

.PHONY: all clean
`

var gitignoreDev = `
# Go template downloaded with gut
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work
.gut

# Dev files
*.log
devManifest.*
.init

dist/
`

var templateDevManifest = `
{
  "executable": "{{.ModuleName}}.out",
  "build_command": "make",
  "user_config": {
    "default": {
      "my_token": "Bearer"
    }
  },
  "tables": [
    "{{.TableName}}"
  ],
  "log_file": "dev.log",
  "log_level": "debug"
}`

var templateGoreleaserDev = `
version: 2

before:
  hooks:
    # You may remove this if you don't use go modules.
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    binary: {{.ModuleName}}
    id: anyquery
    ldflags: "-s -w"

    goarch:
      - amd64
      - arm64

archives:
  - format: binary

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
`

var templateProdManifest = `
name = "{{.ModuleName}}"
version = "0.1.0"
description = "My awesome plugin description"
author = "A fantastic developer"
license = "UNLICENSED"
repository = ""
homepage = ""
type = "anyquery"
minimumAnyqueryVersion = "0.0.1"

tables = ["{{.TableName}}"]

# The user configuration schema
[[userConfig]]
name = "my_token"
description = "A description of why this token is needed"
type = "string"
required = true # If the user must provide a value

[[file]]
platform = "linux/amd64"
directory = "dist/anyquery_linux_amd64_v1"
executablePath = "{{.ModuleName}}"

[[file]]
platform = "linux/arm64"
directory = "dist/anyquery_linux_arm64"
executablePath = "{{.ModuleName}}"

[[file]]
platform = "darwin/amd64"
directory = "dist/anyquery_darwin_amd64_v1"
executablePath = "{{.ModuleName}}"

[[file]]
platform = "darwin/arm64"
directory = "dist/anyquery_darwin_arm64"
executablePath = "{{.ModuleName}}"

[[file]]
platform = "windows/amd64"
directory = "dist/anyquery_windows_amd64_v1"
executablePath = "{{.ModuleName}}.exe"

[[file]]
platform = "windows/arm64"
directory = "dist/anyquery_windows_arm64"
executablePath = "{{.ModuleName}}.exe"
`

var mainTemplate = template.Must(template.New("main").Parse(templateMainGo))

var tableTemplate = template.Must(template.New("table").Parse(templateTableGo))

var makefileTemplate = template.Must(template.New("makefile").Parse(templateMakefile))

var manifestDevTemplate = template.Must(template.New("manifest").Parse(templateDevManifest))

var goreleaserDevTemplate = template.Must(template.New("goreleaser").Parse(templateGoreleaserDev))

var manifestProdTemplate = template.Must(template.New("manifest").Parse(templateProdManifest))

func replaceNonAlphanumeric(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, s)
}

func DevInit(cmd *cobra.Command, args []string) error {
	// Get the filename of the url in the arguments
	// If there is no filename, use the current directory as a name
	moduleName := path.Base(args[0])
	moduleURL := args[0]

	// If there is a second argument, use it as the directory
	var dir string
	if len(args) >= 2 {
		dir = args[1]
	} else {
		dir = "."
	}

	// Make the directory
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}

	err = os.Chdir(dir)
	if err != nil {
		return fmt.Errorf("could not change directory: %w", err)
	}

	// Ensure that the directory is empty
	fileEntry, err := os.ReadDir(".")
	if err != nil {
		return fmt.Errorf("could not read directory: %w", err)
	}

	if len(fileEntry) > 0 {
		return fmt.Errorf("directory is not empty")
	}

	// Replace any non-alphanumeric characters with an underscore
	moduleName = replaceNonAlphanumeric(moduleName)

	// Initialize the module
	goCmd := exec.Command("go", "mod", "init", args[0])
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr

	fmt.Println("⏩︎ Initializing module")
	err = goCmd.Run()
	if err != nil {
		return fmt.Errorf("could not initialize module: %w", err)
	}

	// Import the rpc package
	goCmd = exec.Command("go", "get", "-u", "github.com/julien040/anyquery/rpc")
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	err = goCmd.Run()
	if err != nil {
		return fmt.Errorf("could not import rpc package: %w", err)
	}

	// Create the main.go file
	mainFile, err := os.Create("main.go")
	if err != nil {
		return fmt.Errorf("could not create main.go: %w", err)
	}
	defer mainFile.Close()

	err = mainTemplate.Execute(mainFile, templateDevArgs{
		ModuleName: moduleName,
		ModuleURL:  moduleURL,
		TableName:  moduleName,
		FileName:   "main.go",
	})
	if err != nil {
		return fmt.Errorf("could not execute main template: %w", err)
	}

	// Create the table.go file
	tableFile, err := os.Create(moduleName + ".go")
	if err != nil {
		return fmt.Errorf("could not create table.go: %w", err)
	}
	defer tableFile.Close()

	err = tableTemplate.Execute(tableFile, templateDevArgs{
		ModuleName: moduleName,
		ModuleURL:  moduleURL,
		TableName:  moduleName,
		FileName:   moduleName + ".go",
	})
	if err != nil {
		return fmt.Errorf("could not execute table template: %w", err)
	}

	// Create the Makefile
	makefile, err := os.Create("Makefile")
	if err != nil {
		return fmt.Errorf("could not create Makefile: %w", err)
	}
	defer makefile.Close()

	err = makefileTemplate.Execute(makefile, templateDevArgs{
		ModuleName: moduleName,
		ModuleURL:  moduleURL,
		TableName:  moduleName,
		FileName:   "Makefile",
	})

	if err != nil {
		return fmt.Errorf("could not execute makefile template: %w", err)
	}

	// Create the .gitignore file
	err = os.WriteFile(".gitignore", []byte(gitignoreDev), 0644)
	if err != nil {
		return fmt.Errorf("could not create .gitignore: %w", err)
	}

	// Create the devManifest.json file
	manifestFile, err := os.Create("devManifest.json")
	if err != nil {
		return fmt.Errorf("could not create devManifest.json: %w", err)
	}

	err = manifestDevTemplate.Execute(manifestFile, templateDevArgs{
		ModuleName: moduleName,
		ModuleURL:  moduleURL,
		TableName:  moduleName,
		FileName:   "devManifest.json",
	})

	if err != nil {
		return fmt.Errorf("could not execute manifest template: %w", err)
	}

	// Create the .goreleaser.yaml file
	goreleaserFile, err := os.Create(".goreleaser.yaml")
	if err != nil {
		return fmt.Errorf("could not create .goreleaser.yaml: %w", err)
	}

	err = goreleaserDevTemplate.Execute(goreleaserFile, templateDevArgs{
		ModuleName: moduleName,
		ModuleURL:  moduleURL,
		TableName:  moduleName,
		FileName:   ".goreleaser.yaml",
	})

	if err != nil {
		return fmt.Errorf("could not execute goreleaser template: %w", err)
	}

	// Create the manifest.toml file
	manifestFile, err = os.Create("manifest.toml")
	if err != nil {
		return fmt.Errorf("could not create manifest.toml: %w", err)
	}

	err = manifestProdTemplate.Execute(manifestFile, templateDevArgs{
		ModuleName: moduleName,
		ModuleURL:  moduleURL,
		TableName:  moduleName,
		FileName:   "manifest.toml",
	})

	if err != nil {
		return fmt.Errorf("could not execute manifest template: %w", err)
	}

	// Run a final go mod tidy
	goCmd = exec.Command("go", "mod", "tidy")
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	fmt.Println("⏩︎ Running go mod tidy")
	err = goCmd.Run()
	if err != nil {
		return fmt.Errorf("could not run go mod tidy: %w", err)
	}

	fmt.Println()
	fmt.Println("🎉️ Done! Your module is ready to be developed")
	if dir != "." {
		fmt.Println("📂 Open the directory with cd", dir)
	}
	fmt.Println("👉️ Run 'make' to build the module")
	fmt.Println("🧑‍💻 To develop, launch `anyquery --dev` in developer mode with the flag --dev")
	fmt.Printf("🔍 And load the plugin by running SELECT load_dev_plugin('%s', 'devManifest.json');\n", moduleName)
	fmt.Printf("🔃 To build and reload the plugin, run `SELECT reload_dev_plugin('%s');\n", moduleName)

	return nil
}

func DevNewTable(cmd *cobra.Command, args []string) error {
	// Get the table name
	tableName := args[0]

	tableName = strings.TrimSpace(tableName)
	tableName = strings.TrimSuffix(tableName, ".go")

	// Replace any non-alphanumeric characters with an underscore
	tableName = replaceNonAlphanumeric(tableName)

	// Create the table.go file
	tableFile, err := os.Create(tableName + ".go")
	if err != nil {
		return fmt.Errorf("could not create table.go: %w", err)
	}
	defer tableFile.Close()

	err = tableTemplate.Execute(tableFile, templateDevArgs{
		ModuleName: tableName,
		ModuleURL:  "",
		TableName:  tableName,
		FileName:   tableName + ".go",
	})
	if err != nil {
		return fmt.Errorf("could not execute table template: %w", err)
	}

	fmt.Println("🎉️ Done! Your table is ready to be developed")
	fmt.Printf("👉️ Add %s to the arguments of rpc.NewPlugin e.g. plugin := rpc.NewPlugin(...otherTable, %sCreator)\n", tableName, tableName)
	fmt.Printf("👉️ And add %s to your devManifest e.g. \"tables\" : [..otherTables, \"%s\"]\n", tableName, tableName)

	return nil
}
