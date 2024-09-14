package helper

import (
	"encoding/json"
	"path"
	"time"

	"github.com/adrg/xdg"
)

// Find the directory where the plugin can cache data
//
// It is recommended to use this function to get the cache directory
// because the SQL function clear_plugin_cache(plugin_name) will clear
// the directory returned by this function
//
// Internally, this function uses the XDG_CACHE_HOME environment variable
// so that anyone can override the cache directory to its needs
func GetCachePath(pluginName string) string {
	return path.Join(xdg.CacheHome, "anyquery", "plugins", pluginName)
}

func serializeJSON(v interface{}) interface{} {
	res, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(res)
}

// Serialize a value to a type that can be sent to Anyquery, and respecting Anyquery guidelines
//
// This is useful to not bother with nil-checks, JSON serialization, etc.
// when returning values from the plugin
//
// Internally, because Anyquery uses JSON for any non-primitive types,
// this function will convert any non-primitive types to JSON
//
//   - When it encounters a pointer, it will return nil if the pointer is nil or the value if the pointer is not nil
//   - When it encounters a slice, it will return nil if the slice is empty or the JSON representation of the slice if it is not empty
//   - When it encounters a time.Time, it will return the RFC3339 representation of the time
//   - Primitive types are returned as is
func Serialize(v interface{}) interface{} {
	switch val := v.(type) {
	case string, int, int64, float64, bool, uint, uint8, uint16, uint32, uint64, uintptr, int8, int16, int32, float32:
		return val
	case []string, []int, []int64, []float64, []bool, []byte:
		return val
	case []interface{}:
		if len(v.([]interface{})) == 0 {
			return nil
		}
		// Convert to a JSON
		return serializeJSON(val)
	case *string:
		if v == nil || val == nil {
			return nil
		}
		return *val
	case *int:
		if v == nil || val == nil {
			return nil
		}
		return *val
	case *int64:
		if v == nil || val == nil {
			return nil
		}
		return *val
	case *float64:
		if v == nil || val == nil {
			return nil
		}
		return *val
	case *bool:
		if v == nil || val == nil {
			return nil
		}
		return *val

	case *[]string:
		if v == nil || val == nil || len(*val) == 0 {
			return nil
		}
		return *val

	case time.Time:
		return val.Format(time.RFC3339)
	case *time.Time:
		if v == nil || val == nil {
			return nil
		}
		return val.Format(time.RFC3339)
	default:
		if val == nil {
			return nil
		}
		return serializeJSON(val)
	}
}
