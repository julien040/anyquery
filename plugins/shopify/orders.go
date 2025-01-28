package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

const orderTokenCost = 13

const graphQlOrders = `
query {
  orders(first: 250 %s) {
	pageInfo {
    		endCursor
    		hasNextPage
    		hasPreviousPage
    		startCursor
  	}
	nodes{
        id
        unpaid
        confirmed
        displayFinancialStatus
        displayFulfillmentStatus
        email
        fulfillable
        fullyPaid
        note
        requiresShipping
        totalWeight

        
       	totalPriceSet {
           presentmentMoney {
             amount
           }
         }
         currentTotalPriceSet {
           presentmentMoney {
             amount
           }
         }
           
        totalDiscountsSet {
           presentmentMoney {
             amount
           }
         }
        

        
        returnStatus
        requiresShipping
        name
        processedAt
        createdAt
        updatedAt

      }
    }
  
}
`

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func (r *rateLimit) ordersCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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
	db, err := openCache("orders", token)
	if err != nil {
		return nil, nil, err
	}

	return &ordersTable{
			cache:     db,
			token:     token,
			storeName: storeName,
			rateLimit: r,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the order",
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "financial_status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "fulfillment_status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "return_status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "consumer_email",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeDateTime,
				},
				{
					Name: "processed_at",
					Type: rpc.ColumnTypeDateTime,
				},
				{
					Name: "updated_at",
					Type: rpc.ColumnTypeDateTime,
				},
				{
					Name: "unpaid",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "confirmed",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "fulfillable",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "fully_paid",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "requires_shipping",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "total_weight",
					Type: rpc.ColumnTypeFloat,
				},
				{
					Name: "total_price",
					Type: rpc.ColumnTypeFloat,
				},
				{
					Name: "current_total_price",
					Type: rpc.ColumnTypeFloat,
				},
				{
					Name: "total_discounts",
					Type: rpc.ColumnTypeFloat,
				},
			},
		}, nil
}

type ordersTable struct {
	cache     *badger.DB
	token     string
	storeName string
	rateLimit *rateLimit
}

type ordersCursor struct {
	cache      *badger.DB
	token      string
	nextCursor string
	storeName  string
	rateLimit  *rateLimit
}

// Create a new cursor that will be used to read rows
func (t *ordersTable) CreateReader() rpc.ReaderInterface {
	return &ordersCursor{
		cache:      t.cache,
		token:      t.token,
		rateLimit:  t.rateLimit,
		storeName:  t.storeName,
		nextCursor: "",
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *ordersCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	// Get from the cache if possible
	cacheKey := fmt.Sprintf("orders_%s", t.nextCursor)
	var rows [][]interface{}
	apiResponse := &OrderResponse{}

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
		err = t.rateLimit.Wait(t.token, orderTokenCost)
		if err != nil {
			return nil, true, err
		}

		nextCursorField := ""
		if t.nextCursor != "" {
			nextCursorField = fmt.Sprintf("after: \"%s\"", t.nextCursor)
		}

		res, err := client.R().SetHeader("X-Shopify-Access-Token", t.token).
			SetHeader("Content-Type", "application/json").
			SetBody(&GraphqlQuery{Query: fmt.Sprintf(graphQlOrders, nextCursorField)}).
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
		if len(apiResponse.Data.Orders.Nodes) == 250 {
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

	// Extract the rows
	for _, node := range apiResponse.Data.Orders.Nodes {
		email := interface{}(nil)
		if node.Email != nil {
			email = *node.Email
		}

		rows = append(rows, []interface{}{
			node.ID,
			node.Name,
			node.DisplayFinancialStatus,
			node.DisplayFulfillmentStatus,
			node.ReturnStatus,
			email,
			node.CreatedAt,
			node.ProcessedAt,
			node.UpdatedAt,
			node.Unpaid,
			node.Confirmed,
			node.Fulfillable,
			node.FullyPaid,
			node.RequiresShipping,
			node.TotalWeight,
			// Convert the string to a float and return nil if invalid
			convertStrToFloat(node.TotalPriceSet.PresentmentMoney.Amount),
			convertStrToFloat(node.CurrentTotalPriceSet.PresentmentMoney.Amount),
			convertStrToFloat(node.TotalDiscountsSet.PresentmentMoney.Amount),
		})
	}

	// Update the cursor
	if apiResponse.Data.Orders.PageInfo.HasNextPage {
		t.nextCursor = apiResponse.Data.Orders.PageInfo.EndCursor
	} else {
		t.nextCursor = ""
	}

	return rows, t.nextCursor == "" || len(apiResponse.Data.Orders.Nodes) < 250, nil

}

// A destructor to clean up resources
func (t *ordersTable) Close() error {
	return nil
}
