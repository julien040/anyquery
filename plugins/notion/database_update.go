package main

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
)

func (t *table) Update(rows [][]interface{}) error {
	for _, row := range rows {
		primaryKey, ok := row[0].(string)
		if !ok {
			return fmt.Errorf("invalid page id: %v", row[0])
		}

		pageUpdateRequest := &notionapi.PageUpdateRequest{
			Properties: map[string]notionapi.Property{},
		}

		for i, colName := range t.columns {
			// Skip the system columns
			if colName == "_page_id" || colName == "_page_url" || colName == "_icon_url" || colName == "_cover_url" ||
				colName == "_created_time" || colName == "_last_edited_time" {
				continue
			}

			prop, ok := t.database.Properties[colName]
			if !ok {
				continue
			}

			var value interface{}
			if i < len(row) {
				value = row[i+1] // Skip the primary key
			} else {
				value = nil
			}

			propValue := marshal(value, prop)
			if propValue == nil {
				continue
			}

			pageUpdateRequest.Properties[colName] = propValue

		}

		_, err := t.client.Page.Update(context.Background(), notionapi.PageID(primaryKey), pageUpdateRequest)
		if err != nil {
			return err
		}

	}

	clearCache(t.cacheDB)

	return nil
}
