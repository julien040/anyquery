package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

const productVariantTokenCost = 24

const graphQlProductsVariant = `
query {
  productVariants(first: 250 %s) {
	pageInfo {
    		endCursor
    		hasNextPage
    		hasPreviousPage
    		startCursor
  	}
	nodes {
        availableForSale
        barcode
        createdAt
        updatedAt
        displayName
        id
        inventoryQuantity
        position
        price
        product {
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
        sellableOnlineQuantity
        sku
        title

      }
    }
  
}`

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func (r *rateLimit) products_variantCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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
	db, err := openCache("product_variants", token)
	if err != nil {
		return nil, nil, err
	}
	return &products_variantTable{
			cache:     db,
			token:     token,
			storeName: storeName,
			rateLimit: r,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "barcode",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "updated_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "display_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "sku",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "available_for_sale",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "inventory_quantity",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "position",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "sellable_online_quantity",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_vendor",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_type",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_updated_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_description",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_description_html",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_store_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_store_preview_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "product_total_inventory",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

type products_variantTable struct {
	cache     *badger.DB
	token     string
	storeName string
	rateLimit *rateLimit
}

type products_variantCursor struct {
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
func (t *products_variantCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get from the cache if possible
	cacheKey := fmt.Sprintf("product_variants_%s", t.nextCursor)
	var rows [][]interface{}
	apiResponse := &ProductsVariantResponse{}

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
		log.Printf("Failed to get orders from the cache: %v", err)
		// Wait for the rate limiter
		err = t.rateLimit.Wait(t.token, productVariantTokenCost)
		if err != nil {
			return nil, true, err
		}

		nextCursorField := ""
		if t.nextCursor != "" {
			nextCursorField = fmt.Sprintf("after: \"%s\"", t.nextCursor)
		}

		res, err := client.R().SetHeader("X-Shopify-Access-Token", t.token).
			SetHeader("Content-Type", "application/json").
			SetBody(&GraphqlQuery{Query: fmt.Sprintf(graphQlProductsVariant, nextCursorField)}).
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
		log.Printf("Fetched %d product var", len(apiResponse.Data.ProductVariant.Nodes))
		if len(apiResponse.Data.ProductVariant.Nodes) == 250 {
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

	// Prepare the rows
	if apiResponse.Errors != nil {
		return nil, true, fmt.Errorf("failed to fetch orders: %v", apiResponse.Errors)
	}

	// Update the cursor
	if apiResponse.Data.ProductVariant.PageInfo.HasNextPage {
		t.nextCursor = apiResponse.Data.ProductVariant.PageInfo.EndCursor
	} else {
		t.nextCursor = ""
	}

	// Extract the rows
	for _, node := range apiResponse.Data.ProductVariant.Nodes {
		product := node.Product
		rows = append(rows, []interface{}{
			node.ID,
			node.Barcode,
			node.CreatedAt,
			node.UpdatedAt,
			node.DisplayName,
			node.Sku,
			node.Title,
			node.AvailableForSale,
			node.InventoryQuantity,
			node.Position,
			node.SellableOnlineQuantity,
			product.ID,
			product.Title,
			product.Vendor,
			product.ProductType,
			product.CreatedAt,
			product.UpdatedAt,
			product.Status,
			product.Description,
			product.DescriptionHTML,
			product.OnlineStoreURL,
			product.OnlineStorePreviewURL,
			product.TotalInventory,
		})
	}

	return rows, t.nextCursor == "" || len(rows) < 250, nil

}

// Create a new cursor that will be used to read rows
func (t *products_variantTable) CreateReader() rpc.ReaderInterface {
	return &products_variantCursor{
		cache:      t.cache,
		token:      t.token,
		rateLimit:  t.rateLimit,
		storeName:  t.storeName,
		nextCursor: "",
	}
}

// A slice of rows to insert
func (t *products_variantTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *products_variantTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *products_variantTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *products_variantTable) Close() error {
	return nil
}
