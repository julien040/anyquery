package controller

import (
	"os"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// sqlPalette holds the SGR sequence written before each token kind.
// An empty string leaves the token in the terminal's default color.
type sqlPalette struct {
	keyword, str, number, function, operator, comment, quoted, command string
}

// Atom One Dark colors, as 24-bit SGR sequences
var oneDarkPalette = sqlPalette{
	keyword:  "\x1b[38;2;198;120;221m", // #C678DD
	str:      "\x1b[38;2;152;195;121m", // #98C379
	number:   "\x1b[38;2;209;154;102m", // #D19A66
	function: "\x1b[38;2;97;175;239m",  // #61AFEF
	operator: "\x1b[38;2;86;182;194m",  // #56B6C2
	comment:  "\x1b[3;38;2;92;99;112m", // #5C6370, italic
	quoted:   "\x1b[38;2;224;108;117m", // #E06C75
	command:  "\x1b[38;2;229;192;123m", // #E5C07B
}

// The 16 base colors adapt to the terminal theme, so they read well on light and dark backgrounds
var ansiPalette = sqlPalette{
	keyword:  "\x1b[35m",
	str:      "\x1b[32m",
	number:   "\x1b[33m",
	function: "\x1b[34m",
	operator: "\x1b[36m",
	comment:  "\x1b[90m",
	quoted:   "\x1b[31m",
	command:  "\x1b[33m",
}

const sgrReset = "\x1b[0m"

// newSQLHighlighter picks a palette for the current terminal, or returns nil to disable highlighting.
// It must run before readline switches the terminal to raw mode, because
// HasDarkBackground queries the terminal (and caches the answer for the process).
func newSQLHighlighter() func([]rune) string {
	if os.Getenv("NO_COLOR") != "" || !isSTDoutAtty() {
		return nil
	}
	palette := ansiPalette
	if lipgloss.ColorProfile() == termenv.TrueColor && lipgloss.HasDarkBackground() {
		palette = oneDarkPalette
	}
	return palette.highlight
}

// highlight returns line with SGR sequences around each token.
// Every input rune is kept as is, so the input may be incomplete (e.g. an unterminated string).
func (p sqlPalette) highlight(line []rune) string {
	var b strings.Builder
	write := func(color string, text []rune) {
		if color == "" {
			b.WriteString(string(text))
			return
		}
		b.WriteString(color)
		b.WriteString(string(text))
		b.WriteString(sgrReset)
	}

	// Dot and slash commands: only the command name is colored
	start := 0
	for start < len(line) && unicode.IsSpace(line[start]) {
		start++
	}
	if start < len(line) && (line[start] == '.' || line[start] == '\\') {
		end := start
		for end < len(line) && !unicode.IsSpace(line[end]) {
			end++
		}
		write("", line[:start])
		write(p.command, line[start:end])
		write("", line[end:])
		return b.String()
	}

	n := len(line)
	at := func(i int) rune {
		if i < n {
			return line[i]
		}
		return 0
	}

	for i := 0; i < n; {
		r, next := line[i], at(i+1)
		start, color := i, ""
		switch {
		case unicode.IsSpace(r):
			for i < n && unicode.IsSpace(line[i]) {
				i++
			}
		case r == '\'':
			i, color = scanQuoted(line, i, '\''), p.str
		case r == '"' || r == '`':
			i, color = scanQuoted(line, i, r), p.quoted
		case r == '[':
			i, color = scanQuoted(line, i, ']'), p.quoted
		case r == '-' && next == '-':
			for i < n && line[i] != '\n' {
				i++
			}
			color = p.comment
		case r == '/' && next == '*':
			i += 2
			for i < n && !(line[i] == '*' && at(i+1) == '/') {
				i++
			}
			i, color = min(i+2, n), p.comment
		case unicode.IsDigit(r) || (r == '.' && unicode.IsDigit(next)):
			// Also consumes hex digits, the x of 0x and exponents
			for i < n && (isWordRune(line[i]) || line[i] == '.') {
				i++
			}
			color = p.number
		case r == '?' || ((r == ':' || r == '@' || r == '$') && isWordRune(next)):
			i++
			for i < n && isWordRune(line[i]) {
				i++
			}
			color = p.number
		case isWordRune(r):
			for i < n && (isWordRune(line[i]) || line[i] == '$') {
				i++
			}
			j := i
			for j < n && unicode.IsSpace(line[j]) {
				j++
			}
			word := strings.ToUpper(string(line[start:i]))
			_, isKeyword := sqliteKeywords[word]
			switch {
			case at(j) == '(' && (!isKeyword || word == "REPLACE" || word == "LIKE" || word == "GLOB"):
				// replace(), like() and glob() are both keywords and functions
				color = p.function
			case isKeyword:
				color = p.keyword
			}
		default:
			i++
			if strings.ContainsRune("+-*/%<>=!|&~^", r) {
				color = p.operator
			}
		}
		write(color, line[start:i])
	}
	return b.String()
}

// scanQuoted returns the index after the closing quote of the token starting at line[i],
// or len(line) if it is unterminated. A doubled closing quote is an escape.
func scanQuoted(line []rune, i int, closing rune) int {
	for i++; i < len(line); i++ {
		if line[i] != closing {
			continue
		}
		if closing != ']' && i+1 < len(line) && line[i+1] == closing {
			i++
			continue
		}
		return i + 1
	}
	return len(line)
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// sqliteKeywords is the list returned by sqlite3_keyword_name
var sqliteKeywords = func() map[string]struct{} {
	m := map[string]struct{}{}
	for _, k := range strings.Fields(`ABORT ACTION ADD AFTER ALL ALTER ALWAYS ANALYZE AND AS ASC
		ATTACH AUTOINCREMENT BEFORE BEGIN BETWEEN BY CASCADE CASE CAST CHECK COLLATE COLUMN
		COMMIT CONFLICT CONSTRAINT CREATE CROSS CURRENT CURRENT_DATE CURRENT_TIME
		CURRENT_TIMESTAMP DATABASE DEFAULT DEFERRABLE DEFERRED DELETE DESC DETACH DISTINCT DO
		DROP EACH ELSE END ESCAPE EXCEPT EXCLUDE EXCLUSIVE EXISTS EXPLAIN FAIL FILTER FIRST
		FOLLOWING FOR FOREIGN FROM FULL GENERATED GLOB GROUP GROUPS HAVING IF IGNORE IMMEDIATE
		IN INDEX INDEXED INITIALLY INNER INSERT INSTEAD INTERSECT INTO IS ISNULL JOIN KEY LAST
		LEFT LIKE LIMIT MATCH MATERIALIZED NATURAL NO NOT NOTHING NOTNULL NULL NULLS OF OFFSET
		ON OR ORDER OTHERS OUTER OVER PARTITION PLAN PRAGMA PRECEDING PRIMARY QUERY RAISE RANGE
		RECURSIVE REFERENCES REGEXP REINDEX RELEASE RENAME REPLACE RESTRICT RETURNING RIGHT
		ROLLBACK ROW ROWS SAVEPOINT SELECT SET TABLE TEMP TEMPORARY THEN TIES TO TRANSACTION
		TRIGGER UNBOUNDED UNION UNIQUE UPDATE USING VACUUM VALUES VIEW VIRTUAL WHEN WHERE
		WINDOW WITH WITHOUT TRUE FALSE`) {
		m[k] = struct{}{}
	}
	return m
}()
