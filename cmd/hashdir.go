package cmd

import (
	"github.com/julien040/anyquery/controller"
	"github.com/spf13/cobra"
)

var hashDirCmd = &cobra.Command{
	Use:   "hashdir [path]",
	Short: "Hash a directory and print the result",
	Long: `Hash a directory and print the result.
If no path is provided, the current directory is used.
This is mainly used to write the manifest file for a plugin.

The hash is computed using https://pkg.go.dev/golang.org/x/mod/sumdb/dirhash`,

	RunE: controller.HashDir,
}

func init() {
	rootCmd.AddCommand(hashDirCmd)
}
