//go:build !prql

package prql

func ToSQL(prqlQuery string) (string, []CompileMessage) {
	return prqlQuery, nil
}
