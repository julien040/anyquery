package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Lets you connect to anyquery remotely",
	Long: `Listens for incoming connections and allows you to run queries
using any MySQL client.`,
	RunE: controller.Server,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().String("host", "127.0.0.1", "Host to listen on")
	serverCmd.Flags().IntP("port", "p", 8070, "Port to listen on")
	serverCmd.Flags().StringP("database", "d", "anyquery.db", "Database to connect to (a path or :memory:)")
	serverCmd.Flags().Bool("in-memory", false, "Use an in-memory database")
	serverCmd.Flags().Bool("readonly", false, "Start the server in read-only mode")
	serverCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	serverCmd.Flags().String("log-format", "text", "Log format (text, json)")
	serverCmd.Flags().String("log-file", "/dev/stdout", "Log file")
	serverCmd.Flags().String("auth-file", "", "Path to the authentication file")
	serverCmd.Flags().Bool("dev", false, "Run the program in developer mode")
	serverCmd.Flags().StringSlice("extension", []string{}, "Load one or more extensions by specifying their path. Separate multiple extensions with a comma.")

	addFlag_commandModifiesConfiguration(serverCmd)
}
