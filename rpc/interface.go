package rpc

// InternalExchangeInterface is an interface that defines the methods
// that a plugin must implement to communicate with the main program
//
// This part should be handled by the plugin library and should not be
// implemented by the user
type InternalExchangeInterface interface {
	// Initialize is a method that is called when the plugin is initialized
	//
	// It is called once when the plugin is loaded and is used by the main
	// program to infer the schema of the tables
	Initialize(tableIndex int, config PluginConfig) (DatabaseSchema, error)

	// Query is a method that returns rows for a given SELECT query
	//
	// Constraints are passed as arguments for optimization purposes
	// However, the plugin is free to ignore them because
	// the main program will filter the results to match the constraints
	//
	// The first return value is a 2D slice of interface{} where each row is a slice
	// and each element in the row is an interface{} representing the value.
	// The second return value is a boolean that specifies whether the cursor is exhausted
	// The order and type of the values should match the schema of the table
	Query(tableIndex int, cursorIndex int, constraint QueryConstraint) ([][]interface{}, bool, error)
}

// PluginConfig is a struct that holds the configuration for the plugin
//
// It is mostly used to specify user-defined configuration
// and is passed to the plugin during initialization
type PluginConfig map[string]string

// PluginManifest is a struct that holds the metadata of the plugin
//
// It is often represented as a JSON file in the plugin directory
type PluginManifest struct {
	Name        string
	Version     string
	Author      string
	Description string
	// A list of tables that the plugin will provide
	Tables []string

	UserConfig []PluginConfigField
}

type PluginConfigField struct {
	Name     string
	Required bool
}

// DatabaseSchema holds the schema of the database
//
// It must stay the same throughout the lifetime of the plugin
// and for every cursor opened.
//
// One and only field must be the primary key. If you don't have a primary key,
// you can generate a unique key. The primary key must be unique for each row.
// It is used to update and delete rows.
type DatabaseSchema struct {
	// The columns of the table
	Columns []DatabaseSchemaColumn
	// The primary key is the index of the column where each row has a unique value
	//
	// If set to -1, it means the table does not have a primary key.
	// Therefore, the main program will generate a unique key for each row.
	// However, the table won't be able to update or delete rows.
	//
	// The primary key column type is either ColumnTypeInt or ColumnTypeString
	PrimaryKey int

	// Whether the plugin can handle an INSERT statement
	HandlesInsert bool
	// Whether the plugin can handle an UPDATE statement
	HandlesUpdate bool
	// Whether the plugin can handle a DELETE statement
	HandlesDelete bool

	// The following fields are used to optimize the queries

	// HandleOffset is a boolean that specifies whether the plugin can handle the OFFSET clause.
	// If not, the main program will skip the n offseted rows.
	HandleOffset bool
}

// ColumnType is an enum that represents the type of a column
type ColumnType int8

const (
	// ColumnTypeInt represents an INTEGER column
	ColumnTypeInt ColumnType = iota
	// ColumnTypeFloat represents a REAL column
	ColumnTypeFloat
	// ColumnTypeString represents a TEXT column
	ColumnTypeString
	// ColumnTypeBlob represents a BLOB column
	ColumnTypeBlob
)

type DatabaseSchemaColumn struct {
	// The name of the column
	Name string
	// The type of the column (INTEGER, REAL, TEXT, BLOB)
	Type ColumnType
	// Whether the column is a parameter
	//
	// If a column is a parameter, it will be hidden from the user
	// in the result of a SELECT query
	// and can be passed as an argument of the table
	//
	// For example, a parameter column named account_id
	// can be used as such
	//	SELECT * FROM mytable(<account_id>)
	//	SELECT * FROM mytable WHERE account_id = <account_id>
	//
	// Arguments order is the same as the order of the columns in the schema
	IsParameter bool
}

// These operators are used in the ColumnConstraint struct
// They are extracted from https://tinyurl.com/28seb4bs

type Operator int8

const (
	// OperatorEqual is the equal operator =
	OperatorEqual = 2

	// OperatorGreater is the greater than operator >
	OperatorGreater = 4

	// OperatorLessOrEqual is the less than or equal operator <=
	OperatorLessOrEqual = 8

	// OperatorLess is the less than operator <
	OperatorLess = 16

	// OperatorGreaterOrEqual is the greater than or equal operator >=
	OperatorGreaterOrEqual = 32

	// OperatorMatch is the match operator
	OperatorMatch = 64

	// OperatorLike is the like operator
	OperatorLike = 65

	// OperatorGlob is the glob operator.
	// It represents a simple pattern matching operator
	// with a UNIX syntax
	//
	// Note: A like operator is not provided because anyquery
	// converts it to a glob operator
	OperatorGlob = 66

	// OperatorRegexp is the regexp operator
	OperatorRegexp = 67

	// OperatorNotEqual is the not equal operator !=
	OperatorNotEqual = 68
	// OperatorISNOT is the IS NOT operator
	//
	// Note: will be converted to OperatorNotEqual
	OperatorIsNot = 69

	// OperatorISNOTNULL is the IS NOT NULL operator
	//
	// Note: will be converted to OperatorNotEqual with value nil
	OperatorIsNotNull = 70

	// OperatorISNULL is the IS NULL operator
	//
	// Note: will be converted to OperatorEqual with value nil
	OperatorIsNull = 71

	// OperatorIS is the IS operator
	//
	// Note: will be converted to OperatorEqual
	OperatorIs = 72

	// OperatorLimit is the LIMIT statement in a SQL query
	OperatorLimit = 73

	// OperatorOFFSET is the OFFSET statement in a SQL query
	OperatorOffset = 74
)

// QueryConstraint is a struct that holds the constraints for a SELECT query
//
// It specifies the WHERE conditions in the Columns field,
// the LIMIT and OFFSET in the Limit and Offset fields,
// and the ORDER BY clause in the OrderBy field
type QueryConstraint struct {
	// The constraints for each column (can be skipped and SQLite will handle it)
	Columns []ColumnConstraint

	// The maximum number of rows to return
	//
	// If set to -1, it means no limit
	Limit int

	// The number of rows to skip
	//
	// If set to -1, it means no offset
	Offset int

	// The order by constraints (can be skipped and SQLite will handle it)
	OrderBy []OrderConstraint
}

type OrderConstraint struct {
	ColumnID   int
	Descending bool
}

type ColumnConstraint struct {
	ColumnID int
	Operator Operator
	Value    interface{}
}
