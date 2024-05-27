package namespace

import (
	"database/sql"
	"io"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"
)

func setupTestNamespaceDB(t *testing.T) (*Namespace, *sql.DB) {
	// Open a connection to SQLite
	namespace, err := NewNamespace(NamespaceConfig{
		InMemory: true,
	})
	require.NoError(t, err, "creating a namespace should not return an error")

	db, err := namespace.Register("test_db")
	require.NoError(t, err, "registering a database should not return an error")

	return namespace, db
}

func startNamespaceServer(t *testing.T) *MySQLServer {
	_, db := setupTestNamespaceDB(t)

	logger := log.Default()
	logger.SetOutput(io.Discard)
	logger.SetLevel(log.DebugLevel)
	if testing.Verbose() {
		logger = log.New(os.Stderr)
	}

	server := MySQLServer{
		DB:                     db,
		MustCatchMySQLSpecific: true,
		Address:                "127.0.0.1:8008", // We hope this one is free
		Logger:                 logger,
	}

	go func() {
		logger.Info("Starting MySQL server")
		err := server.Start()
		require.NoError(t, err, "starting the server should not return an error")
		db.Close()
	}()

	return &server
}

func TestMySQLServer(t *testing.T) {
	server := startNamespaceServer(t)
	defer server.Stop()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Connect to the server
	db, err := sqlx.Open("mysql", "testuser:aa@tcp(127.0.0.1:8008)/test_db")
	require.NoError(t, err, "connecting to the server should not return an error")

	t.Run("Create a table", func(t *testing.T) {
		_, err := db.Exec("CREATE TABLE test_table (id INT PRIMARY KEY, name varchar(255), data FLOAT, blob BLOB)")
		require.NoError(t, err, "creating a table should not return an error")
	})

	t.Run("Insert a few rows", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO test_table (id, name, data, blob) VALUES (1, 'test', 3.14, x'010203'),
			(2, 'test2', 3.15, x'040506'), (3, 'test3', 3.16, x'070809')`)
		require.NoError(t, err, "inserting rows should not return an error")
	})

	type testTable struct {
		ID   int
		Name string
		Data float64
		Blob []byte
	}

	t.Run("Select a row", func(t *testing.T) {
		var result []testTable
		expected := []testTable{
			{ID: 1, Name: "test", Data: 3.14, Blob: []byte{1, 2, 3}},
			{ID: 2, Name: "test2", Data: 3.15, Blob: []byte{4, 5, 6}},
			{ID: 3, Name: "test3", Data: 3.16, Blob: []byte{7, 8, 9}},
		}

		err := db.Select(&result, "SELECT * FROM test_table")
		require.NoError(t, err, "selecting a row should not return an error")

		require.Equal(t, expected, result, "the selected row should match the expected row")
	})

	t.Run("Run a prepared statement", func(t *testing.T) {
		stmt, err := db.Preparex("SELECT * FROM test_table WHERE id = ?")
		require.NoError(t, err, "preparing a statement should not return an error")

		var result testTable
		expected := testTable{ID: 2, Name: "test2", Data: 3.15, Blob: []byte{4, 5, 6}}

		err = stmt.Get(&result, 2)
		require.NoError(t, err, "running a prepared statement should not return an error")

		require.Equal(t, expected, result, "the selected row should match the expected row")
	})

	t.Run("Request from dual", func(t *testing.T) {
		var result int
		err := db.Get(&result, "SELECT 1 FROM dual")
		require.NoError(t, err, "selecting from dual should not return an error")
	})

	t.Run("Show statements are working", func(t *testing.T) {
		t.Run("SHOW TABLES", func(t *testing.T) {
			var tables []string
			err := db.Select(&tables, "SHOW TABLES")
			require.NoError(t, err, "showing tables should not return an error")

			require.Contains(t, tables, "test_table", "the table should be in the list of tables")
			require.Contains(t, tables, "sqlite_schema", "the sqlite schema table should be in the list of tables")
		})

		t.Run("SHOW DATABASES", func(t *testing.T) {
			var databases []string
			err := db.Select(&databases, "SHOW DATABASES")
			require.NoError(t, err, "showing databases should not return an error")

			require.Contains(t, databases, "main", "the main database should be in the list of databases")
			require.Contains(t, databases, "information_schema", "the information schema database should be in the list of databases")
			require.Contains(t, databases, "mysql", "the mysql database should be in the list of databases")

		})

		t.Run("SHOW COLUMNS", func(t *testing.T) {
			var columns []struct {
				Field   string         `db:"Field"`
				Type    string         `db:"Type"`
				Null    string         `db:"Null"`
				Key     string         `db:"Key"`
				Default sql.NullString `db:"Default"`
				Extra   sql.NullString `db:"Extra"`
			}
			err := db.Select(&columns, "SHOW COLUMNS FROM test_table")
			require.NoError(t, err, "showing columns should not return an error")

			require.Len(t, columns, 4, "there should be 4 columns in the table")

			for _, column := range columns {
				switch column.Field {
				case "id":
					require.Equal(t, "int", column.Type, "the id column should be of type int")
					require.Equal(t, "NO", column.Null, "the id column should not be nullable")
					require.Equal(t, "PRI", column.Key, "the id column should be a primary key")
					require.Equal(t, "", column.Default.String, "the id column should not have a default value")
				case "name":
					require.Equal(t, "varchar(255)", column.Type, "the name column should be of type varchar(255)")
					require.Equal(t, "YES", column.Null, "the name column should be nullable")
					require.Equal(t, "", column.Key, "the name column should not be a primary key")
					require.Equal(t, "", column.Default.String, "the name column should not have a default value")
				case "data":
					require.Equal(t, "float", column.Type, "the data column should be of type float")
					require.Equal(t, "YES", column.Null, "the data column should be nullable")
					require.Equal(t, "", column.Key, "the data column should not be a primary key")
					require.Equal(t, "", column.Default.String, "the data column should not have a default value")
				case "blob":
					require.Equal(t, "blob", column.Type, "the blob column should be of type blob")
					require.Equal(t, "YES", column.Null, "the blob column should be nullable")
					require.Equal(t, "", column.Key, "the blob column should not be a primary key")
					require.Equal(t, "", column.Default.String, "the blob column should not have a default value")
				default:
					t.Errorf("unexpected column: %s", column.Field)
				}
			}

		})

		t.Run("SHOW CREATE TABLE", func(t *testing.T) {
			createTable := struct {
				Table       string `db:"Table"`
				CreateTable string `db:"Create Table"`
				Charset     string `db:"character_set_client"`
				Collation   string `db:"collation_connection"`
			}{}
			err := db.Get(&createTable, "SHOW CREATE TABLE test_table")
			require.NoError(t, err, "showing create table should not return an error")

			require.Equal(t, "test_table", createTable.Table, "the table name should be test_table")
			require.Contains(t, createTable.CreateTable, "CREATE TABLE test_table", "the create table statement should contain the table name")
			require.Equal(t, "utf8mb4", createTable.Charset, "the charset should be utf8mb4")
			require.Equal(t, "BINARY", createTable.Collation, "the collation should be BINARY")
		})
	})

	// Information schema tests
	t.Run("Information schema tests", func(t *testing.T) {
		t.Run("Select from information schema tables", func(t *testing.T) {
			var result []struct {
				TableSchema string `db:"TABLE_SCHEMA"`
				TableName   string `db:"TABLE_NAME"`
				TableType   string `db:"TABLE_TYPE"`
			}
			err := db.Select(&result, "SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES")
			require.NoError(t, err, "selecting from information schema should not return an error")

			require.Greater(t, len(result), 3, "there should be more than 3 tables in the information schema")

			for _, row := range result {
				require.NotEmpty(t, row.TableSchema, "the table schema should not be empty")
				require.NotEmpty(t, row.TableName, "the table name should not be empty")
				require.NotEmpty(t, row.TableType, "the table type should not be empty")
				// Check if BASE TABLE or VIEW
				require.Contains(t, []string{"BASE TABLE", "VIEW", "SYSTEM VIEW"}, row.TableType, "the table type should be BASE TABLE or VIEW")
			}

		})

		t.Run("Select from information schema columns", func(t *testing.T) {
			var result []struct {
				TableSchema string `db:"TABLE_SCHEMA"`
				TableName   string `db:"TABLE_NAME"`
				ColumnName  string `db:"COLUMN_NAME"`
			}
			err := db.Select(&result, "SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME FROM information_schema.COLUMNS")
			require.NoError(t, err, "selecting from information schema should not return an error")
			// test_table has 4 columns, sqlite_schema has 5 columns, dual has 1 column
			require.LessOrEqual(t, 10, len(result), "there should be more than 10 columns in the information schema")
		})

	})

	// Drop the table
	t.Run("Drop the table", func(t *testing.T) {
		_, err := db.Exec("DROP TABLE test_table")
		require.NoError(t, err, "dropping the table should not return an error")
	})

	// Close the connection
	err = db.Close()
	require.NoError(t, err, "closing the connection should not return an error")

}
