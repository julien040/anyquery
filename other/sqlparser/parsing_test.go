package sqlparser_test

import (
	"testing"

	"github.com/julien040/anyquery/other/sqlparser"
)

// Test the parsing of SQLite queries.

var sqlTest = []struct {
	in  string
	out string
}{
	// Aggregate Functions
	{
		in:  "select count(*) from users",
		out: "select count(*) from users",
	},
	{
		in:  "select avg(salary) from employees",
		out: "select avg(salary) from employees",
	},
	// alter table
	{
		in:  "alter table employees add column address text",
		out: "alter table employees add column address text",
	},
	// analyze
	{
		in:  "analyze employees",
		out: "analyze employees",
	},
	// attach database
	{
		in:  "attach database 'test.db' as testdb",
		out: "attach database 'test.db' as testdb",
	},
	// begin transaction
	{
		in:  "begin transaction",
		out: "begin transaction",
	},
	// Comment
	{
		in:  "-- this is a comment",
		out: "-- this is a comment",
	},
	// commit transaction
	{
		in:  "commit",
		out: "commit",
	},
	// Core Functions
	{
		in:  "select abs(-1)",
		out: "select abs(-1)",
	},
	// create index
	{
		in:  "create index idx_name on employees(name)",
		out: "create index idx_name on employees(name)",
	},
	// create table
	{
		in: `create table users (id integer primary key, name text)`,
		out: `create table users (
	id integer primary key,
	name text
)`,
	},
	// create trigger
	{
		in: `create trigger trg_users after insert on users begin
			update users set name = 'default' where name is null;
		end;`,
		out: `create trigger trg_users after insert on users begin
			update users set name = 'default' where name is null;
		end;`,
	},
	// create view
	{
		in:  "create view user_names as select name from users",
		out: "create view user_names as select name from users",
	},
	// create virtual table
	{
		in:  "create virtual table temp.fts using fts5(content)",
		out: "create virtual table temp.fts using fts5(content)",
	},
	// Date and Time Functions
	{
		in:  "select date('now')",
		out: "select date('now')",
	},
	// delete
	{
		in:  "delete from users where id = 1",
		out: "delete from users where id = 1",
	},
	// detach database
	{
		in:  "detach database testdb",
		out: "detach database testdb",
	},
	// drop index
	{
		in:  "drop index idx_name",
		out: "drop index idx_name",
	},
	// drop table
	{
		in:  "drop table users",
		out: "drop table users",
	},
	// drop trigger
	{
		in:  "drop trigger trg_users",
		out: "drop trigger trg_users",
	},
	// drop view
	{
		in:  "drop view user_names",
		out: "drop view user_names",
	},
	// end transaction
	{
		in:  "end",
		out: "end",
	},
	// explain
	{
		in:  "explain query plan select * from users",
		out: "explain query plan select * from users",
	},
	// Expressions
	{
		in:  "select 2 + 2",
		out: "select 2 + 2",
	},
	// indexed by
	{
		in:  "select * from employees indexed by idx_name where name = 'john'",
		out: "select * from employees indexed by idx_name where name = 'john'",
	},
	// insert
	{
		in:  "insert into users (name) values ('alice')",
		out: "insert into users (name) values ('alice')",
	},
	// JSON Functions
	{
		in:  "select json_extract(data, '$.name') from users_json",
		out: "select json_extract(data, '$.name') from users_json",
	},
	// Keywords
	{
		in:  "select * from pragma_table_info('users')",
		out: "select * from pragma_table_info('users')",
	},
	// Math Functions
	{
		in:  "select round(4.5678, 2)",
		out: "select round(4.5678, 2)",
	},
	// on conflict clause
	{
		in:  "insert into users(id, name) values(1, 'alice') on conflict(id) do update set name = 'alice'",
		out: "insert into users(id, name) values(1, 'alice') on conflict(id) do update set name = 'alice'",
	},
	// pragma
	{
		in:  "pragma foreign_keys = on",
		out: "pragma foreign_keys = on",
	},
	// reindex
	{
		in:  "reindex idx_name",
		out: "reindex idx_name",
	},
	// release savepoint
	{
		in:  "release my_savepoint",
		out: "release my_savepoint",
	},
	// replace
	{
		in:  "replace into users (id, name) values (1, 'bob')",
		out: "replace into users (id, name) values (1, 'bob')",
	},
	// Returning Clause
	{
		in:  "delete from users where name = 'alice' returning id",
		out: "delete from users where name = 'alice' returning id",
	},
	// rollback transaction
	{
		in:  "rollback",
		out: "rollback",
	},
	// savepoint
	{
		in:  "savepoint my_savepoint",
		out: "savepoint my_savepoint",
	},
	// select
	{
		in:  "select * from users",
		out: "select * from users",
	},
	// update
	{
		in:  "update users set name = 'bob' where id = 2",
		out: "update users set name = 'bob' where id = 2",
	},
	// upsert (on conflict)
	{
		in:  "insert into users (id, name) values (1, 'charlie') on conflict(id) do update set name = 'charlie'",
		out: "insert into users (id, name) values (1, 'charlie') on conflict(id) do update set name = 'charlie'",
	},
	// vacuum
	{
		in:  "vacuum",
		out: "vacuum",
	},
	// Window Functions
	{
		in:  "select name, rank() over (partition by department order by salary desc) from employees",
		out: "select name, rank() over (partition by department order by salary desc) from employees",
	},
	// with clause
	{
		in:  "with recursive cnt(x) as (select 1 union all select x+1 from cnt where x<10) select x from cnt",
		out: "with recursive cnt(x) as (select 1 union all select x+1 from cnt where x<10) select x from cnt",
	},
	// EXCEPT
	{
		in:  "select id from table1 except select id from table2",
		out: "select id from table1 except select id from table2",
	},
	// CREATE TABLE AS
	{
		in:  "create table new_table as select * from old_table",
		out: "create table new_table as select * from old_table",
	},
	// INSERT SELECT
	{
		in:  "insert into new_table (select * from old_table where condition)",
		out: "insert into new_table (select * from old_table where condition)",
	},
	// LIKE
	{
		in:  "select * from users where name like 'A%'",
		out: "select * from users where name like 'A%'",
	},
	// UPDATE FROM
	{
		in:  "update users set name = (select name from employees where users.id = employees.id)",
		out: "update users set name = (select name from employees where users.id = employees.id)",
	},
	// CASE WHEN THEN ELSE END
	{
		in:  "select case when score >= 90 then 'A' when score >= 80 then 'B' else 'F' end as grade from scores",
		out: "select case when score >= 90 then 'A' when score >= 80 then 'B' else 'F' end as grade from scores",
	},
	// CROS JOIN
	{
		in:  "select * from table1 cross join table2",
		out: "select * from table1 cross join table2",
	},
	// NATURAL JOIN
	{
		in:  "select * from table1 natural join table2",
		out: "select * from table1 natural join table2",
	},
	// INNER JOIN
	{
		in:  "select * from table1 inner join table2 on table1.id = table2.id",
		out: "select * from table1 inner join table2 on table1.id = table2.id",
	},
	// LEFT JOIN
	{
		in:  "select * from table1 left join table2 on table1.id = table2.id",
		out: "select * from table1 left join table2 on table1.id = table2.id",
	},
	// RIGHT JOIN (not supported in SQLite, but included for completeness)
	{
		in:  "select * from table1 right join table2 on table1.id = table2.id",
		out: "select * from table1 right join table2 on table1.id = table2.id",
	},
	// FULL OUTER JOIN (not supported in SQLite, but included for completeness)
	{
		in:  "select * from table1 full outer join table2 on table1.id = table2.id",
		out: "select * from table1 full outer join table2 on table1.id = table2.id",
	},
	// UNION
	{
		in:  "select * from table1 union select * from table2",
		out: "select * from table1 union select * from table2",
	},
	// UNION ALL
	{
		in:  "select * from table1 union all select * from table2",
		out: "select * from table1 union all select * from table2",
	},
	// INTERSECT
	{
		in:  "select id from table1 intersect select id from table2",
		out: "select id from table1 intersect select id from table2",
	},
	// EXCEPT
	{
		in:  "select id from table1 except select id from table2",
		out: "select id from table1 except select id from table2",
	},
	// EXISTS
	{
		in:  "select * from users where exists (select 1 from orders where orders.user_id = users.id)",
		out: "select * from users where exists (select 1 from orders where orders.user_id = users.id)",
	},
	// IN
	{
		in:  "select * from users where id in (select id from active_users)",
		out: "select * from users where id in (select id from active_users)",
	},
	// NOT IN
	{
		in:  "select * from users where id not in (select id from inactive_users)",
		out: "select * from users where id not in (select id from inactive_users)",
	},
	// SUBQUERY
	{
		in:  "select * from users where id = (select max(id) from users)",
		out: "select * from users where id = (select max(id) from users)",
	},
	// LIMIT
	{
		in:  "select * from users limit 10",
		out: "select * from users limit 10",
	},
	// OFFSET
	{
		in:  "select * from users limit 10 offset 5",
		out: "select * from users limit 10 offset 5",
	},
}

func TestSQLiteParsing(t *testing.T) {
	parser, err := sqlparser.New(sqlparser.Options{
		MySQLServerVersion: "8.0.30",
	})
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	for _, tt := range sqlTest {
		stmt, err := parser.Parse(tt.in)
		if err != nil {
			t.Errorf("Error: %v", err)
		}
		if out := sqlparser.String(stmt); out != tt.out {
			t.Errorf("got %v, want %v", out, tt.out)
		}
	}

}
