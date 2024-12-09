package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jomei/notionapi"
)

// This file contains functions that convert Notion types to Go types and vice versa.

func richTextToString(richText []notionapi.RichText) string {
	var s string
	for _, r := range richText {
		s += r.PlainText
	}
	return s
}

// Convert a Notion property to a Go type.
//
// Types that will be returned:
//   - string
//   - float64
//   - int
//
// If the property is not supported, nil will be returned.
// Arrays are returned as their JSON representation in a string.
// Boolean like checkboxes are returned as an int with 1 for true and 0 for false.
func unmarshal(v notionapi.Property) interface{} {
	if v == nil {
		return nil
	}
	switch value := v.(type) {
	case *notionapi.TextProperty:
		return richTextToString(value.Text)
	case *notionapi.NumberProperty:
		return value.Number
	case *notionapi.SelectProperty:
		return value.Select.Name
	case *notionapi.MultiSelectProperty:
		s := make([]string, len(value.MultiSelect))
		for i, v := range value.MultiSelect {
			s[i] = v.Name
		}
		// Marshal the array to a string
		jsonArr, err := json.Marshal(s)
		if err != nil {
			return nil
		}
		return string(jsonArr)
	case *notionapi.DateProperty:
		var valueTime string
		if value.Date == nil {
			return nil
		}
		if value.Date.Start != nil {
			valueTime = time.Time(*value.Date.Start).Format(time.RFC3339)
		}

		if value.Date.End != nil && value.Date.Start != nil {
			valueTime += "/"
		}

		if value.Date.End != nil {
			valueTime += time.Time(*value.Date.End).Format(time.RFC3339)
		}

		return valueTime
	case *notionapi.CheckboxProperty:
		if value.Checkbox {
			return 1
		}
		return 0

	case *notionapi.RelationProperty:
		ids := make([]string, 0, len(value.Relation))
		for _, v := range value.Relation {
			ids = append(ids, v.ID.String())
		}

		// Marshal the array to a string
		jsonArr, err := json.Marshal(ids)
		if err != nil {
			return nil
		}
		return string(jsonArr)
	case *notionapi.URLProperty:
		return value.URL
	case *notionapi.FormulaProperty:
		switch value.Formula.Type {
		case notionapi.FormulaTypeString:
			return value.Formula.String
		case notionapi.FormulaTypeNumber:
			return value.Formula.Number
		case notionapi.FormulaTypeBoolean:
			if value.Formula.Boolean {
				return 1
			}
			return 0
		case notionapi.FormulaTypeDate:
			var valueTime string
			if value.Formula.Date.Start != nil {
				valueTime = time.Time(*value.Formula.Date.Start).Format(time.RFC3339)
			}
			if value.Formula.Date.End != nil && value.Formula.Date.Start != nil {
				valueTime += "/"
			}
			if value.Formula.Date.End != nil {
				valueTime += time.Time(*value.Formula.Date.End).Format(time.RFC3339)
			}
			return valueTime

		default:
			return nil
		}
	case *notionapi.PeopleProperty:
		ids := make([]string, 0, len(value.People))
		for _, v := range value.People {
			ids = append(ids, v.Name)
		}
		// Marshal the array to a string
		jsonArr, err := json.Marshal(ids)
		if err != nil {
			return nil
		}
		return string(jsonArr)

	case *notionapi.FilesProperty:
		ids := make([]string, 0, len(value.Files))
		for _, v := range value.Files {
			if v.Type == notionapi.FileTypeExternal {
				ids = append(ids, v.External.URL)
			} else {
				ids = append(ids, v.File.URL)
			}
		}
		// Marshal the array to a string
		jsonArr, err := json.Marshal(ids)
		if err != nil {
			return nil
		}
		return string(jsonArr)

	case *notionapi.PhoneNumberProperty:
		return value.PhoneNumber
	case *notionapi.EmailProperty:
		return value.Email
	case *notionapi.TitleProperty:
		return richTextToString(value.Title)
	case *notionapi.RichTextProperty:
		return richTextToString(value.RichText)
	case *notionapi.CreatedByProperty:
		return value.CreatedBy.Name
	case *notionapi.CreatedTimeProperty:
		return time.Time(value.CreatedTime).Format(time.RFC3339)
	case *notionapi.LastEditedByProperty:
		return value.LastEditedBy.Name
	case *notionapi.LastEditedTimeProperty:
		return time.Time(value.LastEditedTime).Format(time.RFC3339)
	case *notionapi.RollupProperty:
		switch value.Rollup.Type {
		case notionapi.RollupTypeNumber:
			return value.Rollup.Number
		case notionapi.RollupTypeDate:
			var valueTime string
			if value.Rollup.Date.Start != nil {
				valueTime = time.Time(*value.Rollup.Date.Start).Format(time.RFC3339)
			}
			if value.Rollup.Date.End != nil && value.Rollup.Date.Start != nil {
				valueTime += "/"
			}
			if value.Rollup.Date.End != nil {
				valueTime += time.Time(*value.Rollup.Date.End).Format(time.RFC3339)
			}
			return valueTime
		case notionapi.RollupTypeArray:
			return nil
		default:
			return nil
		}
	case *notionapi.StatusProperty:
		return value.Status.Name
	case *notionapi.UniqueIDProperty:
		id := strconv.Itoa(value.UniqueID.Number)
		if value.UniqueID.Prefix != nil {
			id = *value.UniqueID.Prefix + id
		}
		return id

	default:
		log.Printf("unmarshal: unsupported type %T", v)
		return nil

	}
}

// Convert a Go type to a Notion property.
func marshal(value interface{}, to notionapi.PropertyConfig) notionapi.Property {
	if value == nil {
		return nil
	}
	switch to.GetType() {
	case notionapi.PropertyConfigTypeRichText, notionapi.PropertyConfigTypeTitle:
		v := ""
		if _, ok := value.(string); ok {
			v = value.(string)
		} else if _, ok := value.(float64); ok {
			v = strconv.FormatFloat(value.(float64), 'f', -1, 64)
		} else if _, ok := value.(int); ok {
			v = strconv.Itoa(value.(int))
		} else {
			return nil
		}
		arrayRichText := []notionapi.RichText{
			{
				Type: notionapi.ObjectTypeText,
				Text: &notionapi.Text{
					Content: v,
					Link:    nil,
				},
				Annotations: &notionapi.Annotations{
					Bold:   false,
					Italic: false,
					Color:  notionapi.ColorDefault,
				},
				PlainText: v,
			},
		}
		if to.GetType() == notionapi.PropertyConfigTypeRichText {
			return notionapi.RichTextProperty{
				RichText: arrayRichText,
			}
		} else {
			return notionapi.TitleProperty{
				ID:    "title",
				Type:  notionapi.PropertyTypeTitle,
				Title: arrayRichText,
			}
		}
	case notionapi.PropertyConfigTypeNumber:
		v := 0.0
		if _, ok := value.(float64); ok {
			v = value.(float64)
		} else if _, ok := value.(int64); ok {
			v = float64(value.(int64))
		} else if _, ok := value.(string); ok {
			f, err := strconv.ParseFloat(value.(string), 64)
			if err != nil {
				return nil
			}
			v = f
		} else {
			return nil
		}
		return notionapi.NumberProperty{
			Number: v,
		}
	case notionapi.PropertyConfigTypeSelect:
		v, ok := value.(string)
		if !ok {
			return nil
		}
		return notionapi.SelectProperty{
			Select: notionapi.Option{
				Name: v,
			},
		}
	case notionapi.PropertyConfigTypeMultiSelect:
		v, ok := value.(string)
		if !ok {
			return nil
		}
		var multiSelect []notionapi.Option
		var options []string
		err := json.Unmarshal([]byte(v), &options)
		if err != nil {
			return nil
		}
		for _, option := range options {
			multiSelect = append(multiSelect, notionapi.Option{Name: option})
		}
		// Otherwise, it sets the value to null and the Notion API validation will fail
		if len(multiSelect) == 0 {
			return nil
		}
		return notionapi.MultiSelectProperty{
			MultiSelect: multiSelect,
		}
	case notionapi.PropertyConfigTypeDate:
		v, ok := value.(string)
		if !ok {
			unixTime, ok := value.(int64)
			if !ok {
				return nil
			}
			t := time.Unix(unixTime, 0)
			date := notionapi.Date(t)
			return notionapi.DateProperty{
				Date: &notionapi.DateObject{
					Start: &date,
					End:   nil,
				},
			}
		}
		// Split the date into start and end
		splitted := strings.Split(v, "/")
		log.Printf("splitted: %v", splitted)
		var start, end *notionapi.Date
		if len(splitted) > 0 {
			startParse, err := parseTime(splitted[0])
			if err != nil {
				return nil
			}
			val := notionapi.Date(startParse)
			start = &val
		}
		log.Printf("start: %v", start)
		if len(splitted) > 1 {
			endParse, err := parseTime(splitted[1])
			if err != nil {
				return nil
			}
			val := notionapi.Date(endParse)
			end = &val
		}
		log.Printf("end: %v", end)

		return notionapi.DateProperty{
			Date: &notionapi.DateObject{
				Start: start,
				End:   end,
			},
		}

	case notionapi.PropertyConfigTypeCheckbox:
		tempVal := false
		switch v := value.(type) {
		case int64:
			tempVal = v == int64(1)
		case bool:
			tempVal = v
		case string:
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				return nil
			}
			tempVal = parsed
		default:
			return nil
		}

		return notionapi.CheckboxProperty{
			Checkbox: tempVal,
		}
	case notionapi.PropertyConfigTypeEmail:
		v, ok := value.(string)
		if !ok {
			return nil
		}
		return notionapi.EmailProperty{
			Email: v,
		}
	case notionapi.PropertyConfigTypePhoneNumber:
		v, ok := value.(string)
		if !ok {
			return nil
		}
		return notionapi.PhoneNumberProperty{
			PhoneNumber: v,
		}

	case notionapi.PropertyConfigTypeURL:
		v, ok := value.(string)
		if !ok {
			return nil
		}
		return notionapi.URLProperty{
			URL: v,
		}

	case notionapi.PropertyConfigStatus:
		v, ok := value.(string)
		if !ok {
			return nil
		}
		return notionapi.StatusProperty{
			Status: notionapi.Option{
				Name: v,
			},
		}

	// We don't support these types yet
	// TODO
	case notionapi.PropertyConfigTypeRelation:
		return nil
	case notionapi.PropertyConfigTypeFormula:
		return nil
	case notionapi.PropertyConfigTypeRollup:
		return nil
	case notionapi.PropertyConfigTypePeople:
		return nil
	case notionapi.PropertyConfigTypeFiles:
		return nil
	}
	return nil
}

func parseTime(timeStr string) (time.Time, error) {
	// Try to parse with different formats
	// RFC3339, DateTime, DateOnly, TimeOnly
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse(time.DateTime, timeStr)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("2006-01-02 15:04", timeStr)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("01/02/2006", timeStr)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse(time.DateOnly, timeStr)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse(time.TimeOnly, timeStr)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("15:04", timeStr)
	if err == nil {
		return t, nil
	}
	return time.Time{}, err

}
