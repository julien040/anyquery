package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

const customerTokenCost = 13

const graphQlCustomer = `
query {
  customers(first: 250 %s) {
	pageInfo {
    		endCursor
    		hasNextPage
    		hasPreviousPage
    		startCursor
  	}
	nodes{
        amountSpent {
          amount
          currencyCode
        }
        createdAt
        updatedAt
        dataSaleOptOut
        displayName
        email
        firstName
        lastName
        id
        locale
        note
        numberOfOrders
        phone
        productSubscriberStatus
        state
        validEmailAddress
        verifiedEmail
        taxExempt
        tags

      }
    }
}`

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func (r *rateLimit) customersCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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
	db, err := openCache("customers", token)
	if err != nil {
		return nil, nil, err
	}
	return &customersTable{
			cache:     db,
			token:     token,
			storeName: storeName,
			rateLimit: r,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the customer",
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
					Name: "display_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "email",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "first_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "last_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "locale",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "note",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "phone",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "subscription_status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "state",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "tags",
					Type: rpc.ColumnTypeJSON,
				},
				{
					Name: "amount_spent",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "amount_spent_currency",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "data_sale_opt_out",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "valid_email",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "verified_email",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "tax_exempt",
					Type: rpc.ColumnTypeBool,
				},
			},
		}, nil
}

type customersTable struct {
	cache     *badger.DB
	token     string
	storeName string
	rateLimit *rateLimit
}

type customersCursor struct {
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
func (t *customersCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get from the cache if possible
	cacheKey := fmt.Sprintf("customer_%s", t.nextCursor)
	var rows [][]interface{}
	apiResponse := &CustomerResponse{}

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
		err = t.rateLimit.Wait(t.token, customerTokenCost)
		if err != nil {
			return nil, true, err
		}

		nextCursorField := ""
		if t.nextCursor != "" {
			nextCursorField = fmt.Sprintf("after: \"%s\"", t.nextCursor)
		}

		res, err := client.R().SetHeader("X-Shopify-Access-Token", t.token).
			SetHeader("Content-Type", "application/json").
			SetBody(&GraphqlQuery{Query: fmt.Sprintf(graphQlCustomer, nextCursorField)}).
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
		if len(apiResponse.Data.Customers.Nodes) == 250 {
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
	if apiResponse.Data.Customers.PageInfo.HasNextPage {
		t.nextCursor = apiResponse.Data.Customers.PageInfo.EndCursor
	} else {
		t.nextCursor = ""
	}

	// Extract the rows
	for _, customer := range apiResponse.Data.Customers.Nodes {
		note := interface{}(nil)
		if customer.Note != nil {
			note = *customer.Note
		}
		rows = append(rows, []interface{}{
			customer.ID,
			customer.CreatedAt,
			customer.UpdatedAt,
			customer.DisplayName,
			customer.Email,
			customer.FirstName,
			customer.LastName,
			customer.Locale,
			note,
			customer.Phone,
			customer.ProductSubscriberStatus,
			customer.State,
			customer.Tags,
			convertStrToFloat(customer.AmountSpent.Amount),
			customer.AmountSpent.CurrencyCode,
			customer.DataSaleOptOut,
			customer.ValidEmailAddress,
			customer.VerifiedEmail,
			customer.TaxExempt,
		})
	}

	return rows, t.nextCursor == "" || len(rows) < 250, nil
}

// Create a new cursor that will be used to read rows
func (t *customersTable) CreateReader() rpc.ReaderInterface {
	return &customersCursor{
		cache:      t.cache,
		token:      t.token,
		rateLimit:  t.rateLimit,
		storeName:  t.storeName,
		nextCursor: "",
	}
}

// A destructor to clean up resources
func (t *customersTable) Close() error {
	return nil
}
