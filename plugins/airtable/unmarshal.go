package main

import "encoding/json"

// Extract a value from a map and return nil if the value is missing
// or the val is not a map.
func helperGetMapField(val interface{}, key string) interface{} {
	if parsedMap, ok := val.(map[string]interface{}); ok {
		if v, ok := parsedMap[key]; ok {
			return v
		}
	}
	return nil
}

func serializeJSON(val interface{}) interface{} {
	serialized, err := json.Marshal(val)
	if err != nil {
		return nil
	} else {
		return string(serialized)
	}
}

func unmarshal(val interface{}, is string) interface{} {
	switch is {
	case "aiText":
		return helperGetMapField(val, "value")
	case "multipleAttachments":
		parsed, ok := val.([]map[string]interface{})
		if !ok {
			return nil
		}
		// Remove the thumbnails field
		for k := range parsed {
			delete(parsed[k], "thumbnails")
		}
		return serializeJSON(parsed)
	case
		"autoNumber",
		"checkbox",
		"count",
		"createdTime",
		"currency",
		"date",
		"dateTime",
		"duration",
		"email",
		"lastModifiedTime",
		"multilineText",
		"number",
		"percent",
		"phoneNumber",
		"rating",
		"richText",
		"rollup",
		"singleLineText",
		"singleSelect",
		"url":
		return val

	case "barcode":
		return helperGetMapField(val, "text")
	case "button":
		return helperGetMapField(val, "url")
	case "singleCollaborator",
		"createdBy",
		"lastModifiedBy",
		"externalSyncSource":
		return helperGetMapField(val, "id")
	case "formula":
		switch parsed := val.(type) {
		case []interface{}:
			// Array of values
			// We need to serialize it because anyquery will reject an array of interface{}
			// it only accepts typed arrays
			return serializeJSON(parsed)
		default:
			//string, float64, bool
			return val
		}

	case
		"multipleRecordLinks",
		"multipleLookupValues",
		"multipleSelects":
		return serializeJSON(val)

	case "multipleCollaborators":
		parsed, ok := val.([]map[string]interface{})
		if !ok {
			return nil
		}
		ids := make([]string, 0, len(parsed))
		for _, v := range parsed {
			if id, ok := v["id"]; ok {
				ids = append(ids, id.(string))
			}
		}
		return serializeJSON(ids)
	default:
		return nil

	}
}
