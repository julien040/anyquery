package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Lets you connect to anyquery remotely",
	Long: `Listens for incoming connections and allows you to run queries
using any postgres client.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("server called")
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringP("host", "h", "localhost", "Host to listen on")
	serverCmd.Flags().IntP("port", "p", 5432, "Port to listen on")
	serverCmd.Flags().StringP("database", "d", "anyquery", "Database to connect to (a path or :memory:)")
	serverCmd.Flags().Bool("readonly", false, "Start the server in read-only mode")
}
