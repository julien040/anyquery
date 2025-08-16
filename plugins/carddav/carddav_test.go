package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/julien040/anyquery/rpc"
)

// Mock CardDAV backend for testing
type mockCardDAVBackend struct {
	addressBooks map[string]*carddav.AddressBook
	contacts     map[string]map[string]*carddav.AddressObject // addressbook -> uid -> contact
}

func newMockBackend() *mockCardDAVBackend {
	backend := &mockCardDAVBackend{
		addressBooks: make(map[string]*carddav.AddressBook),
		contacts:     make(map[string]map[string]*carddav.AddressObject),
	}

	// Create default address books
	backend.addressBooks["/addressbooks/user/personal/"] = &carddav.AddressBook{
		Path:        "/addressbooks/user/personal/",
		Name:        "Personal",
		Description: "Personal contacts",
	}
	backend.addressBooks["/addressbooks/user/work/"] = &carddav.AddressBook{
		Path:        "/addressbooks/user/work/",
		Name:        "Work",
		Description: "Work contacts",
	}

	// Initialize contact maps
	backend.contacts["/addressbooks/user/personal/"] = make(map[string]*carddav.AddressObject)
	backend.contacts["/addressbooks/user/work/"] = make(map[string]*carddav.AddressObject)

	// Add some test contacts
	backend.addTestContact("/addressbooks/user/personal/", createTestContact("test-1", "John Doe", "john@example.com", "+1234567890"))
	backend.addTestContact("/addressbooks/user/personal/", createTestContact("test-2", "Jane Smith", "jane@example.com", "+0987654321"))
	backend.addTestContact("/addressbooks/user/work/", createTestContact("test-3", "Bob Wilson", "bob@company.com", "+5555555555"))

	return backend
}

func createTestContact(uid, fullName, email, phone string) *carddav.AddressObject {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "4.0") // Add required VERSION field
	card.SetValue(vcard.FieldUID, uid)
	card.SetValue(vcard.FieldFormattedName, fullName)

	// Parse name into components
	parts := strings.Split(fullName, " ")
	if len(parts) >= 2 {
		name := &vcard.Name{
			GivenName:  parts[0],
			FamilyName: parts[len(parts)-1],
		}
		card.SetName(name)
	}

	if email != "" {
		card.AddValue(vcard.FieldEmail, email)
	}
	if phone != "" {
		card.AddValue(vcard.FieldTelephone, phone)
	}

	card.SetRevision(time.Now())

	return &carddav.AddressObject{
		Path:    fmt.Sprintf("/addressbooks/user/personal/%s.vcf", uid),
		ModTime: time.Now(),
		ETag:    fmt.Sprintf(`"%s"`, uid),
		Card:    card,
	}
}

func (b *mockCardDAVBackend) addTestContact(addressBookPath string, contact *carddav.AddressObject) {
	if b.contacts[addressBookPath] == nil {
		b.contacts[addressBookPath] = make(map[string]*carddav.AddressObject)
	}
	uid := contact.Card.Value(vcard.FieldUID)
	b.contacts[addressBookPath][uid] = contact
}

// Implement carddav.Backend interface
func (b *mockCardDAVBackend) AddressBookHomeSetPath(ctx context.Context) (string, error) {
	return "/addressbooks/user/", nil
}

func (b *mockCardDAVBackend) ListAddressBooks(ctx context.Context) ([]carddav.AddressBook, error) {
	var books []carddav.AddressBook
	for _, book := range b.addressBooks {
		books = append(books, *book)
	}
	return books, nil
}

func (b *mockCardDAVBackend) GetAddressBook(ctx context.Context, path string) (*carddav.AddressBook, error) {
	book, exists := b.addressBooks[path]
	if !exists {
		return nil, fmt.Errorf("address book not found: %s", path)
	}
	return book, nil
}

func (b *mockCardDAVBackend) CreateAddressBook(ctx context.Context, addressBook *carddav.AddressBook) error {
	b.addressBooks[addressBook.Path] = addressBook
	b.contacts[addressBook.Path] = make(map[string]*carddav.AddressObject)
	return nil
}

func (b *mockCardDAVBackend) DeleteAddressBook(ctx context.Context, path string) error {
	delete(b.addressBooks, path)
	delete(b.contacts, path)
	return nil
}

func (b *mockCardDAVBackend) GetAddressObject(ctx context.Context, path string, req *carddav.AddressDataRequest) (*carddav.AddressObject, error) {
	// Extract address book and contact ID from path
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	contactID := strings.TrimSuffix(parts[len(parts)-1], ".vcf")
	addressBookPath := "/" + strings.Join(parts[1:len(parts)-1], "/") + "/"

	contacts, exists := b.contacts[addressBookPath]
	if !exists {
		return nil, fmt.Errorf("address book not found: %s", addressBookPath)
	}

	contact, exists := contacts[contactID]
	if !exists {
		return nil, fmt.Errorf("contact not found: %s", contactID)
	}

	return contact, nil
}

func (b *mockCardDAVBackend) ListAddressObjects(ctx context.Context, path string, req *carddav.AddressDataRequest) ([]carddav.AddressObject, error) {
	contacts, exists := b.contacts[path]
	if !exists {
		return nil, fmt.Errorf("address book not found: %s", path)
	}

	var objects []carddav.AddressObject
	for _, contact := range contacts {
		objects = append(objects, *contact)
	}
	return objects, nil
}

func (b *mockCardDAVBackend) QueryAddressObjects(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	return b.ListAddressObjects(ctx, path, &query.DataRequest)
}

func (b *mockCardDAVBackend) PutAddressObject(ctx context.Context, path string, card vcard.Card, opts *carddav.PutAddressObjectOptions) (*carddav.AddressObject, error) {
	// Extract address book path - handle double slash issue
	cleanPath := strings.ReplaceAll(path, "//", "/")
	parts := strings.Split(cleanPath, "/")
	addressBookPath := "/" + strings.Join(parts[1:len(parts)-1], "/") + "/"

	contacts, exists := b.contacts[addressBookPath]
	if !exists {
		return nil, fmt.Errorf("address book not found: %s", addressBookPath)
	}

	uid := card.Value(vcard.FieldUID)
	if uid == "" {
		uid = fmt.Sprintf("generated-%d", time.Now().Unix())
		card.SetValue(vcard.FieldUID, uid)
	}

	log.Printf("PutAddressObject: %s %s", addressBookPath, uid)

	// Ensure vCard has VERSION field
	if card.Value(vcard.FieldVersion) == "" {
		card.SetValue(vcard.FieldVersion, "4.0")
	}

	contact := &carddav.AddressObject{
		Path:    cleanPath,
		ModTime: time.Now(),
		ETag:    fmt.Sprintf(`"%s-%d"`, uid, time.Now().Unix()),
		Card:    card,
	}

	contacts[uid] = contact
	return contact, nil
}

func (b *mockCardDAVBackend) DeleteAddressObject(ctx context.Context, path string) error {
	// Extract address book and contact ID from path
	parts := strings.Split(path, "/")
	contactID := strings.TrimSuffix(parts[len(parts)-1], ".vcf")
	addressBookPath := "/" + strings.Join(parts[1:len(parts)-1], "/") + "/"

	contacts, exists := b.contacts[addressBookPath]
	if !exists {
		return fmt.Errorf("address book not found: %s", addressBookPath)
	}

	delete(contacts, contactID)
	return nil
}

// Implement webdav.UserPrincipalBackend interface
func (b *mockCardDAVBackend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	return "/principals/user/", nil
}

// Test helper functions
func setupTestServer(t *testing.T) (*httptest.Server, *mockCardDAVBackend) {
	backend := newMockBackend()
	handler := &carddav.Handler{
		Backend: backend,
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, backend
}

func createTestPlugin(t *testing.T, serverURL string) (*contactsTable, *rpc.DatabaseSchema) {
	config := map[string]any{
		"url":      serverURL,
		"username": "testuser",
		"password": "testpass",
	}

	args := rpc.TableCreatorArgs{
		UserConfig: config,
	}

	table, schema, err := contactsCreator(args)
	if err != nil {
		t.Fatalf("Failed to create contacts table: %v", err)
	}

	contactsTable, ok := table.(*contactsTable)
	if !ok {
		t.Fatalf("Expected *contactsTable, got %T", table)
	}

	return contactsTable, schema
}

// Create a mock HTTP server that properly responds to CardDAV PROPFIND requests
func setupCardDAVMockServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle CardDAV discovery requests
		if r.Method == "PROPFIND" {
			handlePropfind(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func handlePropfind(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Handle different discovery endpoints
	switch {
	case path == "/" || path == "":
		// Root discovery - return current user principal
		respondWithCurrentUserPrincipal(w)
	case strings.HasPrefix(path, "/principals/"):
		// Principal discovery - return address book home set
		respondWithAddressBookHomeSet(w)
	case strings.HasPrefix(path, "/addressbooks/"):
		// Address book discovery - return available address books
		respondWithAddressBooks(w)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func respondWithCurrentUserPrincipal(w http.ResponseWriter) {
	response := `<?xml version="1.0" encoding="UTF-8"?>
<d:multistatus xmlns:d="DAV:">
	<d:response>
		<d:href>/</d:href>
		<d:propstat>
			<d:prop>
				<d:current-user-principal>
					<d:href>/principals/user/</d:href>
				</d:current-user-principal>
			</d:prop>
			<d:status>HTTP/1.1 200 OK</d:status>
		</d:propstat>
	</d:response>
</d:multistatus>`

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusMultiStatus)
	w.Write([]byte(response))
}

func respondWithAddressBookHomeSet(w http.ResponseWriter) {
	response := `<?xml version="1.0" encoding="UTF-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">
	<d:response>
		<d:href>/principals/user/</d:href>
		<d:propstat>
			<d:prop>
				<card:addressbook-home-set>
					<d:href>/addressbooks/user/</d:href>
				</card:addressbook-home-set>
			</d:prop>
			<d:status>HTTP/1.1 200 OK</d:status>
		</d:propstat>
	</d:response>
</d:multistatus>`

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusMultiStatus)
	w.Write([]byte(response))
}

func respondWithAddressBooks(w http.ResponseWriter) {
	response := `<?xml version="1.0" encoding="UTF-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">
	<d:response>
		<d:href>/addressbooks/user/personal/</d:href>
		<d:propstat>
			<d:prop>
				<d:resourcetype>
					<d:collection/>
					<card:addressbook/>
				</d:resourcetype>
				<d:displayname>Personal</d:displayname>
				<card:addressbook-description>Personal contacts</card:addressbook-description>
			</d:prop>
			<d:status>HTTP/1.1 200 OK</d:status>
		</d:propstat>
	</d:response>
	<d:response>
		<d:href>/addressbooks/user/work/</d:href>
		<d:propstat>
			<d:prop>
				<d:resourcetype>
					<d:collection/>
					<card:addressbook/>
				</d:resourcetype>
				<d:displayname>Work</d:displayname>
				<card:addressbook-description>Work contacts</card:addressbook-description>
			</d:prop>
			<d:status>HTTP/1.1 200 OK</d:status>
		</d:propstat>
	</d:response>
</d:multistatus>`

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusMultiStatus)
	w.Write([]byte(response))
}
