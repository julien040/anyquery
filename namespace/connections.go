package namespace

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5"
)

type sqlTable struct {
	Schema     string
	TableName  string
	TableType  string
	TableOwner string
}

// Filter the tables based on the filter string (CEL expression)
// and return the filtered tables
func filterTables(tables []sqlTable, filter string, log hclog.Logger) (filteredTables []sqlTable, err error) {
	filteredTables = []sqlTable{}

	// If the filter is empty, we return all the tables
	if filter == "" {
		return tables, nil
	}

	env, err := cel.NewEnv(
		cel.Variable("table", cel.MapType(cel.StringType, cel.StringType)),
		cel.Variable("all_tables", cel.ListType(cel.MapType(cel.StringType, cel.StringType))),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create the CEL environment: %w", err)
	}

	parsed, issues := env.Parse(filter)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("could not parse the CEL expression: %w", issues.Err())
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("could not check the CEL expression: %w", issues.Err())
	}

	program, err := env.Program(checked)
	if err != nil {
		return nil, fmt.Errorf("could not create the CEL program: %w", err)
	}

	allTables := []map[string]string{}
	for _, table := range tables {
		allTables = append(allTables, map[string]string{
			"schema": table.Schema,
			"name":   table.TableName,
			"type":   table.TableType,
			"owner":  table.TableOwner,
		})
	}

	for i, table := range tables {
		celArgs := map[string]interface{}{
			"table":      allTables[i],
			"all_tables": allTables,
		}

		out, _, err := program.Eval(celArgs)
		if err != nil {
			return nil, fmt.Errorf("could not evaluate the CEL program: %w", err)
		}

		// If the result is a bool and true, we add the table to the list
		if val, ok := out.Value().(bool); ok && val {
			filteredTables = append(filteredTables, table)
		} else if !ok {
			log.Warn("The CEL expression did not return a boolean value", "value", out.Value(), "table", table)
		}

	}

	return
}

// Fetch the list of tables from the database
// and return a list of exec statement to run so that the tables are imported in Anyquery
func RegisterExternalPostgreSQL(params LoadDatabaseConnectionParams, logger hclog.Logger) (statements []string, args [][]driver.Value, err error) {
	statements = []string{}
	args = [][]driver.Value{}

	// Create the connection
	conn, err := pgx.Connect(context.Background(), params.ConnectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to the database: %w", err)
	}
	defer conn.Close(context.Background())

	// Get the list of tables
	query := "SELECT table_schema, table_name, table_type FROM information_schema.tables"
	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get the list of tables: %w", err)
	}
	defer rows.Close()

	tables := []sqlTable{}

	for rows.Next() {
		table := sqlTable{}
		err = rows.Scan(&table.Schema, &table.TableName, &table.TableType)
		if err != nil {
			return nil, nil, fmt.Errorf("could not scan the row: %w", err)
		}

		// We add the table to the list
		tables = append(tables, table)
	}

	// Filter the tables
	filteredTables, err := filterTables(tables, params.Filter, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("could not filter the tables: %w", err)
	}

	// Create the statements

	// First, we attach a schema to the connection. This is just an in-memory database to have a schema name
	statements = append(statements, "ATTACH DATABASE ? AS ?")
	args = append(args, []driver.Value{"file:hello_you?mode=memory&cache=shared", params.SchemaName})
	for _, table := range filteredTables {
		// Compute the table name
		tableName := strings.Builder{}
		tableName.WriteString(params.SchemaName)
		tableName.WriteString(".")
		if table.Schema != "public" {
			tableName.WriteString(fmt.Sprintf("%s_", table.Schema))
		}
		tableName.WriteString(table.TableName)

		pgTableName := strings.Builder{}
		if table.Schema != "public" {
			pgTableName.WriteString(params.SchemaName)
			pgTableName.WriteString(".")
		}
		pgTableName.WriteString(table.TableName)

		// Create the virtual table and its mapping
		statements = append(statements, fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s USING postgres_reader('%s', '%s')", tableName.String(), params.ConnectionString, pgTableName.String()))
		args = append(args, []driver.Value{})
	}

	return
}

// Fetch the list of tables from the database
// and return a list of exec statement to run so that the tables are imported in Anyquery
func RegisterExternalMySQL(params LoadDatabaseConnectionParams, logger hclog.Logger) (statements []string, args [][]driver.Value, err error) {
	statements = []string{}
	args = [][]driver.Value{}

	return
}

// Fetch the list of tables from the database
// and return a list of exec statement to run so that the tables are imported in Anyquery
func RegisterExternalSQLite(params LoadDatabaseConnectionParams, logger hclog.Logger) (statements []string, args [][]driver.Value, err error) {
	statements = []string{}
	args = [][]driver.Value{}

	return
}
