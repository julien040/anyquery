package module

import (
	"database/sql"
	"os"
	"os/exec"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/jmoiron/sqlx"
	"github.com/julien040/anyquery/rpc"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

var schema1 = rpc.DatabaseSchema{
	PrimaryKey: -1,
	Columns: []rpc.DatabaseSchemaColumn{
		{
			Name:        "id",
			Type:        rpc.ColumnTypeInt,
			IsParameter: false,
		},
		{
			Name:        "name",
			Type:        rpc.ColumnTypeString,
			IsParameter: true,
		},
	},
}

var schema2 = rpc.DatabaseSchema{
	PrimaryKey: 0,
	Columns: []rpc.DatabaseSchemaColumn{
		{
			Name:        "id",
			Type:        rpc.ColumnTypeInt,
			IsParameter: false,
		},
		{
			Name:        "name",
			Type:        rpc.ColumnTypeString,
			IsParameter: false,
		},
	},
}

var schema3 = rpc.DatabaseSchema{
	PrimaryKey: -1,
	Columns: []rpc.DatabaseSchemaColumn{
		{
			Name:        "id",
			Type:        rpc.ColumnTypeInt,
			IsParameter: true,
		},
		{
			Name:        "name",
			Type:        rpc.ColumnTypeString,
			IsParameter: false,
		},
		{
			Name:        "size",
			Type:        rpc.ColumnTypeFloat,
			IsParameter: false,
		},
		{
			Name:        "binary",
			Type:        rpc.ColumnTypeBlob,
			IsParameter: false,
		},
	},
}

func TestCreateSQLiteSchema(t *testing.T) {
	t.Parallel()
	type args struct {
		schema   rpc.DatabaseSchema
		expected string
		testName string
	}

	tests := []args{
		{
			schema:   schema1,
			expected: `CREATE TABLE x("id" INTEGER, "name" TEXT HIDDEN);`,
			testName: "No primary key, one column is a parameter",
		},
		{
			schema:   schema2,
			expected: `CREATE TABLE x("id" INTEGER PRIMARY KEY, "name" TEXT) WITHOUT ROWID;`,
			testName: "With a primary key",
		},
		{
			schema:   schema3,
			expected: `CREATE TABLE x("id" INTEGER HIDDEN, "name" TEXT, "size" REAL, "binary" BLOB);`,
			testName: "Multiple columns, one is a parameter",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if got, err := createSQLiteSchema(tt.schema); got != tt.expected {
				if err != nil {
					t.Errorf("CreateSQLiteSchema() error = %v", err)
				}
				t.Errorf("CreateSQLiteSchema() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRawPlugin(t *testing.T) {
	t.Parallel()
	// Build the raw plugin
	os.Mkdir("_test", 0755)
	err := exec.Command("go", "build", "-o", "_test/test.out", "../test/rawplugin.go").Run()
	if err != nil {
		t.Fatalf("Can't build the plugin: %v", err)
	}

	// Register a db connection
	sql.Register("sqlite_custom", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.CreateModule("test", &SQLiteModule{
				PluginPath:     "./_test/test.out",
				ConnectionPool: rpc.NewConnectionPool(),
				Logger:         hclog.NewNullLogger(),
			})
		},
	})

	// Open a connection
	db, err := sql.Open("sqlite_custom", ":memory:")
	require.NoError(t, err, "Can't open the database")

	defer db.Close()

	t.Run("A query without required parameters should fail", func(t *testing.T) {
		_, err = db.Query("SELECT * FROM test")
		// Because we don't have parameters constraints, it should fail
		require.Error(t, err, "A query without required parameters should fail")
	})

	t.Run("A query with required parameters should work", func(t *testing.T) {

		// We run a true query
		rows, err := db.Query("SELECT id, name, size, is_active FROM test('Franck')")
		require.NoError(t, err, "A query with required parameters should work")

		_, err = rows.Columns()
		require.NoError(t, err, "Columns should be retrieved")
		i := 0
		for rows.Next() {
			i++
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&id, &name, &size, &isActive)
			require.NoError(t, err, "A scan should work")
			require.Greater(t, id, int64(0), "The id should be greater than 0")
			if name.Valid {
				require.NotEmpty(t, name, "The name should not be empty")
			}
		}
		require.Equal(t, 20, i, "The number of rows should be 20")

		err = rows.Close()
		require.NoError(t, err, "Rows should be closed")
	})
	t.Run("A query where constraints are removed by SQLite", func(t *testing.T) {
		// We run a true query
		rows, err := db.Query("SELECT id, name, size, is_active FROM test('Franck') WHERE (size IS NULL OR id IS NOT NULL) OR size IS NOT NULL")
		require.NoError(t, err, "A query with required parameters should work")

		_, err = rows.Columns()
		require.NoError(t, err, "Columns should be retrieved")
		i := 0
		for rows.Next() {
			i++
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&id, &name, &size, &isActive)
			require.NoError(t, err, "A scan should work")
			require.Greater(t, id, int64(0), "The id should be greater than 0")
			if name.Valid {
				require.NotEmpty(t, name, "The name should not be empty")
			}
		}
		require.Equal(t, 20, i, "The number of rows should be 20")

		err = rows.Close()
		require.NoError(t, err, "Rows should be closed")
	})

}

func TestRawPlugin2(t *testing.T) {
	t.Parallel()
	// Build the raw plugin
	os.Mkdir("_test", 0755)
	err := exec.Command("go", "build", "-o", "_test/test2.out", "../test/rawplugin2.go").Run()
	if err != nil {
		t.Fatalf("Can't build the plugin: %v", err)
	}

	// Register a db connection
	sql.Register("sqlite_custom2", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.CreateModule("test", &SQLiteModule{
				PluginPath:     "./_test/test2.out",
				ConnectionPool: rpc.NewConnectionPool(),
				Logger:         hclog.NewNullLogger(),
			})
		},
	})

	// Open a connection
	db, err := sql.Open("sqlite_custom2", ":memory:")
	require.NoError(t, err, "Can't open the database")

	defer db.Close()

	t.Run("A query without primary key must have a rowid", func(t *testing.T) {
		rows, err := db.Query("SELECT rowid, * FROM test")
		require.NoError(t, err, "A query without primary key must have a rowid")

		i := 0
		for rows.Next() {
			var rowid int64
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&rowid, &id, &name, &size, &isActive)
			require.NoError(t, err, "A scan should work")
			require.Greater(t, id, int64(0), "The id should be greater than 0")
			if name.Valid && i < 4 { // The name must be not null for the first 4 rows
				require.NotEmpty(t, name, "The name should not be empty")
			}
			if i >= 4 { // We check that the fields are null
				require.False(t, name.Valid, "The name should be null")
				require.False(t, size.Valid, "The size should be null")
				require.False(t, isActive.Valid, "The isActive should be null")
			}
			i++
		}
		rows.Close()

	})

	t.Run("A query with LIMIT and OFFSET", func(t *testing.T) {
		rows, err := db.Query("SELECT rowid, * FROM test LIMIT 3 OFFSET 2")
		require.NoError(t, err, "A query without primary key must have a rowid")

		i := 0
		for rows.Next() {
			var rowid int64
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&rowid, &id, &name, &size, &isActive)
			require.NoError(t, err, "A scan should work")
			require.Greater(t, id, int64(0), "The id should be greater than 0")
			if i == 0 {
				require.Equal(t, "Julien", name.String, "The name should be Julien")
			}
			i++
		}
		require.Equal(t, 3, i, "The number of rows should be 3")
		rows.Close()

	})

	t.Run("A query with LIKE must work", func(t *testing.T) {
		rows, err := db.Query("SELECT rowid, * FROM test WHERE name LIKE '%n%'")
		require.NoError(t, err, "A query without primary key must have a rowid")
		for rows.Next() {
			var rowid int64
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&rowid, &id, &name, &size, &isActive)
			require.NoError(t, err, "A scan should work")
			require.Greater(t, id, int64(0), "The id should be greater than 0")
		}
		rows.Close()

	})

}

// Test a plugin built with the lib plugin
// rather than the raw plugin
func TestLibPlugin(t *testing.T) {
	t.Parallel()
	// Build the raw plugin
	os.Mkdir("_test", 0755)
	err := exec.Command("go", "build", "-o", "_test/normalplugin.out", "../test/normalplugin.go").Run()
	if err != nil {
		t.Fatalf("Can't build the plugin: %v", err)
	}

	// Register a db connection
	sql.Register("sqlite_custom3", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.CreateModule("test", &SQLiteModule{
				PluginPath:     "./_test/normalplugin.out",
				ConnectionPool: rpc.NewConnectionPool(),
				Logger:         hclog.NewNullLogger(),
			})
		},
	})

	// Open a connection
	db, err := sql.Open("sqlite_custom3", ":memory:")
	require.NoError(t, err, "Can't open the database")

	// Run a simple query
	rows, err := db.Query("SELECT * FROM test")
	require.NoError(t, err, "A query must work")
	i := 0
	for rows.Next() {
		var id int64
		var name sql.NullString
		rows.Scan(&id, &name)

		require.Greater(t, id, int64(0), "The id should be greater than 0")
		if name.Valid {
			require.NotEmpty(t, name, "The name should not be empty")
		}
		i++
	}
	require.Equal(t, 2, i, "The number of rows should be 2")

	defer db.Close()

}

func TestOpCode(t *testing.T) {
	t.Parallel()
	// This test ensure that the go-sqlite3 keeps the same opcode

	require.Equal(t, int(sqlite3.OpEQ), int(rpc.OperatorEqual), "The opcode EQ must be the same")
	require.Equal(t, int(sqlite3.OpLT), int(rpc.OperatorLess), "The opcode LT must be the same")
	require.Equal(t, int(sqlite3.OpLE), int(rpc.OperatorLessOrEqual), "The opcode LE must be the same")
	require.Equal(t, int(sqlite3.OpGT), int(rpc.OperatorGreater), "The opcode GT must be the same")
	require.Equal(t, int(sqlite3.OpGE), int(rpc.OperatorGreaterOrEqual), "The opcode GE must be the same")
	require.Equal(t, int(sqlite3.OpLIKE), int(rpc.OperatorLike), "The opcode LIKE must be the same")
	require.Equal(t, int(sqlite3.OpGLOB), int(rpc.OperatorGlob), "The opcode GLOB must be the same")
	require.Equal(t, int(sqlite3.OpMATCH), int(rpc.OperatorMatch), "The opcode MATCH must be the same")
	require.Equal(t, int(sqlite3.OpREGEXP), int(rpc.OperatorRegexp), "The opcode REGEXP must be the same")
	require.Equal(t, int(sqlite3.OpLIMIT), int(rpc.OperatorLimit), "The opcode LIMIT must be the same")
	require.Equal(t, int(sqlite3.OpOFFSET), int(rpc.OperatorOffset), "The opcode OFFSET must be the same")

}

func TestXBestIndexConstraintsValidation(t *testing.T) {
	table := SQLiteTable{
		Schema: rpc.DatabaseSchema{
			PrimaryKey: -1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					IsParameter: false,
				},
			},
		},
	}
	t.Run("A query without the required parameters should fail", func(t *testing.T) {
		// The required parameter is missing
		constraints := []sqlite3.InfoConstraint{
			{
				Column: 1,
				Op:     sqlite3.OpEQ,
				Usable: true,
			},
		}

		ob := []sqlite3.InfoOrderBy{}
		_, err := table.BestIndex(constraints, ob, sqlite3.IndexInformation{})
		require.Error(t, err, "The query should fail because the required parameter is missing")
	})

	t.Run("A query with the required parameters but not usable should fail", func(t *testing.T) {
		// The required parameter is missing
		constraints := []sqlite3.InfoConstraint{
			{
				Column: 0,
				Op:     sqlite3.OpEQ,
				Usable: false,
			},
			{
				Column: 1,
				Op:     sqlite3.OpEQ,
				Usable: true,
			},
		}

		ob := []sqlite3.InfoOrderBy{}
		_, err := table.BestIndex(constraints, ob, sqlite3.IndexInformation{})
		require.Error(t, err, "The query should fail because the required parameter is not usable")
	})

	t.Run("A query with the required parameters should work", func(t *testing.T) {
		constraints := []sqlite3.InfoConstraint{
			{
				Column: 0,
				Op:     sqlite3.OpEQ,
				Usable: true,
			},
			{
				Column: 1,
				Op:     sqlite3.OpEQ,
				Usable: true,
			},
		}

		ob := []sqlite3.InfoOrderBy{}
		_, err := table.BestIndex(constraints, ob, sqlite3.IndexInformation{})
		require.NoError(t, err, "The query should work")
	})

	t.Run("A query with the required parameters should work even if other columns are not usable", func(t *testing.T) {
		constraints := []sqlite3.InfoConstraint{
			{
				Column: 0,
				Op:     sqlite3.OpEQ,
				Usable: true,
			},
			{
				Column: 1,
				Op:     sqlite3.OpEQ,
				Usable: false,
			},
		}

		ob := []sqlite3.InfoOrderBy{}
		_, err := table.BestIndex(constraints, ob, sqlite3.IndexInformation{})
		require.NoError(t, err, "The query should work")
	})

}

func TestCUDOperations(t *testing.T) {
	t.Parallel()
	// Build the raw plugin
	os.Mkdir("_test", 0755)
	err := exec.Command("go", "build", "-o", "_test/insertplugin.out", "../test/insertplugin.go").Run()
	if err != nil {
		t.Fatalf("Can't build the plugin: %v", err)
	}

	// Register a db connection
	sql.Register("sqlite_custom_insert", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.CreateModule("test_insert", &SQLiteModule{
				PluginPath:     "./_test/insertplugin.out",
				ConnectionPool: rpc.NewConnectionPool(),
				Logger:         hclog.NewNullLogger(),
			})
		},
	})

	// Open a connection
	db, err := sql.Open("sqlite_custom_insert", ":memory:")
	require.NoError(t, err, "Can't open the database")
	defer db.Close()

	dbx := sqlx.NewDb(db, "sqlite_custom_insert")

	t.Run("Insert a row", func(t *testing.T) {
		_, err = db.Exec("INSERT INTO test_insert(id, name, age, address) VALUES(6, 'Julien', 30, 'Paris')")
		require.NoError(t, err, "The insert should work")

		// We check that the row is inserted
		var name string
		err = dbx.Get(&name, "SELECT name FROM test_insert WHERE id=6")
		require.NoError(t, err, "The row should be inserted")
		require.Equal(t, "Julien", name, "The name should be Julien")
	})

	t.Run("Update a row", func(t *testing.T) {
		t.Log("Try to update a row")
		_, err = db.Exec("UPDATE test_insert SET name='Franck' WHERE id=1")
		require.NoError(t, err, "The update should work")

		// We check that the row is updated
		var name string
		err = dbx.Get(&name, "SELECT name FROM test_insert WHERE id=1")
		require.NoError(t, err, "The row should be updated")
		require.Equal(t, "Franck", name, "The name should be Franck")

		_, err = db.Exec("UPDATE test_insert SET name='Michel', id=12 WHERE id=1")
		require.NoError(t, err, "The update should work")

		// We check that the row is updated
		err = dbx.Get(&name, "SELECT name FROM test_insert WHERE id=12")
		require.NoError(t, err, "The row should be updated")
		require.Equal(t, "Michel", name, "The name should be Michel")
	})

	t.Run("Delete a row", func(t *testing.T) {
		t.Log("Try to delete a row")
		_, err = db.Exec("DELETE FROM test_insert WHERE id=2")
		require.NoError(t, err, "The delete should work")

		// We check that the row is deleted
		var name string
		err = dbx.Get(&name, "SELECT name FROM test_insert WHERE id=2")
		require.Error(t, err, "The row should be deleted")
	})
}
