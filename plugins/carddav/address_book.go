package main

import (
	"context"
	"fmt"

	"github.com/emersion/go-webdav/carddav"
	"github.com/julien040/anyquery/rpc"
)

// Column indices for address_books table
const (
	addrBookColPath = iota
	addrBookColName
	addrBookColDescription
	addrBookColMaxResourceSize

	// count
	addrBookColCount
)

var addressBookSchema = []rpc.DatabaseSchemaColumn{
	addrBookColPath: {
		Name:        "path",
		Type:        rpc.ColumnTypeString,
		Description: "Address book path (use this for contacts queries)",
	},
	addrBookColName: {
		Name:        "name",
		Type:        rpc.ColumnTypeString,
		Description: "Display name of the address book",
	},
	addrBookColDescription: {
		Name:        "description",
		Type:        rpc.ColumnTypeString,
		Description: "Description of the address book",
	},
	addrBookColMaxResourceSize: {
		Name:        "max_resource_size",
		Type:        rpc.ColumnTypeInt,
		Description: "Maximum resource size",
	},
}

func addressBooksCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	client, err := newCardDAVClient(args.UserConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CardDAV client: %w", err)
	}

	return &addressBooksTable{client: client}, &rpc.DatabaseSchema{
		Columns: addressBookSchema,
	}, nil
}

type addressBooksTable struct {
	client *carddav.Client
}

type addressBooksCursor struct {
	tbl *addressBooksTable
}

func (t *addressBooksTable) CreateReader() rpc.ReaderInterface {
	return &addressBooksCursor{tbl: t}
}

func (t *addressBooksTable) Close() error {
	return nil
}

func (c *addressBooksCursor) Query(constraints rpc.QueryConstraint) ([][]any, bool, error) {
	ctx := context.Background()

	var addressBooks []carddav.AddressBook
	var err error

	// Method 1: Try standard CardDAV discovery
	principal, err := c.tbl.client.FindAddressBookHomeSet(ctx, "")
	if err == nil {
		addressBooks, err = c.tbl.client.FindAddressBooks(ctx, principal)
		if err == nil && len(addressBooks) > 0 {
			// Success! Found address books via discovery
		} else {
			// Method 2: Try finding address books from root
			addressBooks, err = c.tbl.client.FindAddressBooks(ctx, "/")
			if err != nil || len(addressBooks) == 0 {
				// Method 3: Try finding address books from the current user's principal
				userPrincipal, userErr := c.tbl.client.FindCurrentUserPrincipal(ctx)
				if userErr == nil {
					homeSet, homeErr := c.tbl.client.FindAddressBookHomeSet(ctx, userPrincipal)
					if homeErr == nil {
						addressBooks, err = c.tbl.client.FindAddressBooks(ctx, homeSet)
					}
				}
			}
		}
	} else {
		// Method 2: Try finding address books from root
		addressBooks, err = c.tbl.client.FindAddressBooks(ctx, "/")
		if err != nil || len(addressBooks) == 0 {
			// Method 3: Try finding address books from the current user's principal
			userPrincipal, userErr := c.tbl.client.FindCurrentUserPrincipal(ctx)
			if userErr == nil {
				homeSet, homeErr := c.tbl.client.FindAddressBookHomeSet(ctx, userPrincipal)
				if homeErr == nil {
					addressBooks, err = c.tbl.client.FindAddressBooks(ctx, homeSet)
				}
			}
		}
	}

	// If all discovery methods failed, return an error
	if len(addressBooks) == 0 {
		return nil, true, fmt.Errorf("failed to discover address books using multiple methods. Check your CardDAV URL and credentials")
	}

	rows := make([][]any, len(addressBooks))
	for i, book := range addressBooks {
		row := make([]any, len(addressBookSchema))
		row[addrBookColPath] = book.Path
		row[addrBookColName] = book.Name
		row[addrBookColDescription] = book.Description
		row[addrBookColMaxResourceSize] = book.MaxResourceSize
		rows[i] = row
	}

	return rows, true, nil
}
