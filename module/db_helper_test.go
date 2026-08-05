package module

import (
	"testing"

	"github.com/huandu/go-sqlbuilder"
	sqlite3 "github.com/julien040/go-sqlite3-anyquery"
	"github.com/stretchr/testify/require"
)

func TestPostgresQueryQuotesRawColumnNamesOnce(t *testing.T) {
	columns := []databaseColumn{
		{
			Realname:  "version",
			Supported: true,
		},
	}

	query, _, _, _ := constructSQLQuery(
		[]sqlite3.InfoConstraint{{Column: 0, Op: sqlite3.OpEQ, Usable: true}},
		[]sqlite3.InfoOrderBy{{Column: 0}},
		columns,
		`"public"."schema_migrations"`,
		sqlbuilder.PostgreSQL,
	)

	got, _ := query.Build()
	require.Equal(t, `SELECT "version" FROM "public"."schema_migrations" WHERE "version" = $1 ORDER BY "version" ASC`, got)
}

func TestEfficientPostgresQueryQuotesRawColumnNamesOnce(t *testing.T) {
	columns := []databaseColumn{
		{
			Realname:  "dirty",
			Supported: true,
		},
	}

	query, _, _, _ := efficientConstructSQLQuery(
		nil,
		nil,
		columns,
		`"public"."schema_migrations"`,
		1,
		sqlbuilder.PostgreSQL,
	)

	got, _ := query.Build()
	require.Equal(t, `SELECT "dirty" FROM "public"."schema_migrations"`, got)
}
