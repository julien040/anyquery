package rpc

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// InternalExchangeInterface is an interface that defines the methods
// that a plugin must implement to communicate with the main program
//
// This part should be handled by the plugin library and should not be
// implemented by the user
type InternalExchangeInterface interface {
	// Initialize is called when a new table is opened
	//
	// It is used by the main program to infer the schema of the tables
	Initialize(connectionID int, tableIndex int, config PluginConfig) (DatabaseSchema, error)

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
	Query(connectionID int, tableIndex int, cursorIndex int, constraint QueryConstraint) ([][]interface{}, bool, error)

	// Insert is a method that inserts rows into the table
	//
	// The rows are passed as a 2D slice of interface{} where each row is a slice
	// and each element in the row is an interface{} representing the value.
	Insert(connectionID int, tableIndex int, rows [][]interface{}) error

	// Update is a method that updates rows in the table
	//
	// The rows are passed as a 2D slice of interface{} where each row is a slice
	// and each element in the row is an interface{} representing the value.
	Update(connectionID int, tableIndex int, rows [][]interface{}) error

	// Delete is a method that deletes rows from the table
	//
	// The rows are passed as an array of primary keys
	Delete(connectionID int, tableIndex int, primaryKeys []interface{}) error

	// Close is a method that is called when the connection is closed
	//
	// It is used to free resources and close connections
	Close(connectionID int) error
}

// PluginConfig is a struct that holds the configuration for the plugin
//
// It is mostly used to specify user-defined configuration
// and is passed to the plugin during initialization
type PluginConfig map[string]interface{}

// Returns a string value for the key in the plugin configuration
//
// If the key does not exist or is not a string, it returns an empty string
func (p PluginConfig) GetString(key string) string {
	inter, ok := p[key]
	if ok {
		switch v := inter.(type) {
		case string:
			return v
		case int64:
			return strconv.FormatInt(v, 10)
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(v)
		}
	}
	return ""
}

// Returns an int value for the key in the plugin configuration
//
// If the key does not exist or is not an int, it returns 0
func (p PluginConfig) GetInt(key string) int64 {
	inter, ok := p[key]
	if ok {
		switch v := inter.(type) {
		case int64:
			return v
		case float64:
			return int64(v)
		case string:
			val, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0
			}
			return val
		}
	}
	return 0
}

// Returns a float value for the key in the plugin configuration
//
// If the key does not exist or is not a float, it returns 0
func (p PluginConfig) GetFloat(key string) float64 {
	inter, ok := p[key]
	if ok {
		switch v := inter.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case string:
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return 0
			}
			return val
		}
	}
	return 0
}

// Returns a bool value for the key in the plugin configuration
//
// If the key does not exist or is not a bool, it returns false
func (p PluginConfig) GetBool(key string) bool {
	inter, ok := p[key]
	if ok {
		switch v := inter.(type) {
		case bool:
			return v
		case int64:
			return v == 1
		case float64:
			return int64(v) == 1
		case string:
			val, err := strconv.ParseBool(v)
			if err != nil {
				lowered := strings.ToLower(v)
				return lowered == "yes" || lowered == "y"
			}
			return val
		}
	}
	return false
}

// Returns a string array for the key in the plugin configuration
//
// If the key does not exist or is not a string array, it returns nil
func (p PluginConfig) GetStringArray(key string) []string {
	inter, ok := p[key]
	if ok {
		/* if arr, ok := inter.([]string); ok {
			return arr
		} */
		switch v := inter.(type) {
		case []string:
			return v
		// To accomodate for some unmarshalling issues
		case []interface{}:
			arr := make([]string, 0, len(v))
			for _, val := range v {
				switch val := val.(type) {
				case string:
					arr = append(arr, val)
				case int64:
					arr = append(arr, strconv.FormatInt(val, 10))
				case float64:
					arr = append(arr, strconv.FormatFloat(val, 'f', -1, 64))
				case bool:
					arr = append(arr, strconv.FormatBool(val))
				}
			}
			return arr
		}
	}
	return nil
}

// Returns an int array for the key in the plugin configuration
//
// If the key does not exist or is not an int array, it returns nil
func (p PluginConfig) GetIntArray(key string) []int64 {
	inter, ok := p[key]
	if ok {
		switch v := inter.(type) {
		case []int64:
			return v
		case []interface{}:
			arr := make([]int64, 0, len(v))
			for _, val := range v {
				switch val := val.(type) {
				case int64:
					arr = append(arr, val)
				case float64:
					arr = append(arr, int64(val))
				case string:
					num, err := strconv.ParseInt(val, 10, 64)
					if err == nil {
						arr = append(arr, num)
					}
				case bool:
					if val {
						arr = append(arr, 1)
					} else {
						arr = append(arr, 0)
					}
				}
			}
			return arr
		}
	}
	return nil
}

// Returns a float array for the key in the plugin configuration
//
// If the key does not exist or is not a float array, it returns nil
func (p PluginConfig) GetFloatArray(key string) []float64 {
	inter, ok := p[key]
	if ok {
		switch v := inter.(type) {
		case []float64:
			return v
		case []interface{}:
			arr := make([]float64, 0, len(v))
			for _, val := range v {
				switch val := val.(type) {
				case float64:
					arr = append(arr, val)
				case int64:
					arr = append(arr, float64(val))
				case string:
					num, err := strconv.ParseFloat(val, 64)
					if err == nil {
						arr = append(arr, num)
					}
				case bool:
					if val {
						arr = append(arr, 1)
					} else {
						arr = append(arr, 0)
					}
				}
			}
			return arr
		}
	}
	return nil
}

// Returns a bool array for the key in the plugin configuration
//
// If the key does not exist or is not a bool array, it returns nil
func (p PluginConfig) GetBoolArray(key string) []bool {
	inter, ok := p[key]
	if ok {
		switch v := inter.(type) {
		case []bool:
			return v
		case []interface{}:
			arr := make([]bool, 0, len(v))
			for _, val := range v {
				switch val := val.(type) {
				case bool:
					arr = append(arr, val)
				case int64:
					arr = append(arr, val == 1)
				case float64:
					arr = append(arr, int64(val) == 1)
				case string:
					b, err := strconv.ParseBool(val)
					if err == nil {
						arr = append(arr, b)
					}
				}
			}
			return arr
		}
	}
	return nil
}

// Holds the metadata of a table
//
// It is used to provide information about the table to end-users
type TableMetadata struct {
	// The description of the table
	//
	// Useful for LLMs to figure out what the table is about, and which tables are related for joins
	Description string `json:"description"`
	// A few SQL queries that show how to use the table
	// (such as the parameters, the join with other tables, etc.)
	//
	// Prefix each examples with a -- comment that explains what the query does
	//
	//	[]string{"-- Get all the users\nSELECT * FROM users"}
	Examples []string `json:"examples"`
}

// PluginManifest is a struct that holds the metadata of the plugin
type PluginManifest struct {
	Name        string
	Version     string
	Author      string
	Description string
	// A list of tables that the plugin will provide
	Tables []string

	TablesMetadata map[string]TableMetadata `json:"tables_metadata"`

	UserConfig []PluginConfigField
}

type PluginConfigField struct {
	Name        string
	Required    bool
	Type        string // string, int, float, bool, []string, []int, []float, []bool
	Description string
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

	// How many rows should anyquery buffer before sending them to the plugin
	//
	// If set to 0, the main program will send the rows one by one
	// It is used to reduce the number of API calls of plugins
	BufferInsert uint

	// How many rows should anyquery buffer before sending them to the plugin
	//
	// If set to 0, the main program will send the rows one by one
	// It is used to reduce the number of API calls of plugins
	BufferUpdate uint

	// How many rows should anyquery buffer before sending them to the plugin
	//
	// If set to 0, the main program will send the rows one by one
	// It is used to reduce the number of API calls of plugins
	BufferDelete uint

	// Whether the plugin can handle partial updates
	//
	// If this field is set to true, when an UPDATE statement is issued,
	// any non modified columns will be set to nil
	//
	// For example, if the row is
	//	[1, "hello", 3.14]
	// and the update statement is
	//	[1, "world"]
	// the plugin will receive
	//	[1, "world", nil]
	//
	// If set to false, the plugin will receive
	//	[1, "world", 3.14]
	PartialUpdate bool

	// A description of the table
	// (Not used by early versions of anyquery)
	//
	// Useful for LLMs to figure out what the table is about
	Description string
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
	// ColumnTypeBool represents an INTEGER column, and must be either 0 or 1
	ColumnTypeBool
	// ColumnTypeDateTime represents a TEXT column that must be in the RFC3339 format
	ColumnTypeDateTime
	// ColumnTypeDate represents a TEXT column that must be in YYYY-MM-DD format
	ColumnTypeDate
	// ColumnTypeTime represents a TEXT column that must be in the HH:MM:SS format
	ColumnTypeTime
	// ColumnTypeJSON represents a TEXT column that must be a valid JSON string
	ColumnTypeJSON
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

	// Whether the column is required
	//
	// If a column is required, the user must provide a value for it.
	// If not, the query will fail.
	IsRequired bool

	// A description of the column
	// (Not used by early versions of anyquery)
	Description string
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

// Returns the sha256 hash of the query constraint for caching purposes
//
// Internally, it sorts the columns by column index, operator, and value,
// sorts the order by constraints by column index and descending order,
// marshals the query constraint to JSON, and hashes it with sha256
func (qc QueryConstraint) Hash() string {
	// Sort the columns by column index, operator, and value
	// to ensure the hash is the same for the same constraints
	clonedCol := slices.Clone(qc.Columns)
	slices.SortFunc(clonedCol, func(a ColumnConstraint, b ColumnConstraint) int {
		if a.ColumnID != b.ColumnID {
			return a.ColumnID - b.ColumnID
		}
		if a.Operator != b.Operator {
			return int(a.Operator) - int(b.Operator)
		}
		return 0
	})

	clonedOrder := slices.Clone(qc.OrderBy)
	slices.SortFunc(clonedOrder, func(a OrderConstraint, b OrderConstraint) int {
		if a.ColumnID != b.ColumnID {
			return a.ColumnID - b.ColumnID
		}
		if a.Descending != b.Descending {
			if a.Descending {
				return -1
			}
			return 1
		}
		return 0
	})

	clone := QueryConstraint{
		Columns: clonedCol,
		Limit:   qc.Limit,
		Offset:  qc.Offset,
		OrderBy: clonedOrder,
	}

	marshalled, err := json.Marshal(clone)
	if err != nil {
		return ""
	}

	hashed := sha256.Sum256(marshalled)
	return fmt.Sprintf("%x", hashed)
}

// Returns the column constraint for the given column at the index columnID
//
// If the column does not exist, it returns nil
func (qc QueryConstraint) GetColumnConstraint(columnID int) *ColumnConstraint {
	for _, c := range qc.Columns {
		if c.ColumnID == columnID {
			return &c
		}
	}
	return nil
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

// Whether the column constraint is the equal operator
func (cc *ColumnConstraint) IsEqual() bool {
	if cc == nil {
		return false
	}
	return cc.Operator == OperatorEqual
}

// Returns the string value of the column constraint
//
// If the value is not a string, it'll try to convert it to a string
// If it fails or the constraints does not exist, it returns an empty string
func (cc *ColumnConstraint) GetStringValue() string {
	if cc == nil {
		return ""
	}
	switch v := cc.Value.(type) {
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	// Shoud be impossible
	case bool:
		return strconv.FormatBool(v)
	}
	return ""
}

// Returns the int value of the column constraint
//
// If the value is not an int, it'll try to convert it to an int
// If it fails or the constraints does not exist, it returns 0
func (cc *ColumnConstraint) GetIntValue() int64 {
	if cc == nil {
		return 0
	}
	switch v := cc.Value.(type) {
	case string:
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return val
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// Returns the float value of the column constraint
//
// If the value is not a float, it'll try to convert it to a float
// If it fails or the constraints does not exist, it returns 0
func (cc *ColumnConstraint) GetFloatValue() float64 {
	if cc == nil {
		return 0
	}
	switch v := cc.Value.(type) {
	case string:
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return val
	case float64:
		return v
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// Returns the bool value of the column constraint
//
// If the value is not a bool, it'll try to convert it to a bool
// If it fails or the constraints does not exist, it returns false
func (cc *ColumnConstraint) GetBoolValue() bool {
	if cc == nil {
		return false
	}
	switch v := cc.Value.(type) {
	// Should be impossible
	case bool:
		return v
	case int64, int:
		return v == 1
	case float64:
		return v == 1
	case string:
		val, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return val
	default:
		return false
	}

}

// Returns the time value of the column constraint
//
// If the value is not a time.Time, it'll try to convert it to a time.Time
// If it fails or the constraints does not exist, it returns the zero value of time.Time
//
// Supported formats are:
//   - time.RFC3339
//   - time.RFC822
//   - time.RubyDate
//   - time.UnixDate
//   - time.DateTime
//   - time.DateOnly
//   - Unix timestamp (int64 or float64)
func (cc *ColumnConstraint) GetTimeValue() time.Time {
	if cc == nil {
		return time.Time{}
	}

	supportedFormats := []string{
		time.RFC3339,
		time.RFC822,
		time.RubyDate,
		time.UnixDate,
		time.DateTime,
		time.DateOnly,
	}

	switch v := cc.Value.(type) {
	// Should be impossible
	case time.Time:
		return v
	case *time.Time:
		if v == nil {
			return time.Time{}
		}
		return *v
	case string:
		for _, format := range supportedFormats {
			t, err := time.Parse(format, v)
			if err == nil {
				return t
			}
		}
		return time.Time{}
	case int64:
		return time.Unix(v, 0)
	case int:
		return time.Unix(int64(v), 0)
	case float64:
		return time.Unix(int64(v), 0)
	default:
		return time.Time{}
	}
}
