package namespace

import (
	"encoding/json"
	"reflect"

	"github.com/mattn/go-sqlite3"
)

/* ------------------------------- Clear cache ------------------------------ */

func registerJSONFunctions(conn *sqlite3.SQLiteConn) {
	var otherFunctions = []struct {
		name     string
		function any
		pure     bool
	}{
		{
			"json_unquote", jsonUnquote, true,
		},
		{
			"json_has", jsonHas, true,
		},
		{
			"json_contains", jsonHas, true,
		},
	}
	for _, f := range otherFunctions {
		conn.RegisterFunc(f.name, f.function, f.pure)
	}
}

func jsonUnquote(s string) any {
	// Parse the string
	var data interface{}
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		return s
	}

	switch data.(type) {
	// Not supported types
	case map[string]interface{}, []interface{}:
		return s
	case string:
		return data.(string)
	default:
		return s
	}
}

// jsonHas returns true if the key is present in the JSON object
// or if the value is present in the JSON array.
func jsonHas(s string, key interface{}) bool {
	// Parse the string
	var data interface{}
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		return false
	}

	switch data.(type) {
	case map[string]interface{}:
		if _, ok := key.(string); !ok {
			return false
		}
		_, ok := data.(map[string]interface{})[key.(string)]
		return ok
	case []interface{}:
		for _, v := range data.([]interface{}) {
			switch value := v.(type) {
			case string:
				if v == key {
					return true
				}
			case map[string]interface{}:
				// Parse the key
				var keyData interface{}
				err := json.Unmarshal([]byte(key.(string)), &keyData)
				if err != nil {
					continue
				}
				return reflect.DeepEqual(v, keyData)

			case float64:
				switch key.(type) {
				case float64:
					if v == key {
						return true
					}

				case int64:
					if int64(value) == key {
						return true
					}

				}

			case bool:
				if value && key == int64(1) {
					return true
				}
				if !value && key == int64(0) {
					return true
				}
			}

		}

	default:
		return s == key
	}
	return false
}
