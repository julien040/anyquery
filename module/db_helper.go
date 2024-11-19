package module

import (
	"github.com/huandu/go-sqlbuilder"
	"github.com/mattn/go-sqlite3"
)

type databaseColumn struct {
	// The name of the column in the database
	Realname string
	// The name of the column locally
	SQLiteName string
	// The type of the column in SQLite
	Type string
	// The type of the column in the remote database
	RemoteType string
	// Whether the column is supported. If not, it's not queryable, insertable, or updatable
	Supported bool
	// DefaultValue value for the column
	DefaultValue interface{}
}

// constructSQLQuery constructs a SQL query from the constraints and order by
//
// It returns the SQL query, the limit index, and the offset index in the constraints
func constructSQLQuery(
	cst []sqlite3.InfoConstraint,
	ob []sqlite3.InfoOrderBy,
	columns []databaseColumn,
	table string,

) (query *sqlbuilder.SelectBuilder, limit int, offset int, used []bool) {

	// Initialize the SQL query builder
	query = sqlbuilder.NewSelectBuilder()
	// Add all the columns to the query
	cols := []string{}
	for _, col := range columns {
		cols = append(cols, col.Realname)
	}

	query.Select(cols...).From(table)

	// Add the constraints (where, limit, offset)
	limit = -1
	offset = -1

	used = make([]bool, len(cst))

	andConditions := []string{}
	j := 0
	for i, c := range cst {
		// If the constraint is not usable, we skip it
		if !c.Usable {
			continue
		}
		// Note the LIMIT and OFFSET constraints indexes in the constraints
		if c.Op == sqlite3.OpLIMIT {
			limit = j
			used[i] = true
			j++
			continue
		} else if c.Op == sqlite3.OpOFFSET {
			offset = j
			used[i] = true
			j++
			continue
		}

		// If we don't have information about the column, we skip it
		if c.Column < 0 || c.Column >= len(columns) {
			continue
		}

		// If the column is not supported, we skip it
		colInfo := columns[c.Column]
		if !colInfo.Supported {
			continue
		}

		switch c.Op {
		case sqlite3.OpEQ:
			andConditions = append(andConditions, query.Equal(colInfo.Realname, colInfo.DefaultValue))
		case sqlite3.OpGT:
			andConditions = append(andConditions, query.GreaterThan(colInfo.Realname, colInfo.DefaultValue))
		case sqlite3.OpGE:
			andConditions = append(andConditions, query.GreaterEqualThan(colInfo.Realname, colInfo.DefaultValue))
		case sqlite3.OpLT:
			andConditions = append(andConditions, query.LessThan(colInfo.Realname, colInfo.DefaultValue))
		case sqlite3.OpLE:
			andConditions = append(andConditions, query.LessEqualThan(colInfo.Realname, colInfo.DefaultValue))
		case sqlite3.OpLIKE:
			andConditions = append(andConditions, query.Like(colInfo.Realname, colInfo.DefaultValue))
		case sqlite3.OpGLOB:
			// Not supported
			continue
		case sqlite3.OpREGEXP:
			// Not supported
			continue
		case sqlite3.OpLIMIT:
			limit = int(c.Column)
		case sqlite3.OpOFFSET:
			offset = int(c.Column)
		}
		used[i] = true
		j++
	}

	query.Where(andConditions...)

	// Add the order by
	for _, o := range ob {
		if o.Desc {
			query.OrderBy(columns[o.Column].Realname + " DESC")
		} else {
			query.OrderBy(columns[o.Column].Realname + " ASC")
		}
	}

	return
}

type SQLQueryToExecute struct {
	// The SQL query to execute
	Query string

	// The arguments to pass to the query
	Args []interface{}

	// The index in the constraints for the limit (-1 if not present)
	LimitIndex int

	// The index in the constraints for the offset (-1 if not present)
	OffsetIndex int
}

func castInt(value interface{}) int64 {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}

	return 0
}

func castFloat(value interface{}) float64 {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case float32:
		return float64(v)
	case float64:
		return v
	}

	return 0
}
