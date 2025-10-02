package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var gptCmd = &cobra.Command{
	Use:     "gpt",
	Aliases: []string{"chat", "chatgpt"},
	Short:   "Open an HTTP server so that ChatGPT can do function calls",
	Long: `Open an HTTP server so that ChatGPT can do function calls. By default, it will expose a tunnel to the internet.

By setting the --host or --port flags, you can disable the tunnel and bind to a specific host and port. In this case, you will need to configure your LLM to connect to this host and port.
It will also enable the authorization token mechanism. By default, the token is randomly generated and can be found when starting the server. You can also provide a token using the ANYQUERY_AI_SERVER_BEARER_TOKEN environment variable.
This token must be supplied in the Authorization header of the request (prefixed with "Bearer "). You can also disable the authorization mechanism by setting the --no-auth flag.`,

	RunE: controller.Gpt,
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol (MCP) server",
	Long: `Start the Model Context Protocol (MCP) server. It is used to provide context for LLM that supports it. 
Pass the --stdio flag to use standard input/output for communication. By default, it will bind locally to localhost:8070 (modify it with the --host, --port and --domain flags).`,
	//You can also expose the tunnel to the internet by using the --tunnel flag (useful when the LLM is on a remote server).`,
	RunE: controller.Mcp,
}

func init() {
	// GPT command
	rootCmd.AddCommand(gptCmd)
	addFlag_commandModifiesConfiguration(gptCmd)
	gptCmd.Flags().StringP("database", "d", "", "Database to connect to (a path or :memory:)")
	gptCmd.Flags().Bool("in-memory", false, "Use an in-memory database")
	gptCmd.Flags().Bool("readonly", false, "Open the SQLite database in read-only mode")
	gptCmd.Flags().Bool("read-only", false, "Open the SQLite database in read-only mode")
	gptCmd.Flags().StringSlice("extension", []string{}, "Load one or more extensions by specifying their path. Separate multiple extensions with a comma.")
	gptCmd.Flags().String("log-file", "", "Log file")
	gptCmd.Flags().String("log-level", "info", "Log level (trace, debug, info, warn, error, off)")
	gptCmd.Flags().String("log-format", "text", "Log format (text, json)")
	gptCmd.Flags().String("host", "", "Host to bind to. If not empty, the tunnel will be disabled")
	gptCmd.Flags().Int("port", 0, "Port to bind to. If not empty, the tunnel will be disabled")
	gptCmd.Flags().Bool("no-auth", false, "Disable the authorization mechanism for locally bound servers")

	// MCP command
	rootCmd.AddCommand(mcpCmd)
	addFlag_commandModifiesConfiguration(mcpCmd)
	mcpCmd.Flags().String("host", "127.0.0.1", "Host to bind to")
	mcpCmd.Flags().String("domain", "", "Domain to use for the HTTP tunnel (empty to use the host)")
	mcpCmd.Flags().Bool("no-auth", false, "Disable the authorization mechanism for locally bound HTTP servers")
	mcpCmd.Flags().Int("port", 8070, "Port to bind to")
	mcpCmd.Flags().Bool("stdio", false, "Use standard input/output for communication")
	mcpCmd.Flags().Bool("tunnel", false, "Use an HTTP tunnel, and expose the server to the internet (when used, --host, --domain and --port are ignored)")
	mcpCmd.Flags().StringP("database", "d", "", "Database to connect to (a path or :memory:)")
	mcpCmd.Flags().Bool("in-memory", false, "Use an in-memory database")
	mcpCmd.Flags().Bool("readonly", false, "Open the SQLite database in read-only mode")
	mcpCmd.Flags().Bool("read-only", false, "Open the SQLite database in read-only mode")
	mcpCmd.Flags().StringSlice("extension", []string{}, "Load one or more extensions by specifying their path. Separate multiple extensions with a comma.")
	mcpCmd.Flags().String("log-file", "", "Log file")
	mcpCmd.Flags().String("log-level", "info", "Log level (trace, debug, info, warn, error, off)")
	mcpCmd.Flags().String("log-format", "text", "Log format (text, json)")

}
