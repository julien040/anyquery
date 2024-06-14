package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/spf13/cobra"
)

func AliasList(cmd *cobra.Command, args []string) error {
	// Open the database
	db, queries, err := requestDatabase(cmd.Flags(), true)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We get the aliases
	ctx := context.Background()
	aliases, err := queries.GetAliases(ctx)
	if err != nil {
		return fmt.Errorf("could not get the aliases: %w", err)
	}

	// We print the aliases
	output := outputTable{
		Writer:  os.Stdout,
		Columns: []string{"Alias", "Table"},
	}
	output.InferFlags(cmd.Flags())
	for _, alias := range aliases {
		output.AddRow(alias.Alias, alias.Tablename)
	}

	return output.Close()
}

func AliasAdd(cmd *cobra.Command, args []string) error {
	// Open the database
	db, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We check the arguments
	// If the user provided the alias and the table, we use them
	// Otherwise, we ask the user for them
	var alias, table string

	if len(args) == 2 {
		alias = args[0]
		table = args[1]
	} else {
		aliasInput := huh.NewInput().Title("How would like to call this table?").
			Description("The alias name must be unique and not already used by a table.").
			Placeholder("myawesomealias").
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("the alias cannot be empty")
				}

				// We check if the alias is already used
				ctx := context.Background()
				alias, err := queries.GetAliasOf(ctx, s)
				if err == nil {
					return fmt.Errorf("the alias %s is already used by the table %s", s, alias.Tablename)
				}

				return nil
			}).Value(&alias)

		tableNamePossibleValues, err := listTableName(queries)
		if err != nil {
			return fmt.Errorf("could not list the table names: %w", err)
		}

		options := make([]huh.Option[string], len(tableNamePossibleValues))
		for i, tableName := range tableNamePossibleValues {
			options[i] = huh.Option[string]{Value: tableName, Key: tableName}
		}

		tableInput := huh.NewSelect[string]().Title("Which table do you want to alias?").
			Options(options...).Value(&table)

		group := huh.NewGroup(aliasInput, tableInput)
		err = huh.NewForm(group).Run()
		if err != nil {
			if err == huh.ErrUserAborted {
				return nil
			}
			return fmt.Errorf("could not get the alias and the table: %w", err)
		}
	}

	// We ensure the alias is unique
	ctx := context.Background()
	_, err = queries.GetAliasOf(ctx, alias)
	if err == nil {
		return fmt.Errorf("the alias %s is already used by a table", alias)
	}

	// We add the alias
	err = queries.AddAlias(ctx, model.AddAliasParams{
		Tablename: table,
		Alias:     alias,
	})
	if err != nil {
		return fmt.Errorf("could not add the alias: %w", err)
	}
	fmt.Printf("✅ The alias %s has been added for the table %s\n", alias, table)

	return nil
}

func AliasDelete(cmd *cobra.Command, args []string) error {
	// Open the database
	db, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// We check the arguments
	// If the user provided the alias, we use it
	// Otherwise, we ask the user for it
	var alias string
	if len(args) > 0 {
		alias = args[0]
	} else {
		aliasList, err := queries.GetAliases(context.Background())
		if err != nil {
			return fmt.Errorf("could not get the aliases: %w", err)
		}

		options := make([]huh.Option[string], len(aliasList))
		for i, alias := range aliasList {
			options[i] = huh.Option[string]{
				Value: alias.Alias,
				Key:   alias.Alias + " (" + alias.Tablename + ")",
			}
		}

		aliasInput := huh.NewSelect[string]().Title("Which alias do you want to delete?").
			Options(options...).Value(&alias)

		err = huh.NewForm(huh.NewGroup(aliasInput)).Run()
		if err != nil {
			if err == huh.ErrUserAborted {
				return nil
			}
			return fmt.Errorf("could not get the alias: %w", err)
		}
	}

	// We ensure the alias exists
	_, err = queries.GetAliasOf(context.Background(), alias)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("the alias %s does not exist", alias)
		}
		return fmt.Errorf("could not get the alias: %w", err)
	}

	// We delete the alias
	err = queries.DeleteAlias(context.Background(), alias)
	if err != nil {
		return fmt.Errorf("could not delete the alias: %w", err)
	}
	fmt.Printf("✅ The alias %s has been deleted\n", alias)

	return nil
}
