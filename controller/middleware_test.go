package controller

import (
	"bytes"
	"testing"

	"github.com/julien040/anyquery/namespace"
	"github.com/stretchr/testify/require"
)

func TestDotParser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		query           string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			query:           ".command arg1 arg2",
			expectedCommand: "command",
			expectedArgs:    []string{"arg1", "arg2"},
		},
		{
			query:           ".command",
			expectedCommand: "command",
			expectedArgs:    []string{},
		},
		{
			query:           ".command arg1",
			expectedCommand: "command",
			expectedArgs:    []string{"arg1"},
		},
	}

	for _, test := range tests {
		command, args := parseDotFunc(test.query)
		require.Equal(t, test.expectedCommand, command)
		require.Equal(t, test.expectedArgs, args)
	}

}

func TestSandboxDeniedDotCommands(t *testing.T) {
	// Listed explicitly so removing one from sandboxDeniedDotCommands fails
	// here. The argument is a path that cannot exist, so a broken guard never
	// runs or writes to something real.
	for _, command := range []string{"shell", "system", "output", "log", "cd", "SHELL"} {
		q := &QueryData{
			SQLQuery: "." + command + " /nonexistent-anyquery-sandbox-test/x",
			Config:   middlewareConfiguration{"dot-command": true, "sandbox": true},
		}
		require.False(t, middlewareDotCommand(q), command)
		require.Equal(t, 2, q.StatusCode, command)
		require.Empty(t, q.Config.GetString("outputFile", ""), command)
	}

	// Harmless dot commands keep working under the sandbox.
	q := &QueryData{
		SQLQuery: ".tables",
		Config:   middlewareConfiguration{"dot-command": true, "sandbox": true},
	}
	middlewareDotCommand(q)
	require.NotEqual(t, 2, q.StatusCode)
	require.NotEmpty(t, q.SQLQuery)
}

func TestTableSuggestion(t *testing.T) {
	ns, err := namespace.NewNamespace(namespace.NamespaceConfig{InMemory: true})
	require.NoError(t, err)
	db, err := ns.Register("")
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec("CREATE TABLE github_my_repositories (id INTEGER)")
	require.NoError(t, err)

	run := func(query string) string {
		var buf bytes.Buffer
		sh := &shell{
			DB:             db,
			Middlewares:    []middleware{middlewareQuery},
			Config:         middlewareConfiguration{"doNotModifyOutput": true},
			OutputFileDesc: &buf,
		}
		sh.Run(query)
		return buf.String()
	}

	// Typo and missing plugin prefix both point at the real table
	require.Contains(t, run("SELECT * FROM github_my_repositores"), "(did you mean: github_my_repositories")
	require.Contains(t, run("SELECT * FROM main.my_repositories"), "(did you mean: github_my_repositories")
	// Exec path (non-SELECT statements)
	require.Contains(t, run("DELETE FROM github_my_repostories"), "(did you mean: github_my_repositories")
	// Nothing close: the plain SQLite error, no suggestion
	out := run("SELECT * FROM zzzzzzzzzzzz")
	require.Contains(t, out, "no such table: zzzzzzzzzzzz")
	require.NotContains(t, out, "did you mean")
}

func TestLevenshtein(t *testing.T) {
	require.Equal(t, 0, levenshtein("abc", "abc"))
	require.Equal(t, 3, levenshtein("", "abc"))
	require.Equal(t, 3, levenshtein("kitten", "sitting"))
}
