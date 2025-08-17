package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/julien040/anyquery/rpc"
)

// Column indices for contacts table schema
const (
	// parameter columns
	colAddressBook = iota
	paramCount     // total number of parameter columns

	// data columns
	colUID = iota - 1
	colETag
	colPath // Primary key
	colFullName
	colGivenName
	colFamilyName
	colMiddleName
	colPrefix
	colSuffix
	colNickname
	colEmail
	colHomeEmail
	colWorkEmail
	colOtherEmail
	colEmails
	colPhone
	colMobilePhone
	colWorkPhone
	colOrganization
	colTitle
	colRole
	colBirthday
	colAnniversary
	colNote
	colURL
	colCategories
	colModifiedAt
	colRawVCard

	colCount // total number of columns (including parameter columns)
)

var contactsSchema = []rpc.DatabaseSchemaColumn{
	// parameter columns
	colAddressBook: {
		Name:        "address_book",
		Type:        rpc.ColumnTypeString,
		IsParameter: true,
		IsRequired:  true,
		Description: "The address book path to query",
	},

	// data columns
	colUID: {
		Name:        "uid",
		Type:        rpc.ColumnTypeString,
		Description: "Unique identifier for the contact",
	},
	colETag: {
		Name:        "etag",
		Type:        rpc.ColumnTypeString,
		Description: "ETag for conflict detection",
	},
	colPath: {
		Name:        "path",
		Type:        rpc.ColumnTypeString,
		Description: "CardDAV resource path",
	},
	colFullName: {
		Name:        "full_name",
		Type:        rpc.ColumnTypeString,
		Description: "Full display name",
	},
	colGivenName: {
		Name:        "given_name",
		Type:        rpc.ColumnTypeString,
		Description: "First name",
	},
	colFamilyName: {
		Name:        "family_name",
		Type:        rpc.ColumnTypeString,
		Description: "Last name",
	},
	colMiddleName: {
		Name:        "middle_name",
		Type:        rpc.ColumnTypeString,
		Description: "Middle name",
	},
	colPrefix: {
		Name:        "prefix",
		Type:        rpc.ColumnTypeString,
		Description: "Name prefix (Mr., Dr., etc.)",
	},
	colSuffix: {
		Name:        "suffix",
		Type:        rpc.ColumnTypeString,
		Description: "Name suffix (Jr., III, etc.)",
	},
	colNickname: {
		Name:        "nickname",
		Type:        rpc.ColumnTypeString,
		Description: "Nickname",
	},
	colEmail: {
		Name:        "email",
		Type:        rpc.ColumnTypeString,
		Description: "Primary email address",
	},
	colHomeEmail: {
		Name:        "home_email",
		Type:        rpc.ColumnTypeString,
		Description: "Home email address",
	},
	colWorkEmail: {
		Name:        "work_email",
		Type:        rpc.ColumnTypeString,
		Description: "Work email address",
	},
	colOtherEmail: {
		Name:        "other_email",
		Type:        rpc.ColumnTypeString,
		Description: "Other email address",
	},
	colEmails: {
		Name:        "emails",
		Type:        rpc.ColumnTypeJSON,
		Description: "Work email address",
	},
	colPhone: {
		Name:        "phone",
		Type:        rpc.ColumnTypeString,
		Description: "Primary phone number",
	},
	colMobilePhone: {
		Name:        "mobile_phone",
		Type:        rpc.ColumnTypeString,
		Description: "Mobile phone number",
	},
	colWorkPhone: {
		Name:        "work_phone",
		Type:        rpc.ColumnTypeString,
		Description: "Work phone number",
	},
	colOrganization: {
		Name:        "organization",
		Type:        rpc.ColumnTypeString,
		Description: "Company/organization name",
	},
	colTitle: {
		Name:        "title",
		Type:        rpc.ColumnTypeString,
		Description: "Job title",
	},
	colRole: {
		Name:        "role",
		Type:        rpc.ColumnTypeString,
		Description: "Job role",
	},
	colBirthday: {
		Name:        "birthday",
		Type:        rpc.ColumnTypeString,
		Description: "Birthday in RFC3339 format",
	},
	colAnniversary: {
		Name:        "anniversary",
		Type:        rpc.ColumnTypeString,
		Description: "Anniversary in RFC3339 format",
	},
	colNote: {
		Name:        "note",
		Type:        rpc.ColumnTypeString,
		Description: "Notes",
	},
	colURL: {
		Name:        "url",
		Type:        rpc.ColumnTypeString,
		Description: "Associated website URL",
	},
	colCategories: {
		Name:        "categories",
		Type:        rpc.ColumnTypeJSON,
		Description: "Categories/tags (comma-separated)",
	},
	colModifiedAt: {
		Name:        "modified_at",
		Type:        rpc.ColumnTypeDateTime,
		Description: "Last modification time (RFC3339)",
	},
	colRawVCard: {
		Name:        "raw_vcard",
		Type:        rpc.ColumnTypeString,
		Description: "Complete vCard data",
	},
}

func contactsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	client, err := newCardDAVClient(args.UserConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CardDAV client: %w", err)
	}

	return &contactsTable{client: client}, &rpc.DatabaseSchema{
		PrimaryKey:    colPath, // we use path as primary key as it directly maps to the CardDAV path
		HandlesInsert: true,
		HandlesUpdate: true,
		HandlesDelete: true,
		Columns:       contactsSchema,
	}, nil
}

type contactsTable struct {
	client *carddav.Client
}

func (t *contactsTable) CreateReader() rpc.ReaderInterface {
	return &contactsCursor{tbl: t}
}

func (t *contactsTable) Close() error {
	return nil
}

type contactsCursor struct{ tbl *contactsTable }

func (c *contactsCursor) Query(constraints rpc.QueryConstraint) ([][]any, bool, error) {
	// Check if address_book constraint exists
	constraint := constraints.GetColumnConstraint(colAddressBook)
	if constraint == nil {
		return nil, true, fmt.Errorf("address_book parameter is required")
	}

	// Get the address book path - empty string is valid for root collection
	addressBook := constraint.GetStringValue()

	ctx := context.Background()

	// Query all contacts from the specified address book
	query := &carddav.AddressBookQuery{
		// Request all contact data
		DataRequest: carddav.AddressDataRequest{
			AllProp: true,
		},
		Limit: constraints.Limit,
	}

	contacts, err := c.tbl.client.QueryAddressBook(ctx, addressBook, query)
	if err != nil {
		return nil, true, fmt.Errorf("failed to query address book '%s': %w", addressBook, err)
	}

	// Convert contacts to rows
	rows := make([][]any, len(contacts))
	for i, contact := range contacts {
		row := parseVCardToRow(&contact)

		// skip parameter columns
		rows[i] = row[paramCount:]
	}

	return rows, true, nil
}

func (t *contactsTable) Insert(rows [][]any) error {
	ctx := context.Background()

	for _, rowCols := range rows {
		// Extract address book from first column (parameter)
		// Empty string is valid for root collection
		addressBook, ok := rowCols[colAddressBook].(string)
		if !ok || addressBook == "" {
			return fmt.Errorf("address book is required for insert")
		}

		card := make(vcard.Card)

		// Set required VERSION field
		card.SetValue(vcard.FieldVersion, "4.0")

		// Set UID if provided, otherwise generate one
		uid := rowCols[colUID].(string)
		if uid == "" {
			uid = fmt.Sprintf("contact-%d", time.Now().Unix())
		}
		card.SetValue(vcard.FieldUID, uid)

		// Create vCard from row data
		if err := updateVCardFromRow(card, rowCols); err != nil {
			return fmt.Errorf("failed to create vCard: %w", err)
		}

		// Generate a path for the new contact
		cleanAddressBook := strings.TrimSuffix(addressBook, "/")
		contactPath := fmt.Sprintf("%s/%s.vcf", cleanAddressBook, uid)

		// Save the contact to the CardDAV server
		_, err := t.client.PutAddressObject(ctx, contactPath, card)
		if err != nil {
			return fmt.Errorf("failed to create contact at '%s': %w", contactPath, err)
		}
	}

	return nil
}

func (t *contactsTable) Update(rows [][]any) error {
	ctx := context.Background()

	log.Printf("Updating %+v rows", rows)

	for _, rowCols := range rows {
		// first column is primary key that's being updated
		contactPath := rowCols[0].(string)
		rowCols = rowCols[1:]

		// GetAddressObject fails for some reason on nextcloud, so we directly open the file and parse it instead
		r, err := t.client.Open(ctx, contactPath)
		if err != nil {
			return fmt.Errorf("failed to open existing contact '%s': %w", contactPath, err)
		}

		card, err := vcard.NewDecoder(r).Decode()
		if err != nil {
			r.Close()
			return fmt.Errorf("failed to decode existing contact '%s': %w", contactPath, err)
		}
		r.Close()

		// Update vCard from row data
		if err := updateVCardFromRow(card, rowCols); err != nil {
			return fmt.Errorf("failed to create vCard: %w", err)
		}

		log.Printf("Updating contact '%s' with '%v'", contactPath, card)

		// TODO: Implement ETag checking for conflict detection
		// For now, we'll just overwrite

		// Save the updated contact
		_, err = t.client.PutAddressObject(ctx, contactPath, card)
		if err != nil {
			return fmt.Errorf("failed to update contact '%s': %w", contactPath, err)
		}
	}

	return nil
}

func (t *contactsTable) Delete(primaryKeys []any) error {
	ctx := context.Background()

	for _, primaryKey := range primaryKeys {
		// first column is primary key that's being deleted
		contactPath := primaryKey.(string)
		log.Printf("Deleting contact '%s'", contactPath)

		err := t.client.RemoveAll(ctx, contactPath)
		if err != nil {
			return fmt.Errorf("failed to delete contact '%s': %w", contactPath, err)
		}
	}

	return nil
}

func parseVCardToRow(addressObj *carddav.AddressObject) []any {
	card := addressObj.Card

	// Parameter columns are not included in row data
	// Use rowCount for the row size and row* constants for indexing
	row := make([]any, len(contactsSchema))

	// Use clean row indices (no parameter columns)
	row[colUID] = card.Value(vcard.FieldUID)
	row[colETag] = addressObj.ETag
	row[colPath] = addressObj.Path
	row[colFullName] = card.Value(vcard.FieldFormattedName)

	// Parse structured name (N field)
	name := card.Name()
	if name != nil {
		row[colGivenName] = name.GivenName
		row[colFamilyName] = name.FamilyName
		row[colMiddleName] = name.AdditionalName
		row[colPrefix] = name.HonorificPrefix
		row[colSuffix] = name.HonorificSuffix
	}

	row[colNickname] = card.Value(vcard.FieldNickname)

	// Parse emails
	if fields, ok := card[vcard.FieldEmail]; ok {
		field := card.Preferred(vcard.FieldEmail)
		if field != nil {
			row[colEmail] = field.Value
		}

		for _, field := range fields {
			typeField := field.Params.Get(vcard.ParamType)
			switch typeField {
			case "HOME":
				row[colHomeEmail] = field.Value
			case "WORK":
				row[colWorkEmail] = field.Value
			case "OTHER":
				row[colOtherEmail] = field.Value
			default:
				if row[colEmail] == "" {
					row[colEmail] = field.Value
				}
			}
		}

		emails, _ := json.Marshal(card.Values(vcard.FieldEmail))
		row[colEmails] = string(emails)
	}

	// Parse phones
	phones := card.Values(vcard.FieldTelephone)
	if len(phones) > 0 {
		row[colPhone] = strings.Join(phones, ",") // First phone becomes primary
		// TODO: Implement proper type checking for mobile/work phones
	}

	row[colOrganization] = card.Value(vcard.FieldOrganization)
	row[colTitle] = card.Value(vcard.FieldTitle)
	row[colRole] = card.Value(vcard.FieldRole)
	row[colBirthday] = card.Value(vcard.FieldBirthday)
	row[colAnniversary] = card.Value(vcard.FieldAnniversary)
	row[colNote] = card.Value(vcard.FieldNote)
	row[colURL] = card.Value(vcard.FieldURL)

	categories, _ := json.Marshal(card.Categories())
	row[colCategories] = string(categories)

	// modified_at (REV field)
	if rev, err := card.Revision(); err == nil {
		row[colModifiedAt] = rev.Format(time.RFC3339)
	} else {
		row[colModifiedAt] = addressObj.ModTime.Format(time.RFC3339)
	}

	// raw_vcard - we need to encode the card back to string
	// For now, we'll use a placeholder
	row[colRawVCard] = ""

	return row
}

func updateVCardFromRow(card vcard.Card, row []any) error {
	// Set formatted name
	if fullName, ok := row[colFullName].(string); ok {
		card.SetValue(vcard.FieldFormattedName, fullName)
	}

	name := card.Name()
	if name == nil {
		name = &vcard.Name{}
	}
	if givenName, ok := row[colGivenName].(string); ok {
		name.GivenName = givenName
	}
	if familyName, ok := row[colFamilyName].(string); ok {
		name.FamilyName = familyName
	}
	if middleName, ok := row[colMiddleName].(string); ok {
		name.AdditionalName = middleName
	}
	if prefix, ok := row[colPrefix].(string); ok {
		name.HonorificPrefix = prefix
	}
	if suffix, ok := row[colSuffix].(string); ok {
		name.HonorificSuffix = suffix
	}

	// only set name if it's not empty
	if *name != (vcard.Name{}) {
		card.SetName(name)
	}

	// Set nickname
	if nickname, ok := row[colNickname].(string); ok {
		card.SetValue(vcard.FieldNickname, nickname)
	}

	// Set primary email emails
	if email, ok := row[colEmail].(string); ok {
		if preferedField := card.Preferred(vcard.FieldEmail); preferedField != nil {
			// found preferred email, update it
			preferedField.Value = email
		} else if idx := slices.IndexFunc(card[vcard.FieldEmail], func(field *vcard.Field) bool {
			return len(field.Params) == 0
		}); idx != -1 {
			// found email without params, update it
			card[vcard.FieldEmail][idx].Value = email
		} else {
			// no email without params, add prefered email
			card.Add(vcard.FieldEmail, &vcard.Field{
				Value: email,
				Params: vcard.Params{
					vcard.ParamPreferred: []string{"1"},
				},
			})
		}
	}
	if homeEmail, ok := row[colHomeEmail].(string); ok {
		updateOrAddCardField(card, vcard.FieldEmail, &vcard.Field{
			Value: homeEmail,
			Params: vcard.Params{
				vcard.ParamType: []string{"HOME"},
			},
		})
	}
	if workEmail, ok := row[colWorkEmail].(string); ok {
		updateOrAddCardField(card, vcard.FieldEmail, &vcard.Field{
			Value: workEmail,
			Params: vcard.Params{
				vcard.ParamType: []string{"WORK"},
			},
		})
	}
	if otherEmail, ok := row[colOtherEmail].(string); ok {
		updateOrAddCardField(card, vcard.FieldEmail, &vcard.Field{
			Value: otherEmail,
			Params: vcard.Params{
				vcard.ParamType: []string{"OTHER"},
			},
		})
	}

	// Set phones
	if phone, ok := row[colPhone].(string); ok && phone != "" {
		updateOrAddCardField(card, vcard.FieldTelephone, &vcard.Field{
			Value: phone,
		})
	}
	if mobilePhone, ok := row[colMobilePhone].(string); ok && mobilePhone != "" {
		updateOrAddCardField(card, vcard.FieldTelephone, &vcard.Field{
			Value: mobilePhone,
			Params: vcard.Params{
				vcard.ParamType: []string{"cell"},
			},
		})
	}
	if workPhone, ok := row[colWorkPhone].(string); ok && workPhone != "" {
		updateOrAddCardField(card, vcard.FieldTelephone, &vcard.Field{
			Value: workPhone,
			Params: vcard.Params{
				vcard.ParamType: []string{"work"},
			},
		})
	}

	// Set other fields
	if organization, ok := row[colOrganization].(string); ok {
		card.SetValue(vcard.FieldOrganization, organization)
	}

	if title, ok := row[colTitle].(string); ok {
		card.SetValue(vcard.FieldTitle, title)
	}

	if role, ok := row[colRole].(string); ok {
		card.SetValue(vcard.FieldRole, role)
	}

	if birthday, ok := row[colBirthday].(string); ok {
		card.SetValue(vcard.FieldBirthday, birthday)
	}

	if anniversary, ok := row[colAnniversary].(string); ok {
		card.SetValue(vcard.FieldAnniversary, anniversary)
	}

	if note, ok := row[colNote].(string); ok {
		card.SetValue(vcard.FieldNote, note)
	}

	if url, ok := row[colURL].(string); ok {
		card.SetValue(vcard.FieldURL, url)
	}

	if categories, ok := row[colCategories].(string); ok {
		categoryList := strings.Split(categories, ",")
		for i, cat := range categoryList {
			categoryList[i] = strings.TrimSpace(cat)
		}
		card.SetCategories(categoryList)
	}

	// Set revision to current time
	card.SetRevision(time.Now())

	return nil
}

func matchesField(field, matchField *vcard.Field) bool {
	for paramName, matchParamValues := range matchField.Params {
		paramValues, ok := field.Params[paramName]
		if !ok {
			return false
		}

		for _, val := range matchParamValues {
			if !slices.Contains(paramValues, val) {
				return false
			}
		}
	}

	return true
}

func updateOrAddCardField(card vcard.Card, name string, newField *vcard.Field) *vcard.Field {
	fieldIndex := slices.IndexFunc(card[name], func(field *vcard.Field) bool {
		return matchesField(field, newField)
	})
	if fieldIndex != -1 {
		field := card[name][fieldIndex]

		// Update field value
		field.Value = newField.Value
		return field
	}

	// Add new field
	card.Add(name, newField)
	return newField
}
