package controller

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/julien040/anyquery/namespace"
	"github.com/spf13/cobra"
)

func ConnectionList(cmd *cobra.Command, args []string) error {
	// Open the database on read-only mode
	db, querier, err := requestDatabase(cmd.Flags(), true)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// Get the connections
	connections, err := querier.GetConnections(context.Background())
	if err != nil {
		return fmt.Errorf("could not get the connections: %w", err)
	}

	// Print the connections
	o := outputTable{
		Columns: []string{"Name", "Type", "Connection string", "Filter"},
		Writer:  os.Stdout,
	}
	o.InferFlags(cmd.Flags())

	for _, c := range connections {
		o.AddRow(c.Connectionname, c.Databasetype, c.Urn, c.Celscript)
	}

	return o.Close()
}

func connectionNameExists(name string, querier *model.Queries) bool {
	_, err := querier.GetConnection(context.Background(), name)
	return err == nil
}

func validateConnectionName(name string, querier *model.Queries) error {
	if !alphaNumeric.MatchString(name) {
		return fmt.Errorf("connection name must be alphanumeric")
	}
	if connectionNameExists(name, querier) {
		return fmt.Errorf("connection %s already exists", name)
	}
	return nil
}

func validateConnectionURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("connection string cannot be empty")
	}
	return nil
}

var alphaNumeric = regexp.MustCompile("^[a-zA-Z0-9_]+$")

func ConnectionAdd(cmd *cobra.Command, args []string) error {
	// Open the database on read-write mode
	db, querier, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// Ask for the connection name, type, connection string, and filter
	// We first try to get the values from the arguments
	// If they are not provided, we ask the user for them

	fields := []huh.Field{}
	connectionName := ""
	databaseType := ""
	connectionString := ""
	filter := ""
	if len(args) > 0 {
		connectionName = args[0]
		if connectionNameExists(connectionName, querier) {
			return fmt.Errorf("connection %s already exists", connectionName)
		}
	} else {
		fields = append(fields, huh.NewInput().
			Title("Connection name").
			Validate(func(s string) error {
				return validateConnectionName(s, querier)
			}).
			Description("The name of the connection. All the imported tables will be prefixed with this name.").
			Value(&connectionName))

	}
	if len(args) > 1 {
		databaseType = args[1]
		if !slices.Contains(namespace.SupportedConnections, databaseType) {
			return fmt.Errorf("unsupported connection type %s. Make sure it's one of %s. Also ensure Anyquery is up to date.", databaseType, strings.Join(namespace.SupportedConnections, ", "))
		}
	} else {
		options := make([]huh.Option[string], len(namespace.SupportedConnections))
		for i, c := range namespace.SupportedConnections {
			options[i] = huh.NewOption(c, c)
		}
		fields = append(fields, huh.NewSelect[string]().
			Title("Connection type").Options(options...).
			Description(fmt.Sprintf("The type of the connection. Supported types are %s", strings.Join(namespace.SupportedConnections, ", "))).
			Value(&databaseType))
	}

	if len(args) > 2 {
		connectionString = args[2]
		if err := validateConnectionURL(connectionString); err != nil {
			return err
		}
	} else {
		fields = append(fields, huh.NewInput().
			Title("Connection string (URL)").
			Description("The connection string to the database. For example, for MySQL, it's 'username:password@tcp(host:1234)/database'").
			Validate(validateConnectionURL).
			Value(&connectionString))
	}
	if len(args) > 3 {
		filter = args[3]
	} else {
		fields = append(fields, huh.NewText().
			Title("Filter (cel expression)").
			Placeholder("true").
			ShowLineNumbers(true).
			Description("A CEL expression to filter the tables to import. For example, table.name in ['table1', 'table2''\nLeave \"true\" to import all tables.\n"+
				"Refer to https://anyquery.dev/docs/database/cel-script/ for more information").
			Value(&filter))
	}

	// Ask the user for the values if they are not provided
	if len(fields) > 0 {
		if !isSTDinAtty() || !isSTDoutAtty() {
			return fmt.Errorf("interactive mode is required to add a connection. Otherwise, provide the connection name, type, connection string, and filter as arguments")
		}
		grp := huh.NewGroup(fields...).Title("Connection information").Description("Let's add a new database connection to Anyquery")
		err := huh.NewForm(grp).Run()
		if err != nil {
			return fmt.Errorf("could not ask for the connection information: %w", err)
		}
	}

	if filter == "" {
		filter = "true" // Default filter to import all tables
	}

	// Add the connection
	err = querier.AddConnection(context.Background(), model.AddConnectionParams{
		Connectionname:     connectionName,
		Databasetype:       databaseType,
		Urn:                connectionString,
		Celscript:          filter,
		Additionalmetadata: "{}",
	})

	if err != nil {
		return fmt.Errorf("could not add the connection to the database: %w", err)
	}

	fmt.Printf("✅ Successfully added connection %s\n", connectionName)

	return nil
}

func ConnectionRemove(cmd *cobra.Command, args []string) error {
	// Open the database on read-write mode
	db, querier, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// Ask for the connection name
	connectionName := ""
	if len(args) > 0 {
		connectionName = args[0]
	}

	if connectionName == "" {
		if !isSTDinAtty() || !isSTDoutAtty() {
			return fmt.Errorf("interactive mode is required to remove a connection. Otherwise, provide the connection name as an argument")
		}

		options := []huh.Option[string]{}
		connections, err := querier.GetConnections(context.Background())
		if err != nil {
			return fmt.Errorf("could not get the connections: %w", err)
		}
		for _, c := range connections {
			options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", c.Connectionname, c.Databasetype), c.Connectionname))
		}

		if len(options) == 0 {
			return fmt.Errorf("no connections found to remove")
		}

		err = huh.NewSelect[string]().
			Title("Connection to remove").
			Options(options...).
			Value(&connectionName).
			Run()
		if err != nil {
			return fmt.Errorf("could not ask for the connection to remove: %w", err)
		}

	}

	// Ensure the connection exists
	if !connectionNameExists(connectionName, querier) {
		return fmt.Errorf("connection %s does not exist", connectionName)
	}

	// Remove the connection
	err = querier.DeleteConnection(context.Background(), connectionName)
	if err != nil {
		return fmt.Errorf("could not remove the connection: %w", err)
	}

	fmt.Printf("✅ Successfully removed connection %s\n", connectionName)

	return nil
}
