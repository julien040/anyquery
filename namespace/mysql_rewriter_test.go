package namespace

import (
	"testing"

	"github.com/julien040/anyquery/other/sqlparser"
	"github.com/stretchr/testify/require"
)

func TestRewriter(t *testing.T) {
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
			want:  "select 'main'",
		},
		"An unknown function must be left as is": {
			query: "SELECT unknown_function()",
			want:  "select unknown_function()",
		},
		"A known session variable must be rewritten": {
			query: "SELECT  @@session.auto_increment_increment AS auto_increment_increment",
			want:  "select 1 as auto_increment_increment",
		},
		"A known global variable must be rewritten": {
			query: "SELECT  @@character_set_client AS character_set_client",
			want:  "select 'utf8mb4' as character_set_client",
		},
		"An unknown variable must be replaced by an empty string": {
			query: "SELECT  @my_variable AS my_variable",
			want:  "select '' as my_variable",
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
			want:  "select cast('2021-01-01' as DATE)",
		},
		"A locate expression must be rewritten to a instr expression": {
			query: "SELECT LOCATE('a', 'abc')",
			want:  "select instr('abc', 'a')",
		},
		"A MySQL specific function must be rewritten to a SQLite function": {
			query: "SELECT IF(1=1, 'true', 'false'), LEFT('abc', 1), upper('abc')",
			want:  "select iif(1 = 1, 'true', 'false'), ltrim('abc', 1), upper('abc')",
		},
		"A MySQL specific function can be replaced by a literal": {
			query: "SELECT database(), user()",
			want:  "select 'main', 'root'",
		},
		"A MySQL variable can be replaced by a literal": {
			query: "SELECT @@session.auto_increment_increment, @@character_set_client, @my_variable, @@global.unknown_variable",
			want:  "select 1, 'utf8mb4', '', ''",
		},

		// Ensure the query is not modified
		"SELECT query with a WHERE clause": {
			query: "SELECT * FROM my_table WHERE k = 'value'",
			want:  "select * from my_table where k = 'value'",
		},
		"SELECT query with a table argument": {
			query: "SELECT * FROM my_table(arg1, 'string', 1, true, 1.1)",
			want:  "select * from my_table(arg1, 'string', 1, true, 1.1)",
		},
		"DELETE query with a WHERE clause and a table argument": {
			query: "DELETE FROM my_table(arg1, 'string', 1, true, 1.1) WHERE k = 'value'",
			want:  "delete from my_table(arg1, 'string', 1, true, 1.1) where k = 'value'",
		},
		"UPDATE query with a WHERE clause and a table argument": {
			query: "UPDATE my_table(arg1, 'string', 1, true, 1.1) SET k = 'value' WHERE k = 'value'",
			want:  "update my_table(arg1, 'string', 1, true, 1.1) set k = 'value' where k = 'value'",
		},
		"UNION query": {
			query: "SELECT * FROM my_table UNION SELECT * FROM my_table2",
			want:  "select * from my_table union select * from my_table2",
		},
		"Union query with table arguments": {
			query: "SELECT * FROM my_table(arg1, 'string', 1, true, 1.1) UNION SELECT * FROM my_table2(arg1, 'string', 1, true, 1.1)",
			want:  "select * from my_table(arg1, 'string', 1, true, 1.1) union select * from my_table2(arg1, 'string', 1, true, 1.1)",
		},
		"With a subquery": {
			query: "SELECT * FROM (SELECT * FROM my_table) AS subquery", // As is required by SQLite
			want:  "select * from (select * from my_table) as subquery",
		},
		"SELECT query with a CTE": {
			query: "WITH my_cte AS (SELECT * FROM my_table) SELECT * FROM my_cte",
			want:  "with my_cte as (select * from my_table) select * from my_cte",
		},
		"SELECT query with a CTE, a UNION and table arguments": {
			query: "WITH my_cte AS (SELECT * FROM my_table(arg1, 'string', 1, true, 1.1)) SELECT * FROM my_cte UNION SELECT * FROM my_table2(arg1, 'string', 1, true, 1.1)",
			want:  "with my_cte as (select * from my_table(arg1, 'string', 1, true, 1.1)) select * from my_cte union select * from my_table2(arg1, 'string', 1, true, 1.1)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, stmt, err := GetQueryType(tt.query)
			if err != nil {
				t.Fatalf("unexpected error while parsing: %v", err)
			}

			rewriteSelectStatement(&stmt)

			require.Equal(t, tt.want, sqlparser.String(stmt), "unexpected query for %s", tt.query)
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
		// Statements with arguments
		{
			query: "Update my_table(arg1, arg2) SET k = 'value'",
			want:  sqlparser.StmtUpdate,
		},
		{
			query: "DELETE FROM my_table(arg1, arg2) WHERE k = 'value'",
			want:  sqlparser.StmtDelete,
		},
		{
			query: "DELETE FROM my_table(arg1, arg2) WHERE (k1, k2) IN (SELECT k1, k2 FROM my_table2)",
			want:  sqlparser.StmtDelete,
		},
		{
			query: "SELECT * FROM my_table(arg1, arg2)",
			want:  sqlparser.StmtSelect,
		},
		{
			query: "INSERT INTO my_table(arg1, arg2) VALUES (1, 'value')",
			want:  sqlparser.StmtInsert,
		},
		// Placeholder arguments
		{
			query: "SELECT * FROM my_table WHERE k = ?",
			want:  sqlparser.StmtSelect,
		},
		{
			query: "SELECT * FROM my_table WHERE k = @1",
			want:  sqlparser.StmtSelect,
		},
		{
			query: "SELECT * FROM my_table WHERE k = :1",
			want:  sqlparser.StmtSelect,
		},
	}

	for _, tt := range tests {

		queryType, _, err := GetQueryType(tt.query)
		if err != nil {
			t.Fatalf("unexpected error while parsing: %v", err)
		}

		require.Equal(t, tt.want, queryType, "unexpected query type for %s", tt.query)
	}
}
