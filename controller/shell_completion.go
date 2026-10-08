package controller

import (
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/julien040/anyquery/module"
	"github.com/julien040/anyquery/namespace"
	"github.com/reeflective/readline"
)

// shellCommands lists the dot and slash commands handled by
// middlewareDotCommand, middlewareSlashCommand and shell.Run
var shellCommands = []string{
	".cd", ".csv", ".databases", ".exit", ".help", ".indexes", ".json", ".jsonl",
	".language", ".log", ".maxrows", ".mode", ".output", ".pql", ".print", ".prql",
	".quit", ".read", ".schema", ".separator", ".shell", ".sql", ".tables",
	"\\d", "\\d+", "\\di", "\\dt", "\\dv", "\\l", "\\q",
}

// completionContext describes what the word under the cursor can be
type completionContext struct {
	// One of keywords, tables, expression, dot, call, commands, modes,
	// languages, files or none
	kind string
	// The part of the word already typed, replaced by the completion
	prefix string
	// The qualifier before the dot for "dot", the function or table before
	// the parenthesis for "call"
	name string
}

// completionContextFor guesses from the text before the cursor what to complete.
// It only looks at the previous token, so it is a heuristic, not a SQL parser.
func completionContextFor(text string) completionContext {
	trimmed := strings.TrimLeftFunc(text, unicode.IsSpace)
	if strings.HasPrefix(trimmed, ".") || strings.HasPrefix(trimmed, "\\") {
		cmd, args, hasArgs := strings.Cut(trimmed, " ")
		if !hasArgs {
			return completionContext{kind: "commands", prefix: cmd}
		}
		prefix := args[strings.LastIndexByte(args, ' ')+1:]
		switch strings.ToLower(cmd) {
		case ".mode", ".format":
			return completionContext{kind: "modes", prefix: prefix}
		case ".language", ".languages":
			return completionContext{kind: "languages", prefix: prefix}
		case ".schema", "\\d", "\\d+":
			return completionContext{kind: "tables", prefix: prefix}
		case ".read", ".output", ".cd", ".log":
			return completionContext{kind: "files", prefix: prefix}
		}
		return completionContext{kind: "none"}
	}

	// Find out whether the cursor is in a string, a quoted identifier or a comment
	var quote byte
	quoteStart, lineComment, blockComment := 0, false, false
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case lineComment:
			lineComment = c != '\n'
		case blockComment:
			blockComment = !(c == '/' && text[i-1] == '*')
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '-' && i+1 < len(text) && text[i+1] == '-':
			lineComment = true
		case c == '/' && i+1 < len(text) && text[i+1] == '*':
			blockComment = true
			i++
		case c == '\'' || c == '"' || c == '`':
			quote, quoteStart = c, i
		}
	}
	if lineComment || blockComment || quote == '"' || quote == '`' {
		return completionContext{kind: "none"}
	}
	if quote == '\'' {
		// Paths are completed in read_csv('…') and the other file readers, and in ATTACH '…'
		before := strings.TrimRightFunc(text[:quoteStart], unicode.IsSpace)
		if fn, ok := strings.CutSuffix(before, "("); ok {
			if _, ok := supportedTableFunctions[strings.ToLower(trailingWord(strings.TrimRightFunc(fn, unicode.IsSpace)))]; ok {
				return completionContext{kind: "files", prefix: text[quoteStart+1:]}
			}
		}
		if word := strings.ToUpper(trailingWord(before)); word == "ATTACH" || word == "DATABASE" {
			return completionContext{kind: "files", prefix: text[quoteStart+1:]}
		}
		return completionContext{kind: "none"}
	}

	prefix := trailingWord(text)
	before := text[:len(text)-len(prefix)]
	if qualifier, ok := strings.CutSuffix(before, "."); ok {
		if name := trailingWord(qualifier); name != "" {
			return completionContext{kind: "dot", prefix: prefix, name: name}
		}
		return completionContext{kind: "none"}
	}
	before = strings.TrimRightFunc(before, unicode.IsSpace)
	if before == "" || strings.HasSuffix(before, ";") {
		return completionContext{kind: "keywords", prefix: prefix}
	}
	if fn, ok := strings.CutSuffix(before, "("); ok {
		return completionContext{kind: "call", prefix: prefix, name: trailingWord(strings.TrimRightFunc(fn, unicode.IsSpace))}
	}
	switch strings.ToUpper(trailingWord(before)) {
	case "FROM", "JOIN", "INTO", "UPDATE", "TABLE":
		return completionContext{kind: "tables", prefix: prefix}
	}
	return completionContext{kind: "expression", prefix: prefix}
}

// trailingWord returns the identifier at the end of s
func trailingWord(s string) string {
	i := len(s)
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if !isWordRune(r) {
			break
		}
		i -= size
	}
	return s[i:]
}

// sqlTokens splits text into words, keeping their dots (schema.table), and
// single punctuation characters. Strings, quoted identifiers and comments are skipped.
func sqlTokens(text string) []string {
	var tokens []string
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		switch {
		case r == '\'' || r == '"' || r == '`':
			end := strings.IndexRune(text[i+1:], r)
			if end < 0 {
				return tokens
			}
			i += end + 2
		case strings.HasPrefix(text[i:], "--"):
			end := strings.IndexByte(text[i:], '\n')
			if end < 0 {
				return tokens
			}
			i += end + 1
		case strings.HasPrefix(text[i:], "/*"):
			end := strings.Index(text[i+2:], "*/")
			if end < 0 {
				return tokens
			}
			i += end + 4
		case isWordRune(r):
			j := i
			for j < len(text) {
				r, size := utf8.DecodeRuneInString(text[j:])
				if !isWordRune(r) && r != '.' {
					break
				}
				j += size
			}
			tokens = append(tokens, text[i:j])
			i = j
		case unicode.IsSpace(r):
			i += size
		default:
			tokens = append(tokens, string(r))
			i += size
		}
	}
	return tokens
}

// tableRef is a table named in a query, with its alias if it has one
type tableRef struct{ table, alias string }

// referencedTables returns the tables named after FROM, JOIN, UPDATE and INTO,
// including comma lists and aliases (FROM a x, b AS y), so that only their
// columns are loaded, and the names defined by WITH name AS (…).
// It scans tokens rather than parsing SQL, so a subquery used as a table
// gets no alias and the columns of a CTE are unknown.
func referencedTables(text string) (refs []tableRef, ctes []string) {
	tokens := sqlTokens(text)
	at := func(i int) string {
		if i < len(tokens) {
			return tokens[i]
		}
		return ""
	}
	isName := func(tok string) bool {
		r, _ := utf8.DecodeRuneInString(tok)
		_, isKeyword := sqliteKeywords[strings.ToUpper(tok)]
		return tok != "" && isWordRune(r) && !isKeyword
	}
	// skipParens returns the index after the parenthesis group opening at i
	skipParens := func(i int) int {
		for depth := 0; i < len(tokens); i++ {
			switch tokens[i] {
			case "(":
				depth++
			case ")":
				if depth--; depth == 0 {
					return i + 1
				}
			}
		}
		return i
	}
	for i := range tokens {
		switch strings.ToUpper(tokens[i]) {
		case "FROM", "JOIN", "UPDATE", "INTO":
			for j := i + 1; isName(at(j)); j++ {
				ref := tableRef{table: at(j)}
				// Arguments of a table call, or the column list of INSERT INTO t (a, b)
				if j++; at(j) == "(" {
					j = skipParens(j)
				}
				if strings.EqualFold(at(j), "AS") {
					j++
				}
				if isName(at(j)) {
					ref.alias = at(j)
					j++
				}
				refs = append(refs, ref)
				if at(j) != "," {
					break
				}
			}
		case "WITH":
			j := i + 1
			if strings.EqualFold(at(j), "RECURSIVE") {
				j++
			}
			for isName(at(j)) {
				name := at(j)
				// Optional column list: WITH name(a, b) AS (…)
				if j++; at(j) == "(" {
					j = skipParens(j)
				}
				if !strings.EqualFold(at(j), "AS") {
					break
				}
				for j++; strings.EqualFold(at(j), "NOT") || strings.EqualFold(at(j), "MATERIALIZED"); j++ {
				}
				ctes = append(ctes, name)
				if at(j) != "(" {
					break
				}
				if j = skipParens(j); at(j) != "," {
					break
				}
				j++
			}
		}
	}
	return refs, ctes
}

type tableColumn struct {
	name, typ string
	// hidden columns of virtual tables are the table parameters
	hidden bool
	// Set for the columns of anyquery plugin tables, from the plugin schema
	description string
	required    bool
}

// shortDescription keeps the first line of a description, cut to fit the menu
func shortDescription(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 60 {
		return string(r[:59]) + "…"
	}
	return s
}

// paramLabel is the name of a parameter in the hint. Required parameters end
// with "*", the usual marker of required form fields, rather than wrapping the
// optional ones in brackets: plugins that don't flag their required parameters
// then show plain names instead of claiming every parameter is optional.
func paramLabel(col tableColumn) string {
	if col.required {
		return col.name + "*"
	}
	return col.name
}

// completionCache keeps the schema between two completions.
// It is reset after any statement that could change the schema.
type completionCache struct {
	loaded    bool
	tables    []string
	schemas   []string
	functions []string
	columns   map[string][]tableColumn
	// virtual holds the lowercase names of modules and virtual tables,
	// the only tables that can have parameters
	virtual map[string]bool
	// plugins maps the lowercase names of anyquery plugin tables to their
	// metadata, read from the manifests without starting the plugins
	plugins map[string]namespace.TableMetadata
}

// queryStrings returns the first column of every row, or nil on error
func queryStrings(db *sql.DB, query string, args ...any) []string {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var s string
		if rows.Scan(&s) == nil {
			result = append(result, s)
		}
	}
	return result
}

// load reads the schema once. ns may be nil, plugin tables then have no description.
func (c *completionCache) load(db *sql.DB, ns *namespace.Namespace) {
	if c.loaded {
		return
	}
	c.loaded = true
	c.columns = map[string][]tableColumn{}
	c.virtual = map[string]bool{}
	c.plugins = map[string]namespace.TableMetadata{}
	if ns != nil {
		for _, table := range ns.ListPluginsTables() {
			c.plugins[strings.ToLower(table.Name)] = table
		}
	}
	for _, name := range queryStrings(db,
		"SELECT name FROM pragma_module_list UNION SELECT name FROM pragma_table_list WHERE type = 'virtual'") {
		c.virtual[strings.ToLower(name)] = true
	}
	c.tables = queryStrings(db, tableListQuery)
	// Tables of attached databases are also reachable as schema.table
	c.schemas = queryStrings(db, "SELECT name FROM pragma_database_list WHERE name NOT IN ('main', 'temp')")
	c.tables = append(c.tables, queryStrings(db,
		"SELECT schema || '.' || name FROM pragma_table_list WHERE schema NOT IN ('main', 'temp')")...)
	for _, name := range queryStrings(db, "SELECT DISTINCT name FROM pragma_function_list") {
		// Skip operators such as -> and ->>
		if trailingWord(name) == name {
			c.functions = append(c.functions, name)
		}
	}
}

// columnsOf loads the columns of a table once. Failures are cached too,
// because loading the columns of a plugin table starts the plugin.
// For plugin tables, the descriptions come from the schema the plugin
// returned when pragma_table_xinfo connected the table.
func (c *completionCache) columnsOf(db *sql.DB, ns *namespace.Namespace, table string) []tableColumn {
	key := strings.ToLower(table)
	if cols, ok := c.columns[key]; ok {
		return cols
	}
	query := "SELECT name, type, hidden FROM pragma_table_xinfo(?)"
	args := []any{table}
	if schema, name, ok := strings.Cut(table, "."); ok {
		query = "SELECT name, type, hidden FROM pragma_table_xinfo(?, ?)"
		args = []any{name, schema}
	}
	var cols []tableColumn
	if rows, err := db.Query(query, args...); err == nil {
		for rows.Next() {
			var col tableColumn
			var hidden int
			if rows.Scan(&col.name, &col.typ, &hidden) == nil {
				col.hidden = hidden == 1
				cols = append(cols, col)
			}
		}
		rows.Close()
	}
	if meta, ok := c.plugins[key]; ok && ns != nil {
		if desc, err := ns.DescribeTable(meta.Name); err == nil {
			for i := range cols {
				for _, col := range desc.Columns {
					if strings.EqualFold(col.Name, cols[i].name) {
						cols[i].description = shortDescription(col.Description)
						cols[i].required = col.IsRequired
					}
				}
			}
		}
	}
	c.columns[key] = cols
	return cols
}

// complete is the readline completer of the shell
func (p *shell) complete(line []rune, cursor int) readline.Completions {
	text := string(line[:cursor])
	ctx := completionContextFor(text)
	if ctx.kind == "none" || (p.DB == nil && ctx.kind != "files") {
		return readline.Completions{}
	}

	var cands []readline.Completion
	add := func(tag, value, display, description string) {
		cands = append(cands, readline.Completion{Tag: tag, Value: value, Display: display, Description: description})
	}
	refs, ctes := referencedTables(string(line))
	addColumns := func(table string) {
		for _, col := range p.completion.columnsOf(p.DB, p.Namespace, table) {
			description := col.description
			if description == "" {
				description = col.typ
			}
			if col.hidden {
				add("parameters", col.name, col.name, description)
			} else {
				add("columns", col.name, col.name, description)
			}
		}
	}
	tableDescription := func(table string) string {
		return shortDescription(p.completion.plugins[strings.ToLower(table)].Description)
	}
	addTables := func() {
		for _, cte := range ctes {
			add("tables", cte, cte, "CTE")
		}
		for _, t := range p.completion.tables {
			add("tables", t, t, tableDescription(t))
		}
		for fn := range supportedTableFunctions {
			add("functions", fn+"(", fn, "")
		}
	}
	addExpression := func() {
		seen := map[string]bool{}
		for _, ref := range refs {
			if table := strings.ToLower(ref.table); !seen[table] {
				seen[table] = true
				addColumns(ref.table)
			}
		}
		for _, fn := range p.completion.functions {
			add("functions", fn+"(", fn, "")
		}
		for kw := range sqliteKeywords {
			add("keywords", kw, kw, "")
		}
	}

	if p.DB != nil {
		p.completion.load(p.DB, p.Namespace)
	}
	// Tags are listed in the order of their first candidate, so the snippet comes first
	if typed, ok := selectSnippetTyped(text); ok {
		add("snippets", selectSnippet[len(typed)-len(ctx.prefix):], selectSnippet, "")
	}
	switch ctx.kind {
	case "keywords":
		for kw := range sqliteKeywords {
			add("keywords", kw, kw, "")
		}
	case "tables":
		addTables()
	case "dot":
		if slices.ContainsFunc(p.completion.schemas, func(s string) bool { return strings.EqualFold(s, ctx.name) }) {
			for _, t := range p.completion.tables {
				if schema, name, ok := strings.Cut(t, "."); ok && strings.EqualFold(schema, ctx.name) {
					add("tables", name, name, "")
				}
			}
		} else {
			table := ctx.name
			for _, ref := range refs {
				if strings.EqualFold(ref.alias, ctx.name) {
					table = ref.table
					break
				}
			}
			addColumns(table)
		}
	case "call", "expression":
		// Table parameters inside tbl( are positional, so paramsHint shows
		// them below the prompt rather than the menu inserting them
		addExpression()
	case "commands":
		for _, cmd := range shellCommands {
			if p.Config.GetBool("sandbox", false) && sandboxDeniedDotCommands[cmd[1:]] {
				continue
			}
			add("commands", cmd, cmd, "")
		}
	case "modes":
		for mode := range formatName {
			add("modes", mode, mode, "")
		}
	case "languages":
		for _, lang := range []string{"sql", "prql", "pql"} {
			add("languages", lang, lang, "")
		}
	case "files":
		cands = completeFiles(ctx.prefix, p.Restrictions)
	}

	comps := readline.CompleteRaw(cands)
	if ctx.prefix != "" {
		comps.PREFIX = ctx.prefix
	} else {
		// readline falls back to the whitespace-delimited word before the
		// cursor (e.g. "count(" or "users."). Keep that text in front of
		// every value so that only the candidate gets inserted.
		i := strings.LastIndexFunc(text, unicode.IsSpace) + 1
		comps = comps.Prefix(text[i:])
	}
	return comps
}

const selectSnippet = "SELECT * FROM "

// selectSnippetTyped returns the start of the current statement when it is
// a start of selectSnippet that the menu can offer: "sel", "SELECT" or "SELECT "
func selectSnippetTyped(text string) (string, bool) {
	typed := strings.TrimLeftFunc(text[strings.LastIndexByte(text, ';')+1:], unicode.IsSpace)
	upper := strings.ToUpper(typed)
	return typed, typed != "" && (strings.HasPrefix("SELECT", upper) || upper == "SELECT ")
}

// paramsHint returns the parameters of the table being called, e.g.
// "generate_series(start, stop, step)", or "" when there is none.
// It applies right after a table name following FROM or JOIN, and inside
// the parenthesis of a table call, where the current parameter is in bold.
// params returns the parameter names of a table, or nil.
func paramsHint(text string, params func(table string) []string) string {
	// The open parenthesis before the cursor, and the commas at their depth
	type call struct{ pos, commas int }
	var calls []call
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"' || c == '`':
			quote = c
		case c == '(':
			calls = append(calls, call{pos: i})
		case c == ')' && len(calls) > 0:
			calls = calls[:len(calls)-1]
		case c == ',' && len(calls) > 0:
			calls[len(calls)-1].commas++
		}
	}

	// Inside nested calls such as tbl(lower(, the innermost table wins
	for i := len(calls) - 1; i >= 0; i-- {
		name := trailingWord(strings.TrimRightFunc(text[:calls[i].pos], unicode.IsSpace))
		if names := params(name); len(names) > 0 {
			return formatParamsHint(name, names, calls[i].commas)
		}
	}
	if len(calls) == 0 && quote == 0 {
		if ctx := completionContextFor(text); ctx.kind == "tables" {
			if names := params(ctx.prefix); len(names) > 0 {
				return formatParamsHint(ctx.prefix, names, -1)
			}
		}
	}
	return ""
}

// formatParamsHint renders the hint in grey, with the parameter at index
// current in bold (none when current is out of range)
func formatParamsHint(table string, names []string, current int) string {
	const grey, bold, notBold, reset = "\x1b[38;5;245m", "\x1b[1m", "\x1b[22m", "\x1b[0m"
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = name
		if i == current {
			parts[i] = bold + name + notBold
		}
	}
	return grey + table + "(" + strings.Join(parts, ", ") + ")" + reset
}

// tableParams returns the parameters (hidden columns) of a table.
// Only virtual tables have parameters, so other tables are never looked up.
// The lookup runs on the readline goroutine and blocks once per table, the
// first time it is named: for a plugin table, this starts the plugin, as
// running the query would. It can't run in the background because a second
// connection to the in-memory or temp database would see an empty schema.
func (p *shell) tableParams(table string) []string {
	if p.DB == nil || table == "" {
		return nil
	}
	p.completion.load(p.DB, p.Namespace)
	if !p.completion.virtual[strings.ToLower(table)] {
		return nil
	}
	var names []string
	for _, col := range p.completion.columnsOf(p.DB, p.Namespace, table) {
		if col.hidden {
			names = append(names, paramLabel(col))
		}
	}
	return names
}

// completeFiles lists the entries of the directory typed in prefix.
// Hidden files are only listed once the user types the leading dot.
// Under a sandbox, only the entries it allows to read are listed.
//
// `~` is not expanded: neither .read nor the read_* functions expand it,
// so a completed `~/` path could not be opened anyway.
func completeFiles(prefix string, restrictions *module.Restrictions) []readline.Completion {
	dir := prefix[:strings.LastIndexAny(prefix, "/"+string(os.PathSeparator))+1]
	base := prefix[len(dir):]
	readDir := dir
	if readDir == "" {
		readDir = "."
	}
	if restrictions.CheckFileRead(readDir) != nil {
		return nil
	}
	entries, _ := os.ReadDir(readDir)
	var cands []readline.Completion
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".") {
			continue
		}
		if restrictions.CheckFileRead(filepath.Join(readDir, name)) != nil {
			continue
		}
		if entry.IsDir() {
			name += "/"
		}
		cands = append(cands, readline.Completion{Tag: "files", Value: dir + name, Display: name})
	}
	return cands
}
