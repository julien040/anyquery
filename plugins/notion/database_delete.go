package main

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
)

func (t *table) Delete(primaryKeys []interface{}) error {
	for _, pk := range primaryKeys {
		primaryKey, ok := pk.(string)
		if !ok {
			return fmt.Errorf("invalid page id: %v", pk)
		}

		_, err := t.client.Page.Update(context.Background(), notionapi.PageID(primaryKey), &notionapi.PageUpdateRequest{
			Archived:   true,
			Properties: map[string]notionapi.Property{},
		})
		if err != nil {
			return err
		}
	}

	clearCache(t.cacheDB)
	return nil
}
