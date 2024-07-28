package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Tools to help you with using anyquery",
}

var toolHashDirCmd = &cobra.Command{
	Use:   "hashdir [dir]",
	Short: "Hash a directory",
	Long:  `Hash a directory and return a value that can be used in a plugin manifest.`,
	Args:  cobra.ExactArgs(1),
	RunE:  controller.HashDir,
}

var toolMySQLPasswordCmd = &cobra.Command{
	Use:   "mysql-password",
	Short: "Hash a password from stdin to be used in an authentification file",
	Long: `Hash a password from stdin to be used in an authentification file.
The password is hashed using the mysql_native_password algorithm
which can be summarized as HEX(SHA1(SHA1(password)))`,
	Aliases: []string{"hash-password", "mysql-native-password"},
	RunE:    controller.MySQLPassword,
}

var toolDevCmd = &cobra.Command{
	Use:     "dev",
	Aliases: []string{"development", "developer", "developers"},
	Short:   "Development tools",
}

var toolDevInitCmd = &cobra.Command{
	Use:   "init [module URL] [dir]",
	Short: "Initialize a new plugin",
	Long: `Initialize a new plugin in the specified directory. If no directory is specified, the current directory is used.
	The module URL is the go mod URL of the plugin.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: controller.DevInit,
}

var toolDevNewTableCmd = &cobra.Command{
	Use:     "new-table [table name]",
	Short:   "Write the boilerplate for a new table",
	Aliases: []string{"newtable"},
	Long: `Write the boilerplate for a new table in the specified file.
	The table name must only contain alphanumeric characters and underscores. Other characters will be replaced by underscores.`,
	Args: cobra.ExactArgs(1),
	RunE: controller.DevNewTable,
}

var toolGenerateDocCmd = &cobra.Command{
	Use:   "generate-doc [dir]",
	Short: "Generate the markdown documentation of the CLI",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doc.GenMarkdownTree(rootCmd, args[0])
	},
}


func init() {
	rootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(toolHashDirCmd)
	toolCmd.AddCommand(toolMySQLPasswordCmd)
	toolCmd.AddCommand(toolDevCmd)
	toolDevCmd.AddCommand(toolDevInitCmd)
	toolDevCmd.AddCommand(toolDevNewTableCmd)
	toolCmd.AddCommand(toolGenerateDocCmd)
}
