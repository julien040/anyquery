package controller

import (
	"regexp"
	"strings"
	"testing"
)

var sgrRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestHighlightKeepsInput(t *testing.T) {
	inputs := []string{
		"",
		"SELECT 1;",
		"SELECT 'unterminated",
		"SELECT 'it''s' AS x;",
		"SELECT * /* unterminated",
		"SELECT a, -- comment\n  b FROM t\nWHERE x = 0x1F AND y >= 1.5e3;",
		`SELECT "naïve", [col], ` + "`日本` FROM tâble;",
		"SELECT ?, ?1, :name, @var, $param;",
		"  .tables arg",
		`\dt users`,
		"x - -1 / 2 * .5",
	}
	for _, palette := range []sqlPalette{oneDarkPalette, ansiPalette} {
		for _, input := range inputs {
			got := sgrRegex.ReplaceAllString(palette.highlight([]rune(input)), "")
			if got != input {
				t.Errorf("highlight(%q) stripped = %q", input, got)
			}
		}
	}
}

func TestHighlightColors(t *testing.T) {
	p := oneDarkPalette
	tests := []struct {
		input, token, color string
	}{
		{"select 1", "select", p.keyword},
		{"SELECT 'x'", "'x'", p.str},
		{"SELECT 42", "42", p.number},
		{"SELECT count(*)", "count", p.function},
		{"SELECT replace(a, 'b', 'c')", "replace", p.function},
		{"SELECT 1 -- hi", "-- hi", p.comment},
		{`SELECT "col"`, `"col"`, p.quoted},
		{"SELECT a = b", "=", p.operator},
		{".tables x", ".tables", p.command},
	}
	for _, tt := range tests {
		want := tt.color + tt.token + sgrReset
		if got := p.highlight([]rune(tt.input)); !strings.Contains(got, want) {
			t.Errorf("highlight(%q) = %q, want it to contain %q", tt.input, got, want)
		}
	}
	// Identifiers keep the terminal's default color
	if got := p.highlight([]rune("users")); got != "users" {
		t.Errorf("highlight(users) = %q", got)
	}
}
