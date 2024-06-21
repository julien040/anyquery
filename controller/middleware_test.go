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
