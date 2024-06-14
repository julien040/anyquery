package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
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

func init() {
	rootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(toolHashDirCmd)
	toolCmd.AddCommand(toolMySQLPasswordCmd)
}
