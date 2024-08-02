package controller

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/adrg/xdg"
	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/namespace"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const initScript = `
-- Create the dual table that is used by some SHOW commands
CREATE VIEW IF NOT EXISTS dual AS SELECT 'x' AS dummy;
`

func Query(cmd *cobra.Command, args []string) error {
	path := "anyquery.db"
	var inMemory, readOnly, devMode bool

	// Get the flags
	path, _ = cmd.Flags().GetString("database")
	inMemory, _ = cmd.Flags().GetBool("in-memory")
	readOnly, _ = cmd.Flags().GetBool("readonly")
	if !readOnly {
		readOnly, _ = cmd.Flags().GetBool("read-only")
	}

	// Get the file path from the args
	// The arg takes precedence over the flag
	if len(args) > 0 {
		path = args[0]
	}

	if path == ":memory:" {
		inMemory = true
	}

	// If the path is empty, we open an in-memory database
	if path == "" {
		inMemory = true
	}

	if inMemory {
		// Does not matter so we set it to a random value
		// We set it a random value because if the database is in memory,
		// the first arg will be the query, and not the database path
		path = "myrandom.db"
	}

	devMode, _ = cmd.Flags().GetBool("dev")

	// Create the logger
	var outputLog io.Writer
	logFile, _ := cmd.Flags().GetString("log-file")
	if logFile != "" {
		outputLog, _ = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		outputLog = io.Discard
	}

	logAsJSON := false
	logFormat, _ := cmd.Flags().GetString("log-format")
	if logFormat == "json" {
		logAsJSON = true
	}

	logLevelFlag, _ := cmd.Flags().GetString("log-level")
	logLevel := hclog.LevelFromString(logLevelFlag)

	namespace, err := namespace.NewNamespace(namespace.NamespaceConfig{
		InMemory: inMemory,
		ReadOnly: readOnly,
		Path:     path,
		Logger: hclog.New(
			&hclog.LoggerOptions{
				Output:     outputLog,
				JSONFormat: logAsJSON,
				Level:      logLevel,
			},
		),
		DevMode: devMode,
	})
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	anyqueryConfigPath := ""
	anyqueryConfigPath, err = cmd.Flags().GetString("config")
	if anyqueryConfigPath == "" {
		anyqueryConfigPath, err = xdg.ConfigFile("anyquery/config.db")
		if err != nil {
			return err
		}
	}

	err = namespace.LoadAsAnyqueryCLI(anyqueryConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get the extensions
	extensions, _ := cmd.Flags().GetStringSlice("extension")
	for _, extension := range extensions {
		err = namespace.LoadSharedExtension(extension, "")
		if err != nil {
			return fmt.Errorf("failed to load extension: %w", err)
		}
	}

	// Register the namespace
	db, err := namespace.Register("main")
	if err != nil {
		return fmt.Errorf("failed to register namespace: %w", err)
	}
	defer db.Close()

	// Run the init script if the database is not read-only
	if !readOnly {
		_, err = db.Exec(initScript)
		if err != nil {
			return fmt.Errorf("failed to run init script: %w", err)
		}
	}

	// Listen for signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		db.Close()
		os.Exit(0)
	}()

	// Create the shell
	shell := shell{
		DB: db,
		Middlewares: []middleware{
			middlewareSlashCommand, middlewareDotCommand,
			middlewarePRQL, middlewarePQL,
			middlewareMySQL, middlewareFileQuery,
			middlewareQuery,
		},
		Config: middlewareConfiguration{
			"dot-command":   true,
			"mysql":         true,
			"slash-command": true,
		},
		Namespace:      namespace,
		OutputFile:     "stdout",
		OutputFileDesc: os.Stdout,
	}

	// Check if an alternative language is provided
	language, _ := cmd.Flags().GetString("language")
	if language == "prql" || language == "pql" {
		shell.Config.SetString("language", language)
	}

	prql, _ := cmd.Flags().GetBool("prql")
	if prql {
		shell.Config.SetString("language", "prql")
	}

	pql, _ := cmd.Flags().GetBool("pql")
	if pql {
		if prql {
			return fmt.Errorf("cannot use both PRQL and PQL at the same time")
		}
		shell.Config.SetString("language", "pql")
	}

	// Check if stdout is a file
	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile != "" {
		file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("could not open output file: %w", err)
		}
		shell.OutputFile = outputFile
		shell.OutputFileDesc = file
	}

	// Check if the output file is a tty
	// If not, we set the output mode to plain
	if !term.IsTerminal(int(shell.OutputFileDesc.Fd())) {
		shell.Config["outputMode"] = "plain"
	}

	// Get the output mode if defined by the user
	outputFormat, _ := cmd.Flags().GetString("format")
	if outputFormat != "" {
		if _, ok := formatName[outputFormat]; ok {
			shell.Config["outputMode"] = outputFormat
		} else {
			return fmt.Errorf("invalid output format: %s", outputFormat)
		}
	}
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		shell.Config["outputMode"] = "json"
	}
	csvOutput, _ := cmd.Flags().GetBool("csv")
	if csvOutput {
		shell.Config["outputMode"] = "csv"
	}
	plainOutput, _ := cmd.Flags().GetBool("plain")
	if plainOutput {
		shell.Config["outputMode"] = "plain"
	}

	// Run the init scripts
	initScripts, _ := cmd.Flags().GetStringArray("init")
	for _, script := range initScripts {
		// Read the file
		file, err := os.Open(script)
		if err != nil {
			return fmt.Errorf("could not open init file: %w", err)
		}
		content, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("could not read init file: %w", err)
		}

		// Run the script
		shell.Run(string(content))
	}

	// The query command has 3 main modes:
	// - If a query is provided as an argument after the file path, or with the flag --query/-c,
	//   it will be executed and the command will exit
	// - If a query is provided on stdin, it will be executed as soon as a query is provided
	//   and will continue while stdin is open
	// - If no query is provided, and stdin/stdout is a terminal, the command will enter in shell mode

	queryArgs := ""

	// If the database is a file, the second arg is the query
	if len(args) > 1 {
		queryArgs = args[1]
	} else if len(args) > 0 && inMemory {
		// If the database is in memory, the first arg is the query
		queryArgs = args[0]
	}

	queryFlag, _ := cmd.Flags().GetString("query")
	if queryFlag != "" {
		queryArgs = queryFlag
	}

	if queryArgs != "" {
		shell.Run(queryArgs)
		return nil
	}

	if !isSTDinAtty() {
		// Read each query until a delimiter (\n for commands, ; for SQL) is found
		// Execute the query, print the result, and continue until EOF
		//
		// We could have just used io.ReadAll, but by using this approach, query
		// can be streamed and executed as soon as it is provided
		query := strings.Builder{}
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err := reader.ReadString('\n')
			query.WriteString(line)
			if err != nil {
				break
			}
			// We check if the query starts with a dot or a slash
			// If so, we execute it immediately
			if line[0] == '.' || line[0] == '/' {
				shell.Run(query.String())
				query.Reset()
			}

			// Otherwise, we check if the query ends with a semicolon
			// If so, we execute it immediately
			if line[len(line)-2] == ';' {
				shell.Run(query.String())
				query.Reset()
			}

			// If the query is empty, we continue
			if query.String() == "" {
				continue
			}

			// And if the query is not filled, it will be in the next iteration
		}
		// Once we reach EOF, we execute the last query
		shell.Run(query.String())
		return nil
	}

	// Ensure that stdin and stdout are not both a pipe
	// If not, default to shell mode
	if !isSTDinAtty() && !isSTDoutAtty() {
		return fmt.Errorf("stdin and stdout cannot be both a pipe when no query is provided")
	}

	// Run the shell

	// We print the welcome message
	fmt.Println("Welcome to anyquery!")
	fmt.Println("Install plugins by running `anyquery install [plugin]`")
	fmt.Println("Run `anyquery` to open a shell to run queries")
	fmt.Println("Run `anyquery --help` to see the available commands")
	fmt.Println("Visit https://anyquery.dev/integrations to see the available plugins")
	fmt.Println()
	mustContinue := true
	for mustContinue {
		query := shell.InputQuery()
		if query == "" {
			continue
		}
		mustContinue = !shell.Run(query)

		if mustContinue {
			// We just print a new line to separate the queries
			fmt.Println()
		}

	}

	return nil
}
