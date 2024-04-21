package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Install and manage plugins",
	Long: `The store command allows you to install and manage plugins
from the anyquery store. Plugins are used to connect to data sources
and run queries on them.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("store called")
	},
}

func init() {
	rootCmd.AddCommand(storeCmd)
}
