package module

import (
	"fmt"
	"strings"
	"testing"

	"github.com/huandu/go-sqlbuilder"
	"github.com/stretchr/testify/require"
)

func TestEscapeMySQLLiteral(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "well-known text passes through", value: "POINT(1 2)", want: "POINT(1 2)"},
		{name: "linestring passes through", value: "LINESTRING(0 0, 1 1)", want: "LINESTRING(0 0, 1 1)"},
		{name: "single quote is doubled", value: "O'Brien", want: "O''Brien"},
		{name: "quote breakout payload is neutralized", value: "x' OR '1'='1", want: "x'' OR ''1''=''1"},
		{name: "trailing backslash cannot swallow the closing quote", value: `x\`, want: `x\\`},
		{name: "backslash then quote is escaped", value: `x\' OR 1=1 --`, want: `x\\'' OR 1=1 --`},
		{name: "non-string values are stringified", value: 42, want: "42"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, escapeMySQLLiteral(tt.value))
		})
	}
}

// The geometry special case inlines the value into ST_GeomFromText('...')
// because the builder cannot wrap a placeholder in a function call. Assert on
// the assembled INSERT that a hostile value stays inside that literal instead
// of terminating it and appending attacker-controlled SQL.
func TestMySQLGeometryInsertKeepsValueInsideLiteral(t *testing.T) {
	hostile := "x', (SELECT SLEEP(5)) -- "

	builder := sqlbuilder.NewInsertBuilder()
	builder.InsertInto("spatial_table")
	builder.Cols("id", "shape")
	builder.Values(1, sqlbuilder.Raw(fmt.Sprintf("ST_GeomFromText('%s')", escapeMySQLLiteral(hostile))))
	builder.SetFlavor(sqlbuilder.MySQL)
	query, args := builder.Build()

	require.Equal(t, "INSERT INTO spatial_table (id, shape) VALUES (?, ST_GeomFromText('x'', (SELECT SLEEP(5)) -- '))", query)
	require.Len(t, args, 1, "only the id may be parameterized; the geometry value is inlined")
	require.True(t, strings.Contains(query, "ST_GeomFromText('x'',"), "value must be escaped inside the literal: %s", query)
}
