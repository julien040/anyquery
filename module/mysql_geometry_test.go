package module

import (
	"testing"

	"github.com/huandu/go-sqlbuilder"
	"github.com/stretchr/testify/require"
)

// A hostile geometry value must reach MySQL as a bound argument, never as
// SQL text, for both the INSERT and UPDATE paths of MySQLTable.
func TestMySQLGeometryValueIsParameterized(t *testing.T) {
	hostile := "x', (SELECT SLEEP(5)) -- "

	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("spatial_table")
	ib.Cols("id", "shape")
	ib.Values(1, mysqlGeometryValue(hostile))
	ib.SetFlavor(sqlbuilder.MySQL)
	query, args := ib.Build()

	require.Equal(t, "INSERT INTO spatial_table (id, shape) VALUES (?, ST_GeomFromText(?))", query)
	require.Equal(t, []any{1, hostile}, args)

	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update("spatial_table")
	ub.Set(ub.Assign("shape", mysqlGeometryValue(hostile)))
	ub.Where(ub.Equal("id", 1))
	ub.SetFlavor(sqlbuilder.MySQL)
	query, args = ub.Build()

	require.Equal(t, "UPDATE spatial_table SET shape = ST_GeomFromText(?) WHERE id = ?", query)
	require.Equal(t, []any{hostile, 1}, args)
}
