package main

import (
	"context"
	"log"
	"net/url"

	"github.com/jomei/notionapi"
)

func (t *table) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		pageRequest := &notionapi.PageCreateRequest{
			Properties: map[string]notionapi.Property{},
			Parent: notionapi.Parent{
				Type:       notionapi.ParentTypeDatabaseID,
				DatabaseID: notionapi.DatabaseID(t.database.ID),
			},
		}

		for i, colName := range t.columns {
			// Skip the system columns
			if colName == "_page_id" || colName == "_page_url" ||
				colName == "_created_time" || colName == "_last_edited_time" {
				continue
			}

			if colName == "_icon_url" || colName == "_cover_url" {
				// Check if the value is an URL
				if i < len(row) {
					value, ok := row[i].(string)
					if !ok {
						log.Printf("Invalid icon URL: %+v", row[i])
						continue
					}
					parsed, err := url.Parse(value)
					if err != nil {
						log.Printf("Invalid icon URL: %+v", row[i])
						continue
					}
					if parsed.Scheme == "" || parsed.Host == "" {
						log.Printf("Invalid icon URL: %+v", row[i])
						continue
					}
					if colName == "_icon_url" {
						pageRequest.Icon = &notionapi.Icon{
							Type: notionapi.FileTypeExternal,
							External: &notionapi.FileObject{
								URL: value,
							},
						}
					} else if colName == "_cover_url" {
						pageRequest.Cover = &notionapi.Image{
							Type: notionapi.FileTypeExternal,
							External: &notionapi.FileObject{
								URL: value,
							},
						}
					}

				}
			}

			// Get the property of the column
			prop, ok := t.database.Properties[colName]
			if !ok {
				continue
			}

			// Get the value of the column
			var value interface{}
			if i < len(row) {
				value = row[i]
			} else {
				value = nil
			}

			// Convert the value to a Notion property
			propValue := marshal(value, prop)
			if propValue == nil {
				continue
			}
			pageRequest.Properties[colName] = propValue

		}
		// Create the page
		_, err := t.client.Page.Create(context.Background(), pageRequest)
		if err != nil {
			return err
		}

	}

	// Clear the cache
	clearCache(t.cacheDB)

	return nil
}
