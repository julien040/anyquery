package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func ListPlugins(cmd *cobra.Command, args []string) error {
	// Open the database
	db, queries, err := requestDatabase(cmd.Flags(), true)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We get the plugins
	ctx := context.Background()
	_, err = queries.GetPlugins(ctx)
	if err != nil {
		return fmt.Errorf("could not get the plugins: %w", err)
	}

	rows, err := db.Query("SELECT * FROM plugin_installed")
	if err != nil {
		return fmt.Errorf("could not get the plugins: %w", err)
	}

	// We print the plugins
	output := outputTable{
		Writer: os.Stdout,
	}

	output.InferFlags(cmd.Flags())
	err = output.WriteSQLRows(rows)
	if err != nil {
		return fmt.Errorf("could not write the plugins: %w", err)
	}

	err = output.Close()
	if err != nil {
		return fmt.Errorf("could not close the output: %w", err)
	}

	return nil
}
