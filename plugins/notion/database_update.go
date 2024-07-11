package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jomei/notionapi"
)

func (t *table) Update(rows [][]interface{}) error {
	log.Printf("Updating %d rows", len(rows))
	for _, row := range rows {
		primaryKey, ok := row[0].(string)
		log.Printf("Cols: %v", t.columns)
		log.Printf("Updating page %s", primaryKey)
		log.Printf("Row: %v", row)
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

			log.Printf("Updating %s: %v(%T)", colName, propValue, propValue)

			pageUpdateRequest.Properties[colName] = propValue

		}

		marshaled, err := json.Marshal(pageUpdateRequest)
		if err != nil {
			return err
		}

		log.Printf("Update request: %s", marshaled)
		_, err = t.client.Page.Update(context.Background(), notionapi.PageID(primaryKey), pageUpdateRequest)
		if err != nil {
			return err
		}

	}

	clearCache(t.cacheDB)

	return nil
}
