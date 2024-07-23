package main

import (
	"encoding/json"
	"strconv"
	"time"
)

func isInterfaceString(val interface{}) bool {
	_, ok := val.(string)
	return ok
}

func isInterfaceNumber(val interface{}) bool {
	_, ok := val.(float64)
	if !ok {
		_, ok = val.(int64)
	}
	return ok
}

func marshal(val interface{}, to string) interface{} {
	switch to {
	case "barcode":
		if isInterfaceString(val) {
			return map[string]interface{}{"text": val}
		}
	case "checkbox":
		switch parsed := val.(type) {
		case int64:
			return parsed == 1
		case float64:
			return parsed == 1
		case string:
			boolParsed, err := strconv.ParseBool(parsed)
			if err != nil {
				return nil
			}
			return boolParsed
		}
	case "singleCollaborator":
		if isInterfaceString(val) {
			return map[string]interface{}{"id": val}
		}
	case

		"email",
		"multilineText",
		"phoneNumber",
		"richText",
		"singleLineText",
		"singleSelect",
		"url":
		if isInterfaceString(val) {
			return val
		}
	case
		"currency",
		"duration",
		"number",
		"percent":
		if isInterfaceNumber(val) {
			return val
		}
	case "rating":
		switch parsed := val.(type) {
		case int64:
			if parsed > 0 && parsed <= 5 {
				return parsed
			}
		case float64:
			if parsed > 0 && parsed <= 5 {
				return int64(parsed)
			}
		}
	case "date", "dateTime":
		switch parsed := val.(type) {
		case string:
			return parsed
		case int64:
			return time.Unix(parsed, 0).Format(time.RFC3339)
		case float64:
			return time.Unix(int64(parsed), 0).Format(time.RFC3339)
		}
	case
		"multipleRecordLinks",
		"multipleSelects":
		parsedStr, ok := val.(string)
		if !ok {
			return nil
		}
		ids := make([]string, 0)
		err := json.Unmarshal([]byte(parsedStr), &ids)
		if err != nil {
			return nil
		}

		return ids
	case "multipleCollaborators":
		parsed, ok := val.([]interface{})
		if !ok {
			return nil
		}
		ids := make([]map[string]interface{}, 0, len(parsed))
		for _, v := range parsed {
			if isInterfaceString(v) {
				ids = append(ids, map[string]interface{}{"id": v})
			}
		}
		return ids

	}
	return nil
}
