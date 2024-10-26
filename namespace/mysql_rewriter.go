package namespace

import (
	"fmt"
	"strings"

	"github.com/julien040/anyquery/other/sqlparser"
	"vitess.io/vitess/go/sqltypes"
)

// GetQueryType returns the type of the query
// (e.g. SELECT, INSERT, SHOW, SET, etc.)
// Internally, it uses the sqlparser from the vitess project
// to parse the query and return the type.
//
// If a syntax error is found, it returns an unknown type. Therefore, error will be nil.
func GetQueryType(query string) (sqlparser.StatementType, sqlparser.Statement, error) {
	parser, err := sqlparser.New(sqlparser.Options{
		MySQLServerVersion: "8.0.30",
	})

	if err != nil {
		return sqlparser.StmtUnknown, nil, err
	}

	stmt, err := parser.Parse(query)
	if err != nil {
		// We don't return any error here because an incompatible query with MySQL
		// might be compatible with SQLite
		//
		// If the query is not compatible with SQLite, the SQLite driver will return an error
		// so that's not an issue.
		return sqlparser.StmtUnknown, stmt, nil
	}

	// We let the sqlparser package determine the type of the query
	return sqlparser.ASTToStatementType(stmt), stmt, nil
}

// These queries are used to emulate MySQL specific queries

// showDatabasesQuery emulates the "SHOW DATABASES" query
//
// Name explanation:
// SQLite does not have a concept of databases as strong as MySQL (you can attach db though)
// Because some client specify the database to use as a prefix of the table name (e.g "SELECT * FROM mydb.mytable")
// and SQLite supports only two prefix (main and temp), we return only "main" as the databases available
const showDatabasesQuery = "SELECT name as Database FROM pragma_database_list()"

// showTablesQuery emulates the "SHOW TABLES" query
const showTablesQuery = `
SELECT
	name AS "Tables_in_main"
FROM
	pragma_table_list()
WHERE
	SCHEMA LIKE ?
	AND name LIKE ?
UNION
SELECT
	name AS Tables_in_main
FROM
	pragma_module_list()
WHERE name NOT LIKE 'fts%'
AND name NOT LIKE 'rtree%'
AND name NOT LIKE '%_reader'
AND name LIKE ?`

// showFullTablesQuery emulates the "SHOW FULL TABLES" query
const showFullTablesQuery = `
SELECT
	name AS "Tables_in_main",
	CASE TYPE
	WHEN 'view' THEN
		'VIEW'
	ELSE
		'BASE TABLE'
	END AS "Table_type"
FROM
	pragma_table_list()
WHERE 
	schema LIKE ? 
	AND name LIKE ?
UNION
SELECT
	name AS Tables_in_main,
	'BASE TABLE' AS Table_type
FROM
	pragma_module_list()
WHERE name NOT LIKE 'fts%'
AND name NOT LIKE 'rtree%'
AND name NOT LIKE '%_reader'
AND name LIKE ?`

// showColumnsQuery emulates the "SHOW COLUMNS" query
// Must be used in a prepared statement with the table name as a parameter
const showColumnsQuery = `
SELECT
		name AS Field,
	CASE TYPE
	WHEN 'INT' THEN
		'int'
	WHEN 'TEXT' THEN
		'varchar(65535)'
	WHEN 'BLOB' THEN
		'blob'
	WHEN 'NULL' THEN
		'null'
	WHEN 'REAL' THEN
		'float'
	ELSE
		lower(TYPE)
	END AS "Type",
	CASE WHEN (a."notnull" = 1
		OR a. "pk" = 1) THEN
		'NO'
	ELSE
		'YES'
	END AS "Null",
	CASE pk
	WHEN 1 THEN
		'PRI'
	ELSE
		''
	END AS "Key",
	NULL AS "Default",
	'' AS Extra
FROM
	pragma_table_info (?) a
WHERE name LIKE ?`

const showSessionQuery = `
SELECT
	column1 AS Variable_name,
	column2 AS Value
FROM (
	VALUES('Aborted_clients', '24'),
		('Aborted_connects',
			'90'),
		('Uptime',
			'994376'),
		('Ssl_cipher',
			'TLS_AES_256_GCM_SHA384'),
		('Uptime_since_flush_status',
			'994376'))
WHERE
	Variable_name LIKE ?`

const showTableStatusQuery = `
SELECT
    tl.name AS name,
    'SQLite' as Engine,
    10 AS Version,
    iif(tl.type = 'BASE TABLE', 'Dynamic', NULL) AS Row_format,
    0 AS Rows,
    0 AS Avg_row_length,
    0 AS Data_length,
    0 AS Max_data_length,
    0 AS Index_length,
    0 AS Data_free,
    NULL AS Auto_increment,
    '1970-01-01 00:00:00' AS Create_time,
    '1970-01-01 00:00:00' AS Update_time,
    NULL AS Check_time,
    'BINARY' AS Collation,
    NULL AS Checksum,
    '' AS Create_options,
    iif(tl.type = 'VIEW', 'VIEW', '') AS Comment
  FROM
    pragma_table_list tl
  WHERE tl.schema = ?
  AND Name LIKE ?;`

const showIndexesQuery = `
WITH x AS (
	SELECT
		name
	FROM
		pragma_table_list ()
	UNION
	SELECT
		name
	FROM
		pragma_module_list ()
	WHERE
		name NOT LIKE 'fts%%'
		AND name NOT LIKE 'rtree%%'
)
SELECT
	x.name as "Table",
	0 AS Non_unique,
	iif(pk = 1, 'Primary', '') AS Key_name,
	0 AS Seq_in_index,
	pt.name AS Column_name,
	'' AS Collation,
	0 AS Cardinality,
	NULL AS Sub_part,
	NULL AS Packed,
	'' AS "Null",
	'BTREE' AS Index_type,
	'' AS Comment,
	'' AS Index_comment,
	'YES' AS Visible,
	NULL AS Expression
FROM
	x, pragma_table_info (x.name) pt
WHERE pk = 1 AND "Table" = ? AND (%s);
`

const showCreateTableQuery = `
SELECT 
	? as "Table", 
	sql as "Create Table",
	'utf8mb4' as "character_set_client",
	'BINARY' as "collation_connection"
FROM sqlite_master 
WHERE name = ?`

const showCharacterSetsQuery = `
SELECT
	'utf8mb4' AS Charset,
	'UTF-8 Unicode' AS Description,
	'BINARY' AS Default_collation,
	4 AS Maxlen
FROM
	dual
WHERE Charset LIKE ?`

const showEnginesQuery = `
SELECT
	'SQLite' AS Engine,
	'DEFAULT' AS Support,
	'Thank you for using anyquery' AS Comment,
	'YES' AS Transactions,
	'YES' AS XA,
	'YES' AS Savepoints;
`

const showCollationsQuery = `
SELECT
		column1 AS Collation,
		column2 AS Charset,
		column3 AS Id,
		column4 AS "Default",
		column5 AS Compiled,
		column6 AS Sortlen,
		column7 AS Pad_attribute
	FROM (
	VALUES('RTRIM', 'utf8mb4', 0, '', 'YES', 1, 'PAD SPACE'),
		('NOCASE',
		'utf8mb4',
		1,
		'',
		'YES',
		1,
		'NO PAD'),
	('BINARY',
	'utf8mb4',
	2,
	'YES',
	'YES',
	1,
	'NO PAD')
	);`

const showCreateDatabaseQuery = `
SELECT
	? as "Database",
	'CREATE DATABASE "' || ? || '" COLLATE utf8mb4_general_ci' as "Create Database"`

const showWarningsQuery = `
SELECT
	'Note' AS Level,
	0 AS Code,
	'' AS Message
FROM
	dual
WHERE FALSE`

// emptyResultSet is an empty result set
const showEmptyResultSet = `
SELECT
	'' AS "Empty"
FROM
	dual
WHERE FALSE`

// Run a query on the database
// but rewrite, or provide special handling for MySQL specific queries
func (h *handler) runQueryWithMySQLSpecific(connectionID uint32, query string, args ...interface{}) (*sqltypes.Result, error) {

	// Find the type of the query and parse it
	queryType, parsedQuery, err := GetQueryType(query)
	if err != nil {
		return emptyResultSet, err
	}

	// Handle the query based on its type
	switch queryType {
	case sqlparser.StmtShow:
		query, args := RewriteShowStatement(parsedQuery.(*sqlparser.Show))
		return h.runSimpleQuery(connectionID, query, args...)
	case sqlparser.StmtUse:
		return emptyResultSet, nil
	case sqlparser.StmtSet:
		// SQLite does not support the "SET" command
		// so we return an empty result
		return emptyResultSet, nil
	// To catch DESCRIBE and EXPLAIN statements
	case sqlparser.StmtExplain:
		val, ok := parsedQuery.(*sqlparser.ExplainTab)
		if !ok {
			h.Logger.Warnf("Unexpected type for EXPLAIN statement: %T", parsedQuery)
			return emptyResultSet, nil
		}
		return h.runSimpleQuery(connectionID, showColumnsQuery, val.Table.Name.String(), "%")

	case sqlparser.StmtSelect:
		// We rewrite the query to be SQLite compatible
		rewriteSelectStatement(&parsedQuery)
		return h.runSimpleQuery(connectionID, sqlparser.String(parsedQuery), args...)
	case sqlparser.StmtDDL:
		// We run the DDL statement as is without any modification
		// For example, create index will be rewritten to alter table
		// and we don't want that. So we run the query as is
		return h.runSimpleQuery(connectionID, query, args...)

	case sqlparser.StmtUnknown:
		// If the query is not recognized (e.g. syntax error), we run it as is
		return h.runSimpleQuery(connectionID, query, args...)

	// However, for all the other cases, we run the parsed query
	// For instance, it helps transforming START TRANSACTION into BEGIN
	default:
		return h.runSimpleQuery(connectionID, sqlparser.String(parsedQuery), args...)
	}

}

// Take a SHOW statement and return the corresponding SQLite query
func RewriteShowStatement(parsedQuery *sqlparser.Show) (string, []interface{}) {
	// Find the like clause in the SHOW statement
	// If there is no like clause, we return a wildcard
	findLike := func(showStmt *sqlparser.ShowBasic) string {
		like := "%"
		if showStmt.Filter != nil && showStmt.Filter.Like != "" {
			like = showStmt.Filter.Like
		}
		return like
	}

	// For each type of SHOW statement, we return the corresponding result
	switch showType := parsedQuery.Internal.(type) {
	case *sqlparser.ShowBasic:
		switch showType.Command {
		case sqlparser.Table:
			// By default, we show all the tables
			like := findLike(showType)
			// By default, we show the tables from the main database
			dbName := "main"
			if showType.DbName.String() != "" {
				dbName = showType.DbName.String()
			}
			// Because SHOW FULL TABLES returns a different result than SHOW TABLES
			// we need to use a different query
			if showType.Full {
				return showFullTablesQuery, []interface{}{dbName, like, like}
			} else {
				return showTablesQuery, []interface{}{dbName, like, like}
			}
		// SHOW SESSION STATUS and SHOW GLOBAL STATUS
		case sqlparser.StatusGlobal, sqlparser.StatusSession:
			like := findLike(showType)
			return showSessionQuery, []interface{}{like}
		// SHOW VARIABLES, SHOW GLOBAL VARIABLES, SHOW SESSION VARIABLES
		case sqlparser.VariableGlobal, sqlparser.VariableSession:
			// We build a custom select query from the map of variables
			like := findLike(showType)
			query := strings.Builder{}
			query.WriteString("SELECT column1 AS Variable_name, column2 AS Value FROM (VALUES")
			i := 0
			for key, value := range selectVariableRemapper {
				query.WriteString(fmt.Sprintf("('%s', ", key))
				switch value.(type) {
				case string:
					query.WriteString(fmt.Sprintf("'%s')", value))
				case int:
					query.WriteString(fmt.Sprintf("%d)", value))
				default:
					query.WriteString(")")
				}
				// To not add a comma at the end of the last element
				if i < len(selectVariableRemapper)-1 {
					query.WriteString(", ")
				}
				i++
			}
			query.WriteString(") WHERE Variable_name LIKE ?")
			return query.String(), []interface{}{like}

		// SHOW DATABASES, SHOW SCHEMAS
		case sqlparser.Database:
			return showDatabasesQuery, nil

		// SHOW ENGINES
		case sqlparser.Engines:
			return showEnginesQuery, nil

		// SHOW COLLATION
		case sqlparser.Collation:
			like := findLike(showType)
			return showCollationsQuery, []interface{}{like}

		// SHOW CHARACTER SET
		case sqlparser.Charset:
			like := findLike(showType)
			return showCharacterSetsQuery, []interface{}{like}

		// SHOW TABLE STATUS
		case sqlparser.TableStatus:
			like := findLike(showType)
			dbName := "main"
			if showType.DbName.String() != "" {
				dbName = showType.DbName.String()
			}
			return showTableStatusQuery, []interface{}{dbName, like}

		// SHOW INDEXES
		case sqlparser.Index:
			filter := ""
			if showType.Filter != nil {
				filter = sqlparser.String(showType.Filter.Filter)
			}
			// We need to replace the %s in the query by the filter
			if filter == "" {
				filter = "1=1"
			} else {
				filter = strings.TrimPrefix(filter, " where ")
			}
			table := showType.Tbl.Name.String()
			fmt.Println("Filter", filter)
			return fmt.Sprintf(showIndexesQuery, filter), []interface{}{table}

		// SHOW COLUMNS, SHOW FIELDS
		case sqlparser.Column:
			like := findLike(showType)
			return showColumnsQuery, []interface{}{showType.Tbl.Name.String(), like}

		// SHOW WARNINGS
		case sqlparser.Warnings:
			// Return an empty table but with the correct fields
			return showWarningsQuery, nil

		// SHOW CREATE TABLE
		case sqlparser.CreateTbl:
			tblName := showType.Tbl.Name.String()
			return showCreateTableQuery, []interface{}{tblName, tblName}

		// SHOW CREATE VIEW
		case sqlparser.CreateV:
			vName := showType.Tbl.Name.String()
			return showCreateTableQuery, []interface{}{vName, vName}

		default:
			// Because it's a show statement we don't handle, we return an empty result
			return showEmptyResultSet, nil
		}
	case *sqlparser.ShowCreate:
		// We only handle the SHOW CREATE TABLE statement
		switch showType.Command {
		case sqlparser.CreateTbl, sqlparser.CreateV:
			tableName := showType.Op.Name.String()
			return showCreateTableQuery, []interface{}{tableName, tableName}
		case sqlparser.CreateDb:
			dbName := showType.Op.Name.String()
			return showCreateDatabaseQuery, []interface{}{dbName, dbName}
		default:
			return showEmptyResultSet, nil
		}
	default:
		// Because it's a show statement we don't handle, we return an empty result
		return showEmptyResultSet, nil
	}
}

// A map of variables that can be replaced by their default litteral value
// in a rewritten query
//
// For example @@session.auto_increment_increment -> 1
//
// Those values are directly extracted from a default MySQL database on DigitalOcean
// or from the MySQL documentation
var selectVariableRemapper = map[string]interface{}{
	// We don't add @@session because sqlparse strips it away
	"auto_increment_increment":     1,
	"character_set_client":         "utf8mb4",
	"character_set_connection":     "utf8mb4",
	"character_set_results":        "utf8mb4",
	"character_set_server":         "utf8mb4",
	"collation_connection":         "utf8mb4_general_ci",
	"init_connect":                 "",
	"interactive_timeout":          28800,
	"license":                      "GPL", // Technically, anyquery is not GPL but we don't care
	"lower_case_table_names":       2,     // SQLite is case-insensitive
	"max_allowed_packet":           67108864,
	"net_write_timeout":            60,
	"performance_schema":           "OFF",
	"sql_mode":                     "IGNORE_SPACE,ERROR_FOR_DIVISION_BY_ZERO,ONLY_FULL_GROUP_BY",
	"system_time_zone":             "UTC",
	"time_zone":                    "SYSTEM",
	"wait_timeout":                 28800,
	"query_cache_type":             "OFF",
	"query_cache_size":             0,
	"query_cache_limit":            1048576,
	"tx_isolation":                 "REPEATABLE-READ",
	"tx_read_only":                 0,
	"event_scheduler":              "OFF",
	"hostname":                     "127.0.0.1",
	"warning_count":                0,
	"version_comment":              "MySQL Community Server - GPL",
	"version_compile_machine":      "x86_64",
	"version":                      "8.0.30",
	"offline_mode":                 0,
	"transaction_read_only":        0,
	"transaction_isolation":        "REPEATABLE-READ",
	"transaction_allow_batching":   0,
	"transaction_prealloc_size":    4096,
	"transaction_alloc_block_size": 8192,
	"transaction_isolation_level":  "REPEATABLE-READ",
	"autocommit":                   1,
}

// Replace the function by their default value
var selectFunctionRemapper = map[string]interface{}{
	"database":       "main",
	"user":           "root",
	"system_user":    "root",
	"version":        "8.0.30",
	"schema":         "main",
	"connection_id":  0,
	"last_insert_id": 0,
	"current_user":   "root",
}

// Replace the function by their SQLite equivalent
var selectFunctionSQLiteRemapper = map[string]string{
	"left":      "ltrim",
	"right":     "rtrim",
	"if":        "iif",
	"now":       "datetime",
	"ucase":     "upper",
	"lcase":     "lower",
	"locate":    "instr",
	"position":  "instr",
	"substring": "substr",
	"least":     "min",
	"greatest":  "max",
}

var selectFunctionDeleteRemapper = map[string]interface{}{}

// Rewrite the SELECT statement to be SQLite compatible
//
// # Non exhaustive list of things rewritten:
//
//   - Collations (e.g. "utf8mb4_general_ci" -> "BINARY")
//   - SELECT @@myvar -> SELECT 'default value of myvar'
//   - SELECT database(), user(), system_user() -> SELECT 'main', 'root', 'root'
func rewriteSelectStatement(parsedQuery *sqlparser.Statement) {
	// We need to set the func to post because we need to traverse the leaf nodes
	sqlparser.Rewrite(*parsedQuery, nil, func(cursor *sqlparser.Cursor) bool {
		switch node := cursor.Node().(type) {
		case *sqlparser.CollateExpr:
			currentColation := node.Collation
			// If the collation is not one from SQLite, we replace it by the BINARY collation
			if currentColation != "RTRIM" && currentColation != "NOCASE" && currentColation != "BINARY" {
				cursor.Replace(&sqlparser.CollateExpr{
					Expr:      node.Expr,
					Collation: "BINARY",
				})
			}
		// Some functions have their own specific handling
		case *sqlparser.TrimFuncExpr:
			// To do

		case *sqlparser.ConvertExpr:
			// Rewrite it as a cast
			cursor.Replace(&sqlparser.CastExpr{
				Expr: node.Expr,
				Type: node.Type,
			})

		case *sqlparser.LocateExpr:
			// We replace the locate function by instr
			cursor.Replace(&sqlparser.FuncExpr{
				Name: sqlparser.NewIdentifierCI("instr"),
				Exprs: sqlparser.Exprs(
					[]sqlparser.Expr{
						node.Str,
						node.SubStr,
					}),
			})

		case *sqlparser.FuncExpr:
			// If the function is in the list of functions to delete, we replace it by NULL
			if _, ok := selectFunctionDeleteRemapper[strings.ToLower(node.Name.String())]; ok {
				cursor.Replace(&sqlparser.NullVal{})
				return true
			}
			// We rewrite the function to its default value or if the function is available in SQLite,
			// we just rename it
			if val, ok := selectFunctionSQLiteRemapper[strings.ToLower(node.Name.String())]; ok {
				cursor.Replace(&sqlparser.FuncExpr{
					Name:      sqlparser.NewIdentifierCI(val),
					Qualifier: node.Qualifier,
					Exprs:     node.Exprs,
				})
				return true // To avoid replacing the function again
			}

			var literalValue interface{}
			// We don't use sqlparser.String(expr) because it quotes the function name for safety
			if val, ok := selectFunctionRemapper[strings.ToLower(node.Name.String())]; ok {
				switch val.(type) {
				case string:
					literalValue = sqlparser.NewStrLiteral(val.(string))
				case int:
					literalValue = sqlparser.NewIntLiteral(fmt.Sprint(val.(int)))
				default:
					literalValue = sqlparser.NewStrLiteral("")
				}

				cursor.Replace(literalValue.(*sqlparser.Literal))
			}
			// If we don't know the function, we don't rewrite it
			// and we let SQLite handle it
		case *sqlparser.AliasedExpr:
			switch expr := node.Expr.(type) {
			case *sqlparser.Variable:
				// We rewrite the variable to its default value
				// or an empty string if we don't know the default value
				var literalValue interface{}
				if val, ok := selectVariableRemapper[strings.ToLower(sqlparser.String(expr.Name))]; ok {
					switch val.(type) {
					case string:
						literalValue = sqlparser.NewStrLiteral(val.(string))
					case int:
						literalValue = sqlparser.NewIntLiteral(fmt.Sprint(val.(int)))
					default:
						literalValue = sqlparser.NewStrLiteral("")
					}
				} else {
					literalValue = sqlparser.NewStrLiteral("")
				}
				cursor.Replace(&sqlparser.AliasedExpr{
					As:   node.As,
					Expr: literalValue.(*sqlparser.Literal),
				})
			}

		}
		// We continue the traversal
		return true
	})

}
