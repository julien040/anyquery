package main

import (
	"context"
	"testing"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/julien040/anyquery/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test contacts query
func TestContactsQuery(t *testing.T) {
	server, _ := setupTestServer(t)
	contactsTable, schema := createTestPlugin(t, server.URL)
	defer contactsTable.Close()

	// Verify schema
	assert.Len(t, schema.Columns, colCount, "Schema column count mismatch")

	// Test parameter requirement
	cursor := contactsTable.CreateReader().(*contactsCursor)
	emptyConstraints := rpc.QueryConstraint{}

	// Should fail without address book parameter
	_, _, err := cursor.Query(emptyConstraints)
	assert.Error(t, err, "Expected error when address_book parameter is missing")

	// Test with proper constraints using the actual cursor
	constraints := rpc.QueryConstraint{
		Columns: []rpc.ColumnConstraint{
			{
				ColumnID: colAddressBook,
				Operator: rpc.OperatorEqual,
				Value:    "/addressbooks/user/personal/",
			},
		},
	}

	rows, eof, err := cursor.Query(constraints)
	require.NoError(t, err, "Failed to query contacts with cursor")

	assert.True(t, eof, "Expected EOF to be true")
	assert.GreaterOrEqual(t, len(rows), 2, "Expected at least 2 contacts")

	// Verify contact row structure and data
	for i, row := range rows {
		assert.Len(t, row, colCount-paramCount, "Row %d column count mismatch", i) // Excluding parameter columns

		uid := row[colUID-paramCount].(string)
		fullName := row[colFullName-paramCount].(string)

		assert.NotEmpty(t, uid, "Row %d UID should not be empty", i)
		assert.NotEmpty(t, fullName, "Row %d full name should not be empty", i)

		// Verify row has proper path structure
		path := row[colPath-paramCount].(string)
		assert.Contains(t, path, ".vcf", "Row %d path should contain .vcf extension", i)
	}
}

// Test contact insertion
func TestContactInsert(t *testing.T) {
	server, backend := setupTestServer(t)
	contactsTable, _ := createTestPlugin(t, server.URL)
	defer contactsTable.Close()

	// Create test row for insertion
	testRow := make([]any, len(contactsSchema))
	testRow[colAddressBook] = "/addressbooks/user/personal/"
	testRow[colUID] = "test-insert"
	testRow[colFullName] = "Test Insert"
	testRow[colEmail] = "test@insert.com"
	testRow[colPhone] = "+1111111111"
	testRow[colOrganization] = "Test Corp"

	// Insert the contact
	err := contactsTable.Insert([][]any{testRow})
	require.NoError(t, err, "Failed to insert contact")

	// Verify it was added to backend
	personalContacts := backend.contacts["/addressbooks/user/personal/"]
	assert.Len(t, personalContacts, 3, "Expected 3 contacts after insert")

	// Verify the inserted contact
	insertedContact, exists := personalContacts["test-insert"]
	require.True(t, exists, "Inserted contact not found in backend")
	assert.Equal(t, "Test Insert", insertedContact.Card.Value(vcard.FieldFormattedName))
	assert.Equal(t, "test@insert.com", insertedContact.Card.Value(vcard.FieldEmail))
}

// Test contact update
func TestContactUpdate(t *testing.T) {
	server, backend := setupTestServer(t)
	contactsTable, _ := createTestPlugin(t, server.URL)
	defer contactsTable.Close()

	// Get existing contact
	originalContact := backend.contacts["/addressbooks/user/personal/"]["test-1"]
	require.NotNil(t, originalContact, "Test contact not found")

	// Create update row
	updateRow := make([]any, len(contactsSchema))
	updateRow[colAddressBook] = "/addressbooks/user/personal/"
	updateRow[colUID] = "test-1" // Primary key
	updateRow[colFullName] = "John Doe Updated"
	updateRow[colEmail] = "john.updated@example.com"
	updateRow[colPhone] = "+9999999999"
	updateRow[colOrganization] = "Updated Corp"

	// first column is primary key that's being updated
	updatedRow := append([]any{"/addressbooks/user/personal/test-1.vcf"}, updateRow...)

	// Update the contact
	err := contactsTable.Update([][]any{updatedRow})
	require.NoError(t, err, "Failed to update contact")

	// Verify the update
	updatedContact := backend.contacts["/addressbooks/user/personal/"]["test-1"]
	require.NotNil(t, updatedContact, "Updated contact not found")

	assert.Equal(t, "John Doe Updated", updatedContact.Card.Value(vcard.FieldFormattedName), "Contact full name was not updated")
	assert.Equal(t, "john.updated@example.com", updatedContact.Card.Value(vcard.FieldEmail), "Contact email was not updated")
	assert.Equal(t, "Updated Corp", updatedContact.Card.Value(vcard.FieldOrganization), "Contact organization was not updated")
}

// Test contact deletion
func TestContactDelete(t *testing.T) {
	server, backend := setupTestServer(t)
	contactsTable, _ := createTestPlugin(t, server.URL)
	defer contactsTable.Close()

	// Verify contact exists before deletion
	require.NotNil(t, backend.contacts["/addressbooks/user/personal/"]["test-2"], "Test contact not found before deletion")

	// Delete is not fully implemented, so test the actual delete functionality
	err := contactsTable.Delete([]any{"/addressbooks/user/personal/test-2.vcf"})
	require.NoError(t, err, "Delete operation should work")

	// Verify contact was deleted
	assert.Nil(t, backend.contacts["/addressbooks/user/personal/"]["test-2"], "Contact was not deleted from backend")
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	server, _ := setupTestServer(t)
	contactsTable, _ := createTestPlugin(t, server.URL)
	defer contactsTable.Close()

	ctx := context.Background()
	query := &carddav.AddressBookQuery{
		DataRequest: carddav.AddressDataRequest{AllProp: true},
	}

	_, err := contactsTable.client.QueryAddressBook(ctx, "/invalid/addressbook/", query)
	assert.Error(t, err, "Expected error for invalid address book")

	// Test insert without address book
	invalidRow := make([]any, len(contactsSchema))
	invalidRow[colUID] = "test-invalid"
	invalidRow[colFullName] = "Invalid Test"
	// Missing address book

	err = contactsTable.Insert([][]any{invalidRow})
	assert.Error(t, err, "Expected error for insert without address book")

	// Test update without UID
	invalidUpdateRow := make([]any, len(contactsSchema))
	invalidUpdateRow[colFullName] = "No UID Test"
	invalidUpdateRow = append([]any{colAddressBook: "/addressbooks/user/personal/"}, invalidUpdateRow...)
	// Missing UID

	err = contactsTable.Update([][]any{invalidUpdateRow})
	assert.Error(t, err, "Expected error for update without UID")
}

// Test vCard parsing functionality
func TestVCardParsing(t *testing.T) {
	// Create a test contact
	contact := createTestContact("test-parse", "Dr. John Q. Doe Jr.", "john.doe@example.com", "+1-555-123-4567")

	// Parse to row
	row := parseVCardToRow(contact)

	// Verify row length
	assert.Len(t, row, colCount, "Row column count mismatch")

	// Verify specific fields
	assert.Equal(t, "test-parse", row[colUID].(string), "UID mismatch")
	assert.Equal(t, "Dr. John Q. Doe Jr.", row[colFullName].(string), "Full name mismatch")
	assert.Equal(t, "john.doe@example.com", row[colEmail].(string), "Email mismatch")
	assert.Equal(t, "+1-555-123-4567", row[colPhone].(string), "Phone mismatch")

	newCard := make(vcard.Card)

	// Test round-trip: row to vCard
	err := updateVCardFromRow(newCard, row)
	require.NoError(t, err, "Failed to create vCard from row")

	// Verify round-trip fields
	assert.Equal(t, "Dr. John Q. Doe Jr.", newCard.Value(vcard.FieldFormattedName), "Round-trip full name mismatch")
	assert.Equal(t, "john.doe@example.com", newCard.PreferredValue(vcard.FieldEmail), "Round-trip email mismatch")
	assert.Equal(t, "+1-555-123-4567", newCard.Value(vcard.FieldTelephone), "Round-trip phone mismatch")
}
