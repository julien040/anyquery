package namespace

import "strings"

// GetTableName returns the table name for a given plugin, table and profile
//
// The table name is constructed by concatenating the plugin, table and profile
func GetTableName(plugin string, table string, profile string) string {
	builder := strings.Builder{}
	if profile != "default" {
		builder.WriteString(profile)
		builder.WriteString("_")
	}
	builder.WriteString(plugin)
	builder.WriteString("_")
	builder.WriteString(table)

	return builder.String()
}
