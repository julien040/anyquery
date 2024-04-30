package controller

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func ListPlugins(cmd *cobra.Command, args []string) error {
	// Open the database
	db, queries, err := requestDatabase(cmd.Flags())
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We get the plugins
	ctx := context.Background()
	rows, err := queries.GetPlugins(ctx)
	if err != nil {
		return fmt.Errorf("could not get the plugins: %w", err)
	}
	for row := range rows {
		fmt.Println(row)
	}

	fmt.Println(len(rows), "plugins found")

	return nil
}
