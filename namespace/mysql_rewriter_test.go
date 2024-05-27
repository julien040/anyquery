package namespace

import (
	"testing"

	"github.com/stretchr/testify/require"
	"vitess.io/vitess/go/vt/sqlparser"
)

func TestSelectRewriter(t *testing.T) {
	t.Parallel()

	type testCase struct {
		query string
		want  string
	}

	tests := map[string]testCase{
		"Ensure a simple SELECT query is not rewritten": {
			query: "SELECT * FROM my_table",
			want:  "select * from my_table",
		},
		"A known function must be rewritten": {
			query: "SELECT database()",
			want:  "select 'main' from dual",
		},
		"An unknown function must be left as is": {
			query: "SELECT unknown_function()",
			want:  "select unknown_function() from dual",
		},
		"A known session variable must be rewritten": {
			query: "SELECT  @@session.auto_increment_increment AS auto_increment_increment",
			want:  "select 1 as auto_increment_increment from dual",
		},
		"A known global variable must be rewritten": {
			query: "SELECT  @@character_set_client AS character_set_client",
			want:  "select 'utf8mb4' as character_set_client from dual",
		},
		"An unknown variable must be replaced by an empty string": {
			query: "SELECT  @my_variable AS my_variable",
			want:  "select '' as my_variable from dual",
		},
		"A collation must be replaced by the BINARY collation and the ASC order": {
			query: "SELECT * FROM my_table ORDER BY k COLLATE utf8mb4_unicode_ci",
			want:  "select * from my_table order by k collate BINARY asc",
		},
		"A DESC collation must be replaced by the BINARY collation and the DESC order": {
			query: "SELECT * FROM my_table ORDER BY k COLLATE utf8mb4_unicode_ci DESC",
			want:  "select * from my_table order by k collate BINARY desc",
		},
		"A collation in a WHERE clause must be replaced by the BINARY collation": {
			query: "SELECT * FROM my_table WHERE k COLLATE utf8mb4_unicode_ci = 'value'",
			want:  "select * from my_table where k collate BINARY = 'value'",
		},
		"A collation in a HAVING clause must be replaced by the BINARY collation": {
			query: "SELECT * FROM my_table HAVING k COLLATE utf8mb4_unicode_ci = 'value'",
			want:  "select * from my_table having k collate BINARY = 'value'",
		},
		"A collation in a GROUP BY clause must be replaced by the BINARY collation": {
			query: "SELECT * FROM my_table GROUP BY k COLLATE utf8mb4_unicode_ci",
			want:  "select * from my_table group by k collate BINARY",
		},
		"A collation from SQLite must not be rewritten": {
			query: "SELECT * FROM my_table ORDER BY k COLLATE RTRIM",
			want:  "select * from my_table order by k collate RTRIM asc",
		},
		"A convert expression must be rewritten to a CAST expression": {
			query: "SELECT CONVERT('2021-01-01', DATE)",
			want:  "select cast('2021-01-01' as DATE) from dual",
		},
		"A locate expression must be rewritten to a instr expression": {
			query: "SELECT LOCATE('a', 'abc')",
			want:  "select instr('abc', 'a') from dual",
		},
		"A MySQL specific function must be rewritten to a SQLite function": {
			query: "SELECT IF(1=1, 'true', 'false'), LEFT('abc', 1), upper('abc')",
			want:  "select iif(1 = 1, 'true', 'false'), ltrim('abc', 1), upper('abc') from dual",
		},
		"A MySQL specific function can be replaced by a literal": {
			query: "SELECT database(), user()",
			want:  "select 'main', 'root' from dual",
		},
		"A MySQL variable can be replaced by a literal": {
			query: "SELECT @@session.auto_increment_increment, @@character_set_client, @my_variable, @@global.unknown_variable",
			want:  "select 1, 'utf8mb4', '', '' from dual",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stmtType, stmt, err := getQueryType(tt.query)
			if err != nil {
				t.Fatalf("unexpected error while parsing: %v", err)
			}

			if stmtType != sqlparser.StmtSelect {
				t.Fatalf("expected SELECT statement, got: %v", stmtType)
			}

			rewriteSelectStatement(&stmt)

			if got := sqlparser.String(stmt); got != tt.want {
				t.Errorf("got: %s, want: %s", got, tt.want)
			}
		})
	}

}

// To ensure the sqlparser.StatementType is correctly detected
// and not changed in future versions
func TestQueryTypeDetection(t *testing.T) {
	tests := []struct {
		query string
		want  sqlparser.StatementType
	}{
		{
			query: "SELECT * FROM my_table",
			want:  sqlparser.StmtSelect,
		},
		{
			query: "SHOW CREATE TABLE my_table",
			want:  sqlparser.StmtShow,
		},
		{
			query: "SHOW TABLES",
			want:  sqlparser.StmtShow,
		},
		{
			query: "INSERT INTO my_table VALUES (1, 'value')",
			want:  sqlparser.StmtInsert,
		},
		{
			query: "UPDATE my_table SET k = 'value'",
			want:  sqlparser.StmtUpdate,
		},
		{
			query: "DELETE FROM my_table WHERE k = 'value'",
			want:  sqlparser.StmtDelete,
		},
		{
			query: "CREATE TABLE my_table (k INT)",
			want:  sqlparser.StmtDDL,
		},
		{
			query: "START TRANSACTION",
			want:  sqlparser.StmtBegin,
		},
		{
			query: "COMMIT",
			want:  sqlparser.StmtCommit,
		},
	}

	for _, tt := range tests {

		queryType, _, err := getQueryType(tt.query)
		if err != nil {
			t.Fatalf("unexpected error while parsing: %v", err)
		}

		require.Equal(t, tt.want, queryType, "unexpected query type for %s", tt.query)
	}
}
