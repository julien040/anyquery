package controller

import (
	"testing"

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
