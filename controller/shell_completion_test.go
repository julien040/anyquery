package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julien040/anyquery/module"
	"github.com/julien040/anyquery/namespace"
	"github.com/reeflective/readline"
	"github.com/stretchr/testify/require"
)

func TestCompletionContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text string
		want completionContext
	}{
		{"", completionContext{kind: "keywords"}},
		{"SEL", completionContext{kind: "keywords", prefix: "SEL"}},
		{"SELECT 1; INS", completionContext{kind: "keywords", prefix: "INS"}},
		{"SELECT * FROM us", completionContext{kind: "tables", prefix: "us"}},
		{"SELECT * FROM a JOIN ", completionContext{kind: "tables"}},
		{"SELECT na", completionContext{kind: "expression", prefix: "na"}},
		{"SELECT id,na", completionContext{kind: "expression", prefix: "na"}},
		{"SELECT users.na", completionContext{kind: "dot", prefix: "na", name: "users"}},
		{"SELECT * FROM db.", completionContext{kind: "dot", name: "db"}},
		{"SELECT count(", completionContext{kind: "call", name: "count"}},
		{"SELECT * FROM github_repos(us", completionContext{kind: "call", prefix: "us", name: "github_repos"}},
		{"SELECT * FROM read_csv('data/fi", completionContext{kind: "files", prefix: "data/fi"}},
		{"ATTACH 'my ", completionContext{kind: "files", prefix: "my "}},
		{"SELECT 'abc", completionContext{kind: "none"}},
		{"SELECT \"col", completionContext{kind: "none"}},
		{"SELECT 1 -- FROM ", completionContext{kind: "none"}},
		{"SELECT 'a' FROM t WHERE id", completionContext{kind: "expression", prefix: "id"}},
		{".mo", completionContext{kind: "commands", prefix: ".mo"}},
		{".mode js", completionContext{kind: "modes", prefix: "js"}},
		{".read ~/q", completionContext{kind: "files", prefix: "~/q"}},
		{"\\d ", completionContext{kind: "tables"}},
		{".print hello", completionContext{kind: "none"}},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, completionContextFor(tt.text), tt.text)
	}
}

// values returns the inserted values of the completions per tag.
// The cursor is at the "|" in line, or at the end if there is none.
func values(t *testing.T, sh *shell, line string) map[string][]string {
	before, after, _ := strings.Cut(line, "|")
	comps := sh.complete([]rune(before+after), len([]rune(before)))
	result := map[string][]string{}
	comps.EachValue(func(c readline.Completion) readline.Completion {
		result[c.Tag] = append(result[c.Tag], c.Value)
		return c
	})
	return result
}

func TestCompleter(t *testing.T) {
	ns, err := namespace.NewNamespace(namespace.NamespaceConfig{InMemory: true})
	require.NoError(t, err)
	db, err := ns.Register("")
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec("CREATE TABLE users (id INTEGER, name TEXT); CREATE TABLE orders (total REAL)")
	require.NoError(t, err)

	sh := &shell{DB: db}
	require.Contains(t, values(t, sh, "SELECT * FROM us")["tables"], "users")
	require.Contains(t, values(t, sh, "SELECT * FROM us")["functions"], "read_csv(")
	require.ElementsMatch(t, []string{"id", "name"}, values(t, sh, "SELECT | FROM users")["columns"])
	require.ElementsMatch(t, []string{"users.id", "users.name"}, values(t, sh, "SELECT users.")["columns"])
	require.Contains(t, values(t, sh, "SELECT count(")["functions"], "count(lower(")
	require.Contains(t, values(t, sh, ".mode ")["modes"], "json")
	require.Equal(t, []string{"SELECT * FROM "}, values(t, sh, "sel")["snippets"])
	require.Equal(t, []string{"* FROM "}, values(t, sh, "SELECT ")["snippets"])
	require.Equal(t, []string{"json", "root"}, sh.tableParams("json_each"))
	require.Nil(t, sh.tableParams("users"))

	// Aliases, comma lists and CTE names
	require.ElementsMatch(t, []string{"u.id", "u.name"}, values(t, sh, "SELECT u.| FROM users u")["columns"])
	require.ElementsMatch(t, []string{"id", "name", "total"}, values(t, sh, "SELECT | FROM users u, orders AS o")["columns"])
	require.Contains(t, values(t, sh, "WITH recent AS (SELECT 1) SELECT * FROM re")["tables"], "recent")

	// Without a namespace, columns are described by their type
	comps := sh.complete([]rune("SELECT na FROM users"), len("SELECT na"))
	comps.EachValue(func(c readline.Completion) readline.Completion {
		if c.Value == "name" {
			require.Equal(t, "TEXT", c.Description)
		}
		return c
	})

	// A new table shows up once the cache is reset after a non-SELECT statement
	_, err = db.Exec("CREATE TABLE products (sku TEXT)")
	require.NoError(t, err)
	require.NotContains(t, values(t, sh, "SELECT * FROM ")["tables"], "products")
	sh.completion = completionCache{}
	require.Contains(t, values(t, sh, "SELECT * FROM ")["tables"], "products")
}

func TestReferencedTables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text string
		refs []tableRef
		ctes []string
	}{
		{"SELECT * FROM users", []tableRef{{table: "users"}}, nil},
		{"SELECT * FROM users u WHERE u.id = 1", []tableRef{{"users", "u"}}, nil},
		{"SELECT * FROM users AS u", []tableRef{{"users", "u"}}, nil},
		{"SELECT * FROM a x, b AS y, c WHERE", []tableRef{{"a", "x"}, {"b", "y"}, {table: "c"}}, nil},
		{"SELECT * FROM users WHERE id = 1", []tableRef{{table: "users"}}, nil},
		{"SELECT * FROM a LEFT JOIN db.t AS z ON a.id = z.id", []tableRef{{table: "a"}, {"db.t", "z"}}, nil},
		{"SELECT * FROM a JOIN b USING (id) ORDER BY 1", []tableRef{{table: "a"}, {table: "b"}}, nil},
		{"SELECT * FROM json_each('[1, 2]') AS j", []tableRef{{"json_each", "j"}}, nil},
		{"SELECT *\nFROM users\n  u\nJOIN orders o\nON 1", []tableRef{{"users", "u"}, {"orders", "o"}}, nil},
		{"SELECT 'FROM x' FROM users -- FROM y", []tableRef{{table: "users"}}, nil},
		{"UPDATE t SET a = 1", []tableRef{{table: "t"}}, nil},
		{"INSERT INTO t (a, b) VALUES (1, 2)", []tableRef{{table: "t"}}, nil},
		{"WITH x AS (SELECT 1), y(a) AS MATERIALIZED (SELECT 2) SELECT * FROM x",
			[]tableRef{{table: "x"}}, []string{"x", "y"}},
		{"WITH RECURSIVE r AS (SELECT 1 FROM t) SELECT", []tableRef{{table: "t"}}, []string{"r"}},
	}
	for _, tt := range tests {
		refs, ctes := referencedTables(tt.text)
		require.Equal(t, tt.refs, refs, tt.text)
		require.Equal(t, tt.ctes, ctes, tt.text)
	}
}

func TestDescriptions(t *testing.T) {
	t.Parallel()

	require.Equal(t, "List your repositories", shortDescription("  List your repositories\nMore details  "))
	long := strings.Repeat("é", 70)
	require.Equal(t, strings.Repeat("é", 59)+"…", shortDescription(long))

	require.Equal(t, "user*", paramLabel(tableColumn{name: "user", required: true}))
	require.Equal(t, "page", paramLabel(tableColumn{name: "page"}))
	require.Equal(t, "\x1b[38;5;245mrepos(\x1b[1muser*\x1b[22m, page)\x1b[0m",
		formatParamsHint("repos", []string{"user*", "page"}, 0))
}

func TestCompleteFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.csv"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), nil, 0o644))

	list := func(prefix string, restrictions *module.Restrictions) []string {
		var result []string
		for _, c := range completeFiles(prefix, restrictions) {
			result = append(result, c.Value)
		}
		return result
	}
	require.ElementsMatch(t, []string{dir + "/data.csv", dir + "/sub/"}, list(dir+"/", nil))
	require.Contains(t, list(dir+"/.", nil), dir+"/.hidden")

	// Under a sandbox, a directory outside the allowed ones lists nothing,
	// and a symlink escaping an allowed directory is not listed
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.csv"), nil, 0o644))
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret.csv"), filepath.Join(dir, "link.csv")))
	sandbox := &module.Restrictions{AllowedDirs: []string{dir}}
	require.Empty(t, list(outside+"/", sandbox))
	require.ElementsMatch(t, []string{dir + "/data.csv", dir + "/sub/"}, list(dir+"/", sandbox))
	require.Contains(t, list(dir+"/", nil), dir+"/link.csv")
}

func TestCompleteCommandsSandbox(t *testing.T) {
	ns, err := namespace.NewNamespace(namespace.NamespaceConfig{InMemory: true})
	require.NoError(t, err)
	db, err := ns.Register("")
	require.NoError(t, err)
	defer db.Close()

	sh := &shell{DB: db, Config: middlewareConfiguration{}}
	require.Contains(t, values(t, sh, ".s")["commands"], ".shell")

	sh.Config.SetBool("sandbox", true)
	cmds := values(t, sh, ".")["commands"]
	require.Contains(t, cmds, ".read")
	for _, denied := range []string{".shell", ".system", ".output", ".log", ".cd"} {
		require.NotContains(t, cmds, denied)
	}
}

func TestParamsHint(t *testing.T) {
	t.Parallel()

	params := func(table string) []string {
		if strings.EqualFold(table, "json_each") {
			return []string{"json", "root"}
		}
		return nil
	}
	const grey, bold, notBold, reset = "\x1b[38;5;245m", "\x1b[1m", "\x1b[22m", "\x1b[0m"
	tests := []struct{ text, want string }{
		{"SELECT * FROM json_each", grey + "json_each(json, root)" + reset},
		{"SELECT * FROM json_each(", grey + "json_each(" + bold + "json" + notBold + ", root)" + reset},
		{"SELECT * FROM json_each('[1]', ", grey + "json_each(json, " + bold + "root" + notBold + ")" + reset},
		{"SELECT * FROM json_each('[1,2]', '$.a,b", grey + "json_each(json, " + bold + "root" + notBold + ")" + reset},
		{"SELECT * FROM json_each(lower('x'), ", grey + "json_each(json, " + bold + "root" + notBold + ")" + reset},
		{"SELECT * FROM json_each('[1]')", ""},
		{"SELECT * FROM json_each('[1]') WHERE ", ""},
		{"SELECT * FROM users", ""},
		{"SELECT * FROM users(", ""},
		{"SELECT json_each", ""},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, paramsHint(tt.text, params), tt.text)
	}
}

func TestSelectSnippet(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"s", "sel", "SELECT", "SELECT ", "  select ", "SELECT 1; sel"} {
		_, ok := selectSnippetTyped(text)
		require.True(t, ok, text)
	}
	for _, text := range []string{"", "SELECT a", "SELECT  ", "sel ", "INSERT", "SELECT * FROM t WHERE sel"} {
		_, ok := selectSnippetTyped(text)
		require.False(t, ok, text)
	}
}
