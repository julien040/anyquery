package namespace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgsParsing(t *testing.T) {
	tests := []struct {
		args    string
		results []string
	}{
		{
			args:    "hello world",
			results: []string{"hello", "world"},
		},
		{
			args:    "hello \"world\"",
			results: []string{"hello", "world"},
		},
		{
			args:    `myarg="hello world"`,
			results: []string{"myarg=hello world"},
		},
		{
			args:    `myarg="hello world" "myarg2=hello world2"`,
			results: []string{"myarg=hello world", "myarg2=hello world2"},
		},
		{
			args:    `--path "C:\Program Files"`,
			results: []string{"--path", "C:\\Program Files"},
		},
		{
			args:    `"./my path/with spaces" arg2`,
			results: []string{"./my path/with spaces", "arg2"},
		},
	}

	for _, test := range tests {
		t.Run(test.args, func(t *testing.T) {
			require.Equal(t, test.results, parseCommands(test.args), "The results should be correct")
		})
	}

}
