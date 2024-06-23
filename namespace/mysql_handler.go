package namespace

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/mysql/collations"
	"vitess.io/vitess/go/mysql/replication"
	"vitess.io/vitess/go/sqltypes"

	querypb "vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/vtenv"

	log "github.com/charmbracelet/log"

	"golang.org/x/exp/maps"
)

// The listener from the mysql package takes a Handler interface
// It is defined in this file

// Represent a response to a MySQL query
// where no rows are returned (.e.g. INSERT, UPDATE, DELETE)
var emptyResultSet = &sqltypes.Result{
	Fields:       make([]*querypb.Field, 0),
	Rows:         make([]sqltypes.Row, 0),
	RowsAffected: 0,
	InsertID:     0,
	StatusFlags:  0,
}

type handler struct {
	env                 *vtenv.Environment
	DB                  *sql.DB
	RewriteMySQLQueries bool
	databaseInited      bool
	Logger              *log.Logger
	// Allow each MySQL connection to have its own SQLite connection
	connectionMapperSQLite map[uint32]*sql.Conn

	// A mutex to protect the connectionMapperSQLite map
	mutexConnectionMapperSQLite sync.Mutex

	// Track each MySQL connection in a slice
	//
	// You might wonder why we need to keep track of the MySQL connections
	// It's because when we close the MySQL server, the connections are not closed
	// Therefore, handler.ConnectionClosed is never called resulting in the SQLite connections not being closed
	// If we don't close the SQLite connections manually opened by db.Conn, the method db.Close will not call
	// the destructor of virtual plugins, resulting in not killing the plugin process, leaking processes
	//
	// To conclude, leaving MySQL connections open will result in zombie processes, butterfly effect in action
	// An afternoon was lost to find this bug
	connections []*mysql.Conn
}

func (h *handler) NewConnection(c *mysql.Conn) {
	h.mutexConnectionMapperSQLite.Lock()
	defer h.mutexConnectionMapperSQLite.Unlock()
	h.Logger.Info("New connection", "connectionID", c.ConnectionID, "username", c.User, "charset", c.CharacterSet)
	if h.RewriteMySQLQueries && !h.databaseInited {
		err := prepareDatabaseForMySQL(h.DB)
		if err != nil {
			h.Logger.Error("Error preparing database for MySQL compatibility. Some MySQL clients might not work as expected", "err", err)
		} else {
			h.databaseInited = true
		}
	}
	// We create a new connection for the MySQL connection
	// This is useful to have a separate connection for each MySQL connection
	// so that BEGIN and COMMIT can be used
	ctx := context.Background()
	conn, err := h.DB.Conn(ctx)
	h.Logger.Debug("Connection created", "connectionID", c.ConnectionID, "username", c.User)
	if err != nil {
		h.Logger.Error("Error creating connection", "err", err, "connectionID", c.ConnectionID, "username", c.User)
		return
	}

	// We store the connection in a map
	if h.connectionMapperSQLite == nil {
		h.connectionMapperSQLite = make(map[uint32]*sql.Conn)
	}

	h.connectionMapperSQLite[c.ConnectionID] = conn

	// We append the MySQL connection to the list of connections
	h.connections = append(h.connections, c)

}

func (h *handler) ConnectionClosed(c *mysql.Conn) {
	h.Logger.Info("Connection closed", "connectionID", c.ConnectionID, "username", c.User)

	// Close the connection associated with the MySQL connection
	if conn, ok := h.connectionMapperSQLite[c.ConnectionID]; ok {
		fmt.Println("Closing connection MySQL", c.ConnectionID)
		err := conn.Close()
		if err != nil {
			h.Logger.Error("Error closing connection", "err", err, "connectionID", c.ConnectionID, "username", c.User)
		}
		delete(h.connectionMapperSQLite, c.ConnectionID)
	} else {
		h.Logger.Error("SQLite connection not found", "connectionID", c.ConnectionID, "username", c.User)
	}

}

func (h *handler) ComPrepare(c *mysql.Conn, query string, bindVars map[string]*querypb.BindVariable) ([]*querypb.Field, error) {
	h.Logger.Debug("Prepare query", "query", query, "connectionID", c.ConnectionID, "username", c.User)
	return nil, nil
}

func (h *handler) ComStmtExecute(c *mysql.Conn, f *mysql.PrepareData, callback func(*sqltypes.Result) error) error {
	h.Logger.Debug("Execute prepared statement", "connectionID", c.ConnectionID, "username", c.User, "prepareStmt", f.PrepareStmt)

	// We create a slice of interfaces to pass to the Query method
	// They represent the arguments of the prepared statement
	values := make([]interface{}, f.ParamsCount)

	// We get a slice of the keys of the bindVars map
	// and sort them in alphabetical order
	keys := maps.Keys(f.BindVars)
	sort.Strings(keys)

	// We iterate over the keys and fill the values slice
	// Because values are stored as byte slices, we need to convert them to the correct type
	for i, key := range keys {
		varType := f.BindVars[key].Type
		switch varType {
		case querypb.Type_INT64:
			val, err := strconv.Atoi(string(f.BindVars[key].Value))
			if err != nil {
				return err
			}
			values[i] = val
		case querypb.Type_VARCHAR:
			values[i] = string(f.BindVars[key].Value)
		case querypb.Type_FLOAT64:
			val, err := strconv.ParseFloat(string(f.BindVars[key].Value), 64)
			if err != nil {
				return err
			}
			values[i] = val
		case querypb.Type_VARBINARY:
			values[i] = f.BindVars[key].Value
		default:
			values[i] = nil
		}

	}
	res, err := h.runQuery(c.ConnectionID, f.PrepareStmt, values...)
	if err != nil {
		return err
	}
	callback(res)
	return nil

}

func (h *handler) WarningCount(c *mysql.Conn) uint16 {
	return 0
}

func (h *handler) ComResetConnection(c *mysql.Conn) {}

func (h *handler) Env() *vtenv.Environment {
	// Must not be nil
	env, err := vtenv.New(vtenv.Options{
		MySQLServerVersion: "8.0.30",
		TruncateUILen:      80,
		TruncateErrLen:     80,
	})
	if err != nil {
		fmt.Println("Error creating environment: ", err)
	}
	return env
}

func (h *handler) ComQuery(c *mysql.Conn, query string, callback func(*sqltypes.Result) error) error {
	h.Logger.Debug("Received query: ", "query", query, "connectionID", c.ConnectionID, "username", c.User)
	res, err := h.runQuery(c.ConnectionID, query)
	if err != nil {
		h.Logger.Debug("Error running query", "err", err, "query", query, "connectionID", c.ConnectionID, "username", c.User)
		return err
	}

	return callback(res)

}

func (h *handler) ComRegisterReplica(c *mysql.Conn, replicaHost string, replicaPort uint16, replicaUser string, replicaPassword string) error {
	return fmt.Errorf("replication is not supported")

}

func (h *handler) ComBinlogDump(c *mysql.Conn, logFile string, binlogPos uint32) error {
	return fmt.Errorf("replication is not supported")
}

func (h *handler) ComBinlogDumpGTID(c *mysql.Conn, logFile string, logPos uint64, gtidSet replication.GTIDSet) error {
	return fmt.Errorf("replication is not supported")
}

func (h *handler) ConnectionReady(c *mysql.Conn) {}

// Run a SQL query and return the result as a sqltypes.Result
//
// If specified, the query will be rewritten to be compatible with MySQL
func (h *handler) runQuery(connectionID uint32, query string, args ...interface{}) (*sqltypes.Result, error) {
	if !h.RewriteMySQLQueries {
		return h.runSimpleQuery(connectionID, query, args...)
	} else {
		return h.runQueryWithMySQLSpecific(connectionID, query, args...)
	}

}

// Run a SQL query to the h.DB connection, bypasing the MySQL compatibility layer,
// convert the result to a sqltypes.Result and return it
func (h *handler) runSimpleQuery(connectionID uint32, query string, args ...any) (*sqltypes.Result, error) {
	h.Logger.Debug("Running query: ", "query", query)

	// Retrieve the connection associated with the MySQL connection
	conn, ok := h.connectionMapperSQLite[connectionID]
	if !ok {
		h.Logger.Error("SQLite connection not found", "connectionID", connectionID)
		return nil, fmt.Errorf("SQLite connection not found")
	}

	rows, err := conn.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return convertSQLRowsToSQLResult(rows)
}

const numberRowsToAnalyze = 10

// Convert the rows of a SQL query to a sqltypes.Result
// understandable by the Vitess library
func convertSQLRowsToSQLResult(rows *sql.Rows) (*sqltypes.Result, error) {

	// Get the columns of the rows
	cols, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	// Create the receiving slice
	// that will be passed to the Scan method
	scannedValues := make([]interface{}, len(cols))

	res := &sqltypes.Result{}
	res.Rows = make([]sqltypes.Row, 0)
	res.Fields = make([]*querypb.Field, len(cols))

	// For each column, we append an interface to the scannedValues slice
	// that will later be filled with a pointer to the value of the column
	for i := range len(cols) {
		scannedValues[i] = new(interface{})
	}

	// Scan the rows one by one
	for rows.Next() {
		rowToInsert := make([]sqltypes.Value, len(cols))
		err = rows.Scan(scannedValues...)
		if err != nil {
			return nil, err
		}
		// What we have right now is an array of pointers to interfaces
		// We need to convert them to sqltypes.Value
		for i, val := range scannedValues {
			// Ensure the value is a pointer to something
			_, ok := val.(*interface{})
			if !ok {
				rowToInsert[i] = sqltypes.NULL
				continue
			}

			// Type switch between the supported types
			parsed := *(val.(*interface{}))
			switch parsed.(type) {
			case string:
				rowToInsert[i] = sqltypes.NewVarChar(parsed.(string))
			case int64:
				rowToInsert[i] = sqltypes.NewInt64(parsed.(int64))
			case []byte:
				rowToInsert[i] = sqltypes.NewVarBinary(string(parsed.([]byte)))
			case float64:
				rowToInsert[i] = sqltypes.NewFloat64(parsed.(float64))
			case nil:
				rowToInsert[i] = sqltypes.NULL
			default:
				rowToInsert[i] = sqltypes.NULL
			}

		}

		// Once we have scanned the row, we append it to the result
		res.Rows = append(res.Rows, rowToInsert)
	}

	// Create the columns of the result
	// If the query is from a table, we can use the DatabaseTypeName method
	// to get the type of the column in SQLite
	//
	// However, if we do SELECT 7.5 as myfloat, the driver is unable to
	// determine the type of the column, so DatabaseTypeName returns an empty string
	//
	// In this case, I think we should analyze the n first value of the column
	// from the rows and determine the type of the column
	//
	// The n number is the constant numberRowsToAnalyze
	for i, col := range cols {
		var fieldType querypb.Type = querypb.Type_NULL_TYPE
		typeName := col.DatabaseTypeName()
		if typeName == "" {
			// If the driver can't determine the type of the column
			// we analyze the n first rows until we find a non-null value
			// If we don't find any non-null value, we set the type to NULL
			for j := 0; j < len(res.Rows) && j < numberRowsToAnalyze; j++ {
				if res.Rows[j][i].IsNull() {
					continue
				}

				switch res.Rows[j][i].Type() {
				case querypb.Type_INT64:
					fieldType = querypb.Type_INT64
				case querypb.Type_VARCHAR:
					fieldType = querypb.Type_VARCHAR
				case querypb.Type_FLOAT64:
					fieldType = querypb.Type_FLOAT64
				case querypb.Type_VARBINARY:
					fieldType = querypb.Type_VARBINARY
				default:
					fieldType = querypb.Type_NULL_TYPE
				}
			}

		} else {
			// If typeName is not empty, we can use it
			switch strings.ToUpper(typeName) {
			case "INTEGER", "INT", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "UNSIGNED BIG INT", "INT2", "INT8":
				fieldType = querypb.Type_INT64
			case "TEXT", "VARCHAR", "CHAR", "CLOB", "NCHAR", "NVARCHAR", "VARCHAR(255)", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT":
				fieldType = querypb.Type_VARCHAR
			case "REAL", "real", "FLOAT", "float", "DOUBLE PRECISION", "DOUBLE", "NUMERIC", "DECIMAL":
				fieldType = querypb.Type_FLOAT64
			case "BLOB", "BINARY", "VARBINARY", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB":
				fieldType = querypb.Type_VARBINARY
			default:
				fieldType = querypb.Type_NULL_TYPE

			}

			// Because varchar takes a length, it can't be selected by the switch
			// We set it manually
			if fieldType == querypb.Type_NULL_TYPE &&
				(strings.HasPrefix(strings.ToUpper(typeName), "VARCHAR") ||
					strings.HasPrefix(strings.ToUpper(typeName), "CHAR") ||
					strings.HasPrefix(strings.ToUpper(typeName), "TEXT")) {

				fieldType = querypb.Type_VARCHAR
			}

		}
		res.Fields[i] = &querypb.Field{
			Name:     col.Name(),
			Type:     fieldType,
			Database: "main",
		}

		// Taken from https://github.com/vitessio/vitess/blob/main/go/mysql/schema.go#L45
		// MySQL Workbench required the charset to be set.
		if fieldType == querypb.Type_VARCHAR {
			// We set the charset to UTF8mb3 and the column length to the maximum of varchar
			res.Fields[i].ColumnLength = 65535
			res.Fields[i].Charset = uint32(collations.SystemCollation.Collation)
		} else if fieldType == querypb.Type_VARBINARY {
			res.Fields[i].ColumnLength = 65535
			res.Fields[i].Charset = collations.CollationBinaryID
		} else if fieldType == querypb.Type_INT64 || fieldType == querypb.Type_FLOAT64 {
			res.Fields[i].ColumnLength = 11
			res.Fields[i].Charset = uint32(collations.SystemCollation.Collation)
			res.Fields[i].Flags = uint32(querypb.MySqlFlag_BINARY_FLAG)

		}

	}
	return res, nil
}
