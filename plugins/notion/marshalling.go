package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/jomei/notionapi"
)

// This file contains functions that convert Notion types to Go types and vice versa.

func richTextToMarkdown(richText []notionapi.RichText) string {
	var s strings.Builder
	for _, v := range richText {
		if v.Text != nil {
			switch {
			case v.Annotations.Bold:
				s.WriteString("**")
				s.WriteString(v.Text.Content)
				s.WriteString("**")
			case v.Annotations.Italic:
				s.WriteString("*")
				s.WriteString(v.Text.Content)
				s.WriteString("*")
			case v.Annotations.Code:
				s.WriteString("`")
				s.WriteString(v.Text.Content)
				s.WriteString("`")
			case v.Annotations.Strikethrough:
				s.WriteString("~~")
				s.WriteString(v.Text.Content)
				s.WriteString("~~")
			case v.Text.Link != nil:
				s.WriteString("[")
				s.WriteString(v.Text.Content)
				s.WriteString("](")
				s.WriteString(v.Text.Link.Url)
				s.WriteString(")")
			default:
				s.WriteString(v.Text.Content)

			}
		}
	}

	return s.String()

}

func markdownToRichText(markdownStr string) []notionapi.RichText {
	parser := parser.NewWithExtensions(parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock)

	md := []byte(markdownStr)
	mdAST := parser.Parse(md)

	log.Printf("mdASTChildren: %T %#v", mdAST.AsContainer().Children[0].GetChildren(), mdAST.AsContainer().Children[0].GetChildren())

	var richText []notionapi.RichText

	document := mdAST.AsContainer()
	if document == nil {
		log.Printf("markdownToRichText: unsupported type %T", mdAST)
		return nil
	}
	paragraph, ok := document.Children[0].(*ast.Paragraph)
	if !ok || paragraph == nil {
		log.Printf("markdownToRichText: unsupported type %T", document.Children[0])
		return nil
	}

	for i, node := range paragraph.GetChildren() {
		log.Printf("node[%d]: %T %+v", i, node, node)
		switch n := node.(type) {
		case *ast.Text:
			// If the text is longer than 2000 characters, split it into multiple text objects
			for i := 0; i < len(n.Literal); i += 2000 {
				end := i + 2000
				if end > len(n.Literal) {
					end = len(n.Literal)
				}

				richText = append(richText, notionapi.RichText{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{
						Content: string(n.Literal[i:end]),
					},
					Annotations: &notionapi.Annotations{
						Bold:   false,
						Italic: false,
						Color:  notionapi.ColorDefault,
					},
					PlainText: string(n.Literal[i:end]),
				})
			}

		case *ast.Emph:
			// Extract the text from the children
			text := strings.Builder{}
			for _, child := range n.GetChildren() {
				if textNode, ok := child.(*ast.Text); ok && textNode != nil {
					text.WriteString(string(textNode.Literal))
				}
			}
			richText = append(richText, notionapi.RichText{
				Type: notionapi.ObjectTypeText,
				Text: &notionapi.Text{
					Content: text.String(),
				},
				Annotations: &notionapi.Annotations{
					Bold:   false,
					Italic: true,
					Color:  notionapi.ColorDefault,
				},
				PlainText: text.String(),
			})
		case *ast.Strong:
			// Extract the text from the children
			text := strings.Builder{}
			for _, child := range n.GetChildren() {
				if textNode, ok := child.(*ast.Text); ok && textNode != nil {
					text.WriteString(string(textNode.Literal))
				}
			}
			richText = append(richText, notionapi.RichText{
				Type: notionapi.ObjectTypeText,
				Text: &notionapi.Text{
					Content: text.String(),
				},
				Annotations: &notionapi.Annotations{
					Bold:   true,
					Italic: false,
					Color:  notionapi.ColorDefault,
				},
				PlainText: text.String(),
			})
		case *ast.Link:
			richText = append(richText, notionapi.RichText{
				Type: notionapi.ObjectTypeText,
				Text: &notionapi.Text{
					Content: string(n.Content),
					Link:    &notionapi.Link{Url: string(n.Destination)},
				},
				Annotations: &notionapi.Annotations{
					Color: notionapi.ColorDefault,
				},
				PlainText: string(n.Destination),
			})
		case *ast.Code:
			richText = append(richText, notionapi.RichText{
				Type: notionapi.ObjectTypeText,
				Text: &notionapi.Text{
					Content: string(n.Literal),
				},
				Annotations: &notionapi.Annotations{
					Color: notionapi.ColorDefault,
					Code:  true,
				},
				PlainText: string(n.Literal),
			})
		case *ast.Del:
			// Extract the text from the children
			text := strings.Builder{}
			for _, child := range n.GetChildren() {
				if textNode, ok := child.(*ast.Text); ok && textNode != nil {
					text.WriteString(string(textNode.Literal))
				}
			}
			richText = append(richText, notionapi.RichText{
				Type: notionapi.ObjectTypeText,
				Text: &notionapi.Text{
					Content: text.String(),
				},
				Annotations: &notionapi.Annotations{
					Bold:          false,
					Italic:        false,
					Color:         notionapi.ColorDefault,
					Strikethrough: true,
				},
				PlainText: text.String(),
			})
		}
	}

	return richText
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
	log.Printf("unmarshal %T: %#v", v, v)
	if v == nil {
		return nil
	}
	switch value := v.(type) {
	case *notionapi.TextProperty:
		return richTextToMarkdown(value.Text)
	case *notionapi.NumberProperty:
		if value == nil {
			return nil
		}
		return value.Number
	case *notionapi.SelectProperty:
		if value == nil {
			return nil
		}
		return value.Select.Name
	case *notionapi.MultiSelectProperty:
		if value == nil {
			return nil
		}
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
		if value == nil {
			return nil
		}
		var valueTime string
		if value.Date == nil {
			return nil
		}
		if value.Date.Start != nil {
			// Notion date can be handled as datetime, or date only
			if value.Date.DateOnly {
				valueTime = time.Time(*value.Date.Start).Format("2006-01-02")
			} else {
				valueTime = time.Time(*value.Date.Start).Format("2006-01-02T15:04:05Z")
			}
		}

		if value.Date.End != nil && value.Date.Start != nil {
			valueTime += "/"
		}

		if value.Date.End != nil {
			if value.Date.DateOnly {
				valueTime += time.Time(*value.Date.End).Format("2006-01-02")
			} else {
				valueTime += time.Time(*value.Date.End).Format("2006-01-02T15:04:05Z")
			}
		}

		return valueTime
	case *notionapi.CheckboxProperty:
		if value == nil {
			return nil
		}
		if value.Checkbox {
			return 1
		}
		return 0

	case *notionapi.RelationProperty:
		if value == nil {
			return nil
		}
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
		if value == nil {
			return nil
		}
		return value.URL
	case *notionapi.FormulaProperty:
		if value == nil {
			return nil
		}
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
		if value == nil {
			return nil
		}
		ids := make([]string, 0, len(value.People))
		for _, v := range value.People {
			ids = append(ids, string(v.ID))
		}
		// Marshal the array to a string
		jsonArr, err := json.Marshal(ids)
		if err != nil {
			return nil
		}
		return string(jsonArr)

	case *notionapi.FilesProperty:
		if value == nil {
			return nil
		}
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
		if value == nil {
			return nil
		}
		return value.PhoneNumber
	case *notionapi.EmailProperty:
		if value == nil {
			return nil
		}
		return value.Email
	case *notionapi.TitleProperty:
		if value == nil {
			return nil
		}
		return richTextToMarkdown(value.Title)
	case *notionapi.RichTextProperty:
		if value == nil {
			return nil
		}
		return richTextToMarkdown(value.RichText)
	case *notionapi.CreatedByProperty:
		if value == nil {
			return nil
		}
		return value.CreatedBy.Name
	case *notionapi.CreatedTimeProperty:
		if value == nil {
			return nil
		}
		return time.Time(value.CreatedTime).Format(time.RFC3339)
	case *notionapi.LastEditedByProperty:
		if value == nil {
			return nil
		}
		return value.LastEditedBy.Name
	case *notionapi.LastEditedTimeProperty:
		if value == nil {
			return nil
		}
		return time.Time(value.LastEditedTime).Format(time.RFC3339)
	case *notionapi.RollupProperty:
		if value == nil {
			return nil
		}
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
		if value == nil {
			return nil
		}
		return value.Status.Name
	case *notionapi.UniqueIDProperty:
		if value == nil {
			return nil
		}
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
		if _, ok := value.(string); !ok {
			return nil
		}
		arrayRichText := markdownToRichText(value.(string))
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
		var start, end *notionapi.Date
		dateOnly := false
		if len(splitted) > 0 {
			startParse, err := parseTime(splitted[0])
			if err != nil {
				return nil
			}
			val := notionapi.Date(startParse)
			start = &val
			// If the date is only 10 characters long (YYYY-MM-DD), it's a date only
			// It's a workaround for the Notion library, which set the DateOnly field to true
			// to indicate that the date is only a date, not a datetime
			if len(splitted[0]) == 10 {
				dateOnly = true
			}

		}
		if len(splitted) > 1 {
			endParse, err := parseTime(splitted[1])
			if err != nil {
				return nil
			}
			val := notionapi.Date(endParse)
			end = &val

			dateOnly = dateOnly || len(splitted[1]) == 10
		}

		return notionapi.DateProperty{
			Date: &notionapi.DateObject{
				Start:    start,
				End:      end,
				DateOnly: len(splitted[0]) == 10,
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

	case notionapi.PropertyConfigTypePeople:
		// Get the IDs from the JSON array
		v, ok := value.(string)
		if !ok {
			return nil
		}
		people := []string{}
		err := json.Unmarshal([]byte(v), &people)
		if err != nil {
			return nil
		}

		peopleArray := []notionapi.User{}
		for _, id := range people {
			peopleArray = append(peopleArray, notionapi.User{ID: notionapi.UserID(id)})
		}
		return notionapi.PeopleProperty{
			People: peopleArray,
		}

	// We don't support these types yet
	// TODO
	case notionapi.PropertyConfigTypeRelation:
		return nil
	case notionapi.PropertyConfigTypeFormula:
		return nil
	case notionapi.PropertyConfigTypeRollup:
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
