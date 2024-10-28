package prql

import (
	"fmt"
	"os/exec"
	"strings"
)

// Calls the prqlc CLI to compile the PRQL query to SQL
func ToSQL(prqlQuery string) (string, []CompileMessage) {
	_, err := exec.LookPath("prqlc")
	if err != nil {
		return "", []CompileMessage{{
			ErrorCode: 'E',
			Display:   "prqlc not found in PATH.\nMake sure it is installed: https://prql-lang.org/book/project/integrations/prqlc-cli.html#installation",
		}}
	}

	//  prqlc compile --target sql.sqlite"
	cmd := exec.Command("prqlc")
	cmd.Args = append(cmd.Args, "compile", "--target", "sql.sqlite")
	cmd.Stdin = strings.NewReader(prqlQuery)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", []CompileMessage{{
			ErrorCode: 'E',
			Display:   fmt.Sprintf("prqlc failed with error %s: %s", err, string(out)),
		}}
	}

	return string(out), nil
}
