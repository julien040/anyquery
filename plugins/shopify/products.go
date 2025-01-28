package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

const productTokenCost = 13

const graphQlProducts = `
query {
  products(first: 250 %s) {
	pageInfo {
    		endCursor
    		hasNextPage
    		hasPreviousPage
    		startCursor
  	}
	nodes{
        id
        title
        vendor
        productType
        createdAt
        updatedAt
        status
        description
        descriptionHtml
        onlineStoreUrl
        onlineStorePreviewUrl
        productType
        totalInventory

      }
    }
  
}`

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func (r *rateLimit) productsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Extract the token
	token := extractKeyUserConf(args.UserConfig, "token")
	if token == "" {
		return nil, nil, fmt.Errorf("token is missing from the configuration")
	}

	storeName := extractKeyUserConf(args.UserConfig, "store_name")
	if storeName == "" {
		return nil, nil, fmt.Errorf("store_name is missing from the configuration")
	}

	// Open the cache
	db, err := openCache("products", token)
	if err != nil {
		return nil, nil, err
	}
	return &productsTable{
			cache:     db,
			token:     token,
			storeName: storeName,
			rateLimit: r,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the product",
				},
				{
					Name: "title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "vendor",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_type",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeDateTime,
				},
				{
					Name: "updated_at",
					Type: rpc.ColumnTypeDateTime,
				},
				{
					Name: "status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "description",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "description_html",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "store_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "store_preview_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "total_inventory",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

type productsTable struct {
	cache     *badger.DB
	token     string
	storeName string
	rateLimit *rateLimit
}

type productsCursor struct {
	cache      *badger.DB
	token      string
	nextCursor string
	storeName  string
	rateLimit  *rateLimit
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *productsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get from the cache if possible
	cacheKey := fmt.Sprintf("products_%s", t.nextCursor)
	var rows [][]interface{}
	apiResponse := &ProductsResponse{}

	// Try to get the data from the cache
	err := t.cache.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			dec := gob.NewDecoder(bytes.NewReader(val))
			return dec.Decode(apiResponse)
		})
	})

	// If the data is not in the cache, fetch it from the API
	if err != nil {
		// Wait for the rate limiter
		err = t.rateLimit.Wait(t.token, productTokenCost)
		if err != nil {
			return nil, true, err
		}

		nextCursorField := ""
		if t.nextCursor != "" {
			nextCursorField = fmt.Sprintf("after: \"%s\"", t.nextCursor)
		}

		res, err := client.R().SetHeader("X-Shopify-Access-Token", t.token).
			SetHeader("Content-Type", "application/json").
			SetBody(&GraphqlQuery{Query: fmt.Sprintf(graphQlProducts, nextCursorField)}).
			SetResult(apiResponse).
			SetPathParam("shop", t.storeName).
			Post(graphQLEndpoint)

		if err != nil {
			return nil, true, fmt.Errorf("failed to fetch orders: %w", err)
		}

		if res.IsError() {
			return nil, true, fmt.Errorf("failed to fetch orders (status code: %d): %s", res.StatusCode(), res.String())
		}

		// Save the data in the cache only if the page is equal to 250
		// Therefore, if new order are added, they will be fetched
		// while the old ones will be cached
		if len(apiResponse.Data.Products.Nodes) == 250 {
			err = t.cache.Update(func(txn *badger.Txn) error {
				var buf bytes.Buffer
				enc := gob.NewEncoder(&buf)
				err := enc.Encode(apiResponse)
				if err != nil {
					return err
				}

				e := badger.NewEntry([]byte(cacheKey), buf.Bytes()).WithTTL(cacheTTL)
				return txn.SetEntry(e)
			})
			if err != nil {
				log.Printf("Failed to save orders in the cache: %v", err)
			}
		}
	}

	if apiResponse.Errors != nil {
		return nil, true, fmt.Errorf("failed to fetch orders: %v", apiResponse.Errors)
	}

	// Update the cursor
	if apiResponse.Data.Products.PageInfo.HasNextPage {
		t.nextCursor = apiResponse.Data.Products.PageInfo.EndCursor
	} else {
		t.nextCursor = ""
	}

	// Extract the rows
	for _, node := range apiResponse.Data.Products.Nodes {
		rows = append(rows, []interface{}{
			node.ID,
			node.Title,
			node.Vendor,
			node.ProductType,
			node.CreatedAt,
			node.UpdatedAt,
			node.Status,
			node.Description,
			node.DescriptionHTML,
			node.OnlineStoreURL,
			node.OnlineStorePreviewURL,
			node.TotalInventory,
		})
	}

	return rows, t.nextCursor == "" || len(rows) < 250, nil
}

// Create a new cursor that will be used to read rows
func (t *productsTable) CreateReader() rpc.ReaderInterface {
	return &productsCursor{
		cache:      t.cache,
		token:      t.token,
		rateLimit:  t.rateLimit,
		storeName:  t.storeName,
		nextCursor: "",
	}
}

// A destructor to clean up resources
func (t *productsTable) Close() error {
	return nil
}
