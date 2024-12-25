package namespace

import (
	"fmt"
	"os"
	pathlib "path"
	"strconv"

	"github.com/adrg/xdg"
	u "github.com/bcicen/go-units"
	"github.com/julien040/anyquery/module"
	"github.com/mattn/go-sqlite3"
)

/* ------------------------------- Clear cache ------------------------------ */

func registerOtherFunctions(conn *sqlite3.SQLiteConn) {
	var otherFunctions = []struct {
		name     string
		function any
		pure     bool
	}{
		{"clear_file_cache", clear_file_cache, true},
		{"clear_plugin_cache", clear_plugin_cache, true},
		{"convert_unit", convert_unit, true},
		{"format_unit", format_unit, true},
	}
	for _, f := range otherFunctions {
		conn.RegisterFunc(f.name, f.function, f.pure)
	}
}

func clear_file_cache() string {
	pathToRemove := pathlib.Join(xdg.CacheHome, "anyquery", "downloads")

	// Remove the directory
	err := os.RemoveAll(pathToRemove)
	if err != nil {
		return err.Error()
	}

	return ""
}

func clear_plugin_cache(plugin string) string {
	pathToRemove := pathlib.Join(xdg.CacheHome, "anyquery", "plugins", plugin)

	if plugin == "" {
		return "The plugin name is empty"
	}

	// Remove the directory
	err := os.RemoveAll(pathToRemove)
	if err != nil {
		return err.Error()
	}

	return ""
}

type bufferFlusher struct {
	modules *map[string]*module.SQLiteModule
}

func (b *bufferFlusher) Clear(plugin string) string {
	if plugin == "" {
		return "The plugin name is empty"
	}

	if mod, ok := (*b.modules)[plugin]; !ok {
		return "The plugin does not exist"
	} else {
		if mod.Table == nil {
			return "The plugin is not loaded"
		}

		mod.Table.ClearBuffers()
	}

	return ""
}

func (b *bufferFlusher) Flush(plugin string) string {
	if plugin == "" {
		return "The plugin name is empty"
	}

	if mod, ok := (*b.modules)[plugin]; !ok {
		return "The plugin does not exist"
	} else {
		if mod.Table == nil {
			return "The plugin is not loaded"
		}

		err := mod.Table.FlushBuffers()
		if err != nil {
			return err.Error()
		}
	}

	return ""
}

// Conversion functions

func convert_unit(unknownValue any, incomingUnit string, targetUnit string) (float64, error) {
	var value float64
	switch parsed := unknownValue.(type) {
	case int64:
		value = float64(parsed)
	case float64:
		value = parsed
	case string:
		// Try to parse the value
		var err error
		value, err = strconv.ParseFloat(parsed, 64)
		if err != nil {
			return 0, fmt.Errorf("The value is not a number: %s", parsed)
		}
	default:
		return 0, fmt.Errorf("The value is not a number: %v (%T)", unknownValue, unknownValue)
	}

	// Try to parse the incoming unit
	incoming, err := u.Find(incomingUnit)
	if err != nil {
		return 0, err
	}

	// Try to parse the target unit
	target, err := u.Find(targetUnit)
	if err != nil {
		return 0, err
	}

	// Convert the value
	uValue := u.NewValue(value, incoming)
	converted, err := uValue.Convert(target)
	if err != nil {
		return 0, err
	}

	return converted.Float(), nil
}

func format_unit(unknownValue any, unit string, opts ...any) (string, error) {

	var value float64
	switch parsed := unknownValue.(type) {
	case int64:
		value = float64(parsed)
	case float64:
		value = parsed
	case string:
		// Try to parse the value
		var err error
		value, err = strconv.ParseFloat(parsed, 64)
		if err != nil {
			return "", fmt.Errorf("The value is not a number: %s", parsed)
		}
	default:
		return "", fmt.Errorf("The value is not a number: %v (%T)", unknownValue, unknownValue)
	}

	// Try to parse the unit
	parsedUnit, err := u.Find(unit)
	if err != nil {
		return "", err
	}

	formatOpts := u.FmtOptions{
		Label:     true,  // append unit name/symbol
		Short:     false, // use unit symbol
		Precision: 5,
	}

	if len(opts) > 0 {
		if val, ok := opts[0].(int64); ok && val != 0 {
			formatOpts.Short = true
		}
	}

	if len(opts) > 1 {
		if val, ok := opts[1].(int64); ok {
			formatOpts.Precision = int(val)
		}
	}

	// Convert the value
	uValue := u.NewValue(value, parsedUnit)
	return uValue.Fmt(formatOpts), nil

}
