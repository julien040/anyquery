package namespace

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	gocql "github.com/apache/cassandra-gocql-driver/v2"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/cel-go/cel"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5"
	"github.com/julien040/anyquery/other/duckdb"
	"github.com/samber/lo"
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

func registerExternalClickHouse(params LoadDatabaseConnectionParams, logger hclog.Logger) (statements []string, args [][]driver.Value, err error) {
	statements = []string{}
	args = [][]driver.Value{}

	// Create the connection
	opts, err := clickhouse.ParseDSN(params.ConnectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse the connection string: %w", err)
	}

	conn := clickhouse.OpenDB(opts)
	defer conn.Close()

	query := "SELECT database, name, engine FROM system.tables"
	rows, err := conn.QueryContext(context.Background(), query)
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
	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("could not get the list of tables: %w", rows.Err())
	}

	// Remove all the INFORMATION_SCHEMA tables (and leave the information_schema tables without the uppercase)
	// That's because ClickHouse has two schemas: information_schema and INFORMATION_SCHEMA
	// And SQLite does not handle well two tables with the same name in different cases
	// So we filter out the INFORMATION_SCHEMA tables
	tablesWithoutINFORMATION_SCHEMA := lo.Filter(tables, func(table sqlTable, _ int) bool {
		return table.Schema != "INFORMATION_SCHEMA"
	})

	// Filter the tables
	filteredTables, err := filterTables(tablesWithoutINFORMATION_SCHEMA, params.Filter, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("could not filter the tables: %w", err)
	}

	// Create the statements
	// First, we attach a schema to the connection. This is just an in-memory database to have a schema name
	statements = append(statements, "ATTACH DATABASE ? AS ?")
	args = append(args, []driver.Value{fmt.Sprintf("file:%s?mode=memory&cache=shared", params.SchemaName), params.SchemaName})

	for _, table := range filteredTables {
		// Compute the table name
		tableName := strings.Builder{}
		tableName.WriteString(params.SchemaName)
		tableName.WriteString(".")
		if table.Schema != "default" {
			tableName.WriteString(fmt.Sprintf("%s_", table.Schema))
		}
		tableName.WriteString(table.TableName)

		// The table name remote side
		chTableName := strings.Builder{}
		chTableName.WriteString(table.Schema)
		chTableName.WriteString(".")

		chTableName.WriteString(table.TableName)

		// Create the virtual table and its mapping
		statements = append(statements, fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s USING clickhouse_reader('%s', '%s')", tableName.String(), params.ConnectionString, chTableName.String()))
		args = append(args, []driver.Value{})
	}

	return
}

// Fetch the list of tables from the database
// and return a list of exec statement to run so that the tables are imported in Anyquery
func registerExternalPostgreSQL(params LoadDatabaseConnectionParams, logger hclog.Logger) (statements []string, args [][]driver.Value, err error) {
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
	args = append(args, []driver.Value{fmt.Sprintf("file:%s?mode=memory&cache=shared", params.SchemaName), params.SchemaName})
	for _, table := range filteredTables {
		// Compute the table name
		tableName := strings.Builder{}
		tableName.WriteString(params.SchemaName)
		tableName.WriteString(".")
		if table.Schema != "public" {
			tableName.WriteString(fmt.Sprintf("%s_", table.Schema))
		}
		tableName.WriteString(table.TableName)

		// The table name remote side
		pgTableName := strings.Builder{}
		if table.Schema != "public" {
			pgTableName.WriteString(table.Schema)
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
func registerExternalMySQL(params LoadDatabaseConnectionParams, logger hclog.Logger) (statements []string, args [][]driver.Value, err error) {
	statements = []string{}
	args = [][]driver.Value{}

	db, err := sql.Open("mysql", params.ConnectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("could not open the database: %w", err)
	}

	conf, err := mysql.ParseDSN(params.ConnectionString)
	if err != nil || conf == nil {
		return nil, nil, fmt.Errorf("could not parse the connection string: %w", err)
	}

	schema := conf.DBName

	// Get the list of tables
	query := "SELECT table_schema, table_name, table_type FROM information_schema.tables"
	tables := []sqlTable{}

	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get the list of tables: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		table := sqlTable{}
		err = rows.Scan(&table.Schema, &table.TableName, &table.TableType)
		if err != nil {
			return nil, nil, fmt.Errorf("could not scan the row: %w", err)
		}

		// We add the table to the list
		tables = append(tables, table)
	}

	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("could not get the list of tables: %w", rows.Err())
	}

	// Filter the tables
	filteredTables, err := filterTables(tables, params.Filter, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("could not filter the tables: %w", err)
	}

	// Create the statements
	// Similar to the PostgreSQL case, we attach a schema to the connection
	statements = append(statements, "ATTACH DATABASE ? AS ?")
	args = append(args, []driver.Value{fmt.Sprintf("file:%s?mode=memory&cache=shared", params.SchemaName), params.SchemaName})

	for _, table := range filteredTables {
		// Compute the table name
		tableName := strings.Builder{}
		tableName.WriteRune('`')
		tableName.WriteString(params.SchemaName)
		tableName.WriteString("`.`")
		if table.Schema != schema {
			tableName.WriteString(fmt.Sprintf("%s_", table.Schema))
		}
		tableName.WriteString(table.TableName)
		tableName.WriteRune('`')

		// The table name remote side
		mysqlTableName := strings.Builder{}
		mysqlTableName.WriteString(table.Schema)
		mysqlTableName.WriteRune('.')
		mysqlTableName.WriteString(table.TableName)

		// Create the virtual table and its mapping
		statements = append(statements, fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s USING mysql_reader('%s', '%s')", tableName.String(), params.ConnectionString, mysqlTableName.String()))
		args = append(args, []driver.Value{})

	}

	return
}

// Fetch the list of tables from the database
// and return a list of exec statement to run so that the tables are imported in Anyquery
func registerExternalSQLite(params LoadDatabaseConnectionParams, _ hclog.Logger) (statements []string, args [][]driver.Value, err error) {
	statements = []string{"ATTACH DATABASE ? AS ?"}
	args = [][]driver.Value{{params.ConnectionString, params.SchemaName}}

	return
}

func registerExternalDuckDB(params LoadDatabaseConnectionParams, logger hclog.Logger) (statements []string, args [][]driver.Value, err error) {
	statements = []string{}
	args = [][]driver.Value{}

	// Request all the tables
	rows, errChan := duckdb.RunDuckDBQuery(params.ConnectionString, "SELECT table_schema, table_name, table_type, FROM information_schema.tables;")
	if len(errChan) > 0 {
		rowErr := <-errChan
		if rowErr != nil {
			return nil, nil, fmt.Errorf("could not get the list of tables: %w", rowErr)
		}
	}
	tables := []sqlTable{}
	for row := range rows {
		table := sqlTable{}
		ok := true
		table.Schema, ok = row["table_schema"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("could not get the table schema from the row: %v", row)
		}
		table.TableName, ok = row["table_name"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("could not get the table name from the row:	 %v", row)
		}
		table.TableType, ok = row["table_type"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("could not get the table type from the row: %v", row)
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
	// First, we attach a schema to the connection. This is just an in-memory database
	statements = append(statements, "ATTACH DATABASE ? AS ?")
	args = append(args, []driver.Value{fmt.Sprintf("file:%s?mode=memory&cache=shared", params.SchemaName), params.SchemaName})
	for _, table := range filteredTables {
		// Compute the table name
		tableName := strings.Builder{}
		tableName.WriteString(params.SchemaName)
		tableName.WriteString(".")
		if table.Schema != "main" {
			tableName.WriteString(fmt.Sprintf("%s_", table.Schema))
		}
		tableName.WriteString(table.TableName)

		// The table name remote side
		duckDBTableName := strings.Builder{}
		if table.Schema != "main" {
			duckDBTableName.WriteString(table.Schema)
			duckDBTableName.WriteString(".")
		}
		duckDBTableName.WriteString(table.TableName)

		// Create the virtual table and its mapping
		statements = append(statements, fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s USING duckdb_reader(dsn='%s', table='%s')", tableName.String(), params.ConnectionString, duckDBTableName.String()))
		args = append(args, []driver.Value{})
	}

	return
}

func registerExternalCassandra(params LoadDatabaseConnectionParams, log hclog.Logger) (statements []string, args [][]driver.Value, err error) {
	statements = []string{}
	args = [][]driver.Value{}

	// Create the connection
	parsedURL, err := url.Parse(params.ConnectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse the connection string of %s: %w", params.SchemaName, err)
	}

	hosts := strings.Split(parsedURL.Host, ",")
	if len(hosts) == 0 || (len(hosts) == 1 && hosts[0] == "") {
		return nil, nil, fmt.Errorf("no hosts found in the connection string of %s", params.SchemaName)
	}

	for i, host := range hosts {
		hosts[i] = strings.TrimSpace(host)
	}

	// Create the cluster
	cluster := gocql.NewCluster(hosts...)
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		password, _ := parsedURL.User.Password()
		if username != "" || password != "" {
			cluster.Authenticator = gocql.PasswordAuthenticator{
				Username: username,
				Password: password,
			}
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, nil, fmt.Errorf("could not create the Cassandra session for %s: %w", params.SchemaName, err)
	}
	defer session.Close()

	// Get the list of tables
	query := "SELECT keyspace_name, table_name FROM system_schema.tables"
	iter := session.Query(query).Iter()
	tables := []sqlTable{}
	var keyspaceName, tableName string
	for iter.Scan(&keyspaceName, &tableName) {
		table := sqlTable{
			Schema:    keyspaceName,
			TableName: tableName,
			TableType: "TABLE", // Cassandra does not have a table type, so we set it to TABLE
		}
		tables = append(tables, table)
	}
	if err := iter.Close(); err != nil {
		return nil, nil, fmt.Errorf("could not get the list of tables for %s: %w", params.SchemaName, err)
	}

	// Filter the tables
	filteredTables, err := filterTables(tables, params.Filter, log)
	if err != nil {
		return nil, nil, fmt.Errorf("could not filter the tables for %s: %w", params.SchemaName, err)
	}

	// Create the statements
	// First, we attach a schema to the connection. This is just an in-memory database
	statements = append(statements, "ATTACH DATABASE ? AS ?")
	args = append(args, []driver.Value{fmt.Sprintf("file:%s?mode=memory&cache=shared", params.SchemaName), params.SchemaName})
	for _, table := range filteredTables {
		// Compute the table name
		tableName := strings.Builder{}
		tableName.WriteString(params.SchemaName)
		tableName.WriteString(".")
		tableName.WriteString(fmt.Sprintf("%s_", table.Schema))
		tableName.WriteString(table.TableName)
		// The table name remote side
		cassandraTableName := strings.Builder{}
		cassandraTableName.WriteString(table.Schema)
		cassandraTableName.WriteString(".")
		cassandraTableName.WriteString(table.TableName)

		// Create the virtual table and its mapping
		statements = append(statements, fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s USING cassandra_reader(dsn='%s', table='%s')", tableName.String(), params.ConnectionString, cassandraTableName.String()))
		args = append(args, []driver.Value{})
	}

	return
}
