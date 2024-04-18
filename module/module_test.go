package module

import (
	"database/sql"
	"os"
	"os/exec"
	"testing"

	"github.com/julien040/anyquery/rpc"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
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
			expected: "CREATE TABLE x(id INTEGER, name TEXT HIDDEN);",
			testName: "No primary key, one column is a parameter",
		},
		{
			schema:   schema2,
			expected: "CREATE TABLE x(id INTEGER PRIMARY KEY, name TEXT) WITHOUT ROWID;",
			testName: "With a primary key",
		},
		{
			schema:   schema3,
			expected: "CREATE TABLE x(id INTEGER HIDDEN, name TEXT, size REAL, binary BLOB);",
			testName: "Multiple columns, one is a parameter",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if got := createSQLiteSchema(tt.schema); got != tt.expected {
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
				PluginPath: "./_test/test.out",
			})
		},
	})

	// Open a connection
	db, err := sql.Open("sqlite_custom", ":memory:")
	assert.NoError(t, err, "Can't open the database")

	defer db.Close()

	t.Run("A query without required parameters should fail", func(t *testing.T) {
		_, err = db.Query("SELECT * FROM test")
		// Because we don't have parameters constraints, it should fail
		assert.Error(t, err, "A query without required parameters should fail")
	})

	t.Run("A query with required parameters should work", func(t *testing.T) {

		// We run a true query
		rows, err := db.Query("SELECT id, name, size, is_active FROM test('Franck')")
		assert.NoError(t, err, "A query with required parameters should work")

		_, err = rows.Columns()
		assert.NoError(t, err, "Columns should be retrieved")
		i := 0
		for rows.Next() {
			i++
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&id, &name, &size, &isActive)
			assert.NoError(t, err, "A scan should work")
			assert.Greater(t, id, int64(0), "The id should be greater than 0")
			if name.Valid {
				assert.NotEmpty(t, name, "The name should not be empty")
			}
		}
		assert.Equal(t, 20, i, "The number of rows should be 20")

		err = rows.Close()
		assert.NoError(t, err, "Rows should be closed")
	})
	t.Run("A query where constraints are removed by SQLite", func(t *testing.T) {
		// We run a true query
		rows, err := db.Query("SELECT id, name, size, is_active FROM test('Franck') WHERE (size IS NULL OR id IS NOT NULL) OR size IS NOT NULL")
		assert.NoError(t, err, "A query with required parameters should work")

		_, err = rows.Columns()
		assert.NoError(t, err, "Columns should be retrieved")
		i := 0
		for rows.Next() {
			i++
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&id, &name, &size, &isActive)
			assert.NoError(t, err, "A scan should work")
			assert.Greater(t, id, int64(0), "The id should be greater than 0")
			if name.Valid {
				assert.NotEmpty(t, name, "The name should not be empty")
			}
		}
		assert.Equal(t, 20, i, "The number of rows should be 20")

		err = rows.Close()
		assert.NoError(t, err, "Rows should be closed")
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
				PluginPath: "./_test/test2.out",
			})
		},
	})

	// Open a connection
	db, err := sql.Open("sqlite_custom2", ":memory:")
	assert.NoError(t, err, "Can't open the database")

	defer db.Close()

	t.Run("A query without primary key must have a rowid", func(t *testing.T) {
		rows, err := db.Query("SELECT rowid, * FROM test")
		assert.NoError(t, err, "A query without primary key must have a rowid")

		i := 0
		for rows.Next() {
			var rowid int64
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&rowid, &id, &name, &size, &isActive)
			assert.NoError(t, err, "A scan should work")
			assert.Greater(t, id, int64(0), "The id should be greater than 0")
			if name.Valid && i < 4 { // The name must be not null for the first 4 rows
				assert.NotEmpty(t, name, "The name should not be empty")
			}
			if i >= 4 { // We check that the fields are null
				assert.False(t, name.Valid, "The name should be null")
				assert.False(t, size.Valid, "The size should be null")
				assert.False(t, isActive.Valid, "The isActive should be null")
			}
			i++
		}
		rows.Close()

	})

	t.Run("A query with LIMIT and OFFSET", func(t *testing.T) {
		rows, err := db.Query("SELECT rowid, * FROM test LIMIT 3 OFFSET 2")
		assert.NoError(t, err, "A query without primary key must have a rowid")

		i := 0
		for rows.Next() {
			var rowid int64
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&rowid, &id, &name, &size, &isActive)
			assert.NoError(t, err, "A scan should work")
			assert.Greater(t, id, int64(0), "The id should be greater than 0")
			if i == 0 {
				assert.Equal(t, "Julien", name.String, "The name should be Julien")
			}
			i++
		}
		assert.Equal(t, 3, i, "The number of rows should be 3")
		rows.Close()

	})

	t.Run("A query with LIKE must work", func(t *testing.T) {
		rows, err := db.Query("SELECT rowid, * FROM test WHERE name LIKE '%n%'")
		assert.NoError(t, err, "A query without primary key must have a rowid")
		for rows.Next() {
			var rowid int64
			var id int64
			var name sql.NullString
			var size sql.NullFloat64
			var isActive sql.NullBool
			err = rows.Scan(&rowid, &id, &name, &size, &isActive)
			assert.NoError(t, err, "A scan should work")
			assert.Greater(t, id, int64(0), "The id should be greater than 0")
		}
		rows.Close()

	})

}

func TestOpCode(t *testing.T) {
	t.Parallel()
	// This test ensure that the go-sqlite3 keeps the same opcode

	assert.Equal(t, int(sqlite3.OpEQ), int(rpc.OperatorEqual), "The opcode EQ must be the same")
	assert.Equal(t, int(sqlite3.OpLT), int(rpc.OperatorLess), "The opcode LT must be the same")
	assert.Equal(t, int(sqlite3.OpLE), int(rpc.OperatorLessOrEqual), "The opcode LE must be the same")
	assert.Equal(t, int(sqlite3.OpGT), int(rpc.OperatorGreater), "The opcode GT must be the same")
	assert.Equal(t, int(sqlite3.OpGE), int(rpc.OperatorGreaterOrEqual), "The opcode GE must be the same")
	assert.Equal(t, int(sqlite3.OpLIKE), int(rpc.OperatorLike), "The opcode LIKE must be the same")
	assert.Equal(t, int(sqlite3.OpGLOB), int(rpc.OperatorGlob), "The opcode GLOB must be the same")
	assert.Equal(t, int(sqlite3.OpMATCH), int(rpc.OperatorMatch), "The opcode MATCH must be the same")
	assert.Equal(t, int(sqlite3.OpREGEXP), int(rpc.OperatorRegexp), "The opcode REGEXP must be the same")
	assert.Equal(t, int(sqlite3.OpLIMIT), int(rpc.OperatorLimit), "The opcode LIMIT must be the same")
	assert.Equal(t, int(sqlite3.OpOFFSET), int(rpc.OperatorOffset), "The opcode OFFSET must be the same")

}
