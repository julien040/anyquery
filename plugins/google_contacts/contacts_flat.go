package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func google_contactsFlatCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	var token, clientID, clientSecret string

	if rawInter, ok := args.UserConfig["token"]; ok {
		if token, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("token should be a string")
		}
		if token == "" {
			return nil, nil, fmt.Errorf("token should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("token is required")
	}

	if rawInter, ok := args.UserConfig["client_id"]; ok {
		if clientID, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("client_id should be a string")
		}
		if clientID == "" {
			return nil, nil, fmt.Errorf("client_id should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("client_id is required")
	}

	if rawInter, ok := args.UserConfig["client_secret"]; ok {
		if clientSecret, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("client_secret should be a string")
		}
		if clientSecret == "" {
			return nil, nil, fmt.Errorf("client_secret should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("client_secret is required")
	}

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{people.ContactsScope},
	}

	oauthClient := config.Client(context.Background(), &oauth2.Token{
		RefreshToken: token,
	})

	retryableClient := retryablehttp.NewClient()
	retryableClient.HTTPClient = oauthClient

	srv, err := people.NewService(context.Background(), option.WithHTTPClient(retryableClient.StandardClient()))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create people service: %w", err)
	}

	return &google_contacts_flatTable{
			srv: srv,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "addresses",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "age_range",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "biographies",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "birthdays",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "calendar_urls",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "client_data",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "cover_photos",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "email_addresses",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "events",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "gender",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "im_clients",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "interests",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "locales",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "locations",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "names",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "nicknames",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "occupations",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "organizations",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "phone_numbers",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "photos",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "relations",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "sip_addresses",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "skills",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "urls",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "user_defined",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type google_contacts_flatTable struct {
	srv *people.Service
}

type google_contacts_flatCursor struct {
	nextPage string
	srv      *people.Service
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *google_contacts_flatCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	// Create a new request to get the contacts
	req := t.srv.People.Connections.List("people/me")
	req.PersonFields("addresses,ageRanges,biographies,birthdays,calendarUrls,clientData,coverPhotos,emailAddresses,events," +
		"externalIds,genders,imClients,interests,locales,locations,memberships,names,nicknames," +
		"occupations,organizations,phoneNumbers,photos,relations,sipAddresses,skills,urls,userDefined")
	req.PageSize(1000)

	if t.nextPage != "" {
		req.PageToken(t.nextPage)
	}

	// Get the contacts
	connections, err := req.Do()
	if err != nil {
		return nil, true, fmt.Errorf("unable to fetch contacts: %w", err)
	}

	// Prepare the rows
	rows := make([][]interface{}, 0, len(connections.Connections))
	for _, contact := range connections.Connections {
		adress := interface{}(nil)
		if len(contact.Addresses) > 0 {
			adress = contact.Addresses[0].FormattedValue
		}

		ageRange := interface{}(nil)
		if len(contact.AgeRanges) > 0 {
			ageRange = contact.AgeRanges[0].AgeRange
		}

		biographies := interface{}(nil)
		if len(contact.Biographies) > 0 {
			biographies = contact.Biographies[0].Value
		}

		birthdays := interface{}(nil)
		if len(contact.Birthdays) > 0 {
			birthdays = fmt.Sprintf("%d-%d-%d", contact.Birthdays[0].Date.Year, contact.Birthdays[0].Date.Month, contact.Birthdays[0].Date.Day)
		}

		calendarUrls := interface{}(nil)
		if len(contact.CalendarUrls) > 0 {
			calendarUrls = contact.CalendarUrls[0].Url
		}

		clientData := map[string]string{}
		for _, data := range contact.ClientData {
			clientData[data.Key] = data.Value
		}

		events := map[string]string{}
		for _, event := range contact.Events {
			// Format the date as YYYY-MM-DD
			events[event.Type] = fmt.Sprintf("%d-%d-%d", event.Date.Year, event.Date.Month, event.Date.Day)
		}

		imClients := map[string]string{}
		for _, imClient := range contact.ImClients {
			imClients[imClient.Protocol] = imClient.Username
		}

		relations := map[string]string{}
		for _, relation := range contact.Relations {
			relations[relation.Type] = relation.Person
		}

		sipAddresses := map[string]string{}
		for _, sipAddress := range contact.SipAddresses {
			sipAddresses[sipAddress.Type] = sipAddress.Value
		}

		userDefined := map[string]string{}
		for _, value := range contact.UserDefined {
			userDefined[value.Key] = value.Value
		}

		coverPhoto := interface{}(nil)
		if len(contact.CoverPhotos) > 0 {
			coverPhoto = contact.CoverPhotos[0].Url
		}

		emailAdresss := interface{}(nil)
		if len(contact.EmailAddresses) > 0 {
			emailAdresss = contact.EmailAddresses[0].Value
		}

		gender := interface{}(nil)
		if len(contact.Genders) > 0 {
			gender = contact.Genders[0].Value
		}

		interests := interface{}(nil)
		if len(contact.Interests) > 0 {
			interests = contact.Interests[0].Value
		}

		locales := interface{}(nil)
		if len(contact.Locales) > 0 {
			locales = contact.Locales[0].Value
		}

		location := interface{}(nil)
		if len(contact.Locations) > 0 {
			location = contact.Locations[0].Value
		}

		name := interface{}(nil)
		if len(contact.Names) > 0 {
			name = contact.Names[0].DisplayName
		}

		nicknames := interface{}(nil)
		if len(contact.Nicknames) > 0 {
			nicknames = contact.Nicknames[0].Value
		}

		occupations := interface{}(nil)
		if len(contact.Occupations) > 0 {
			occupations = contact.Occupations[0].Value
		}

		organizations := interface{}(nil)
		if len(contact.Organizations) > 0 {
			organizations = contact.Organizations[0].Name
		}

		phoneNumbers := interface{}(nil)
		if len(contact.PhoneNumbers) > 0 {
			phoneNumbers = contact.PhoneNumbers[0].Value
		}

		photos := interface{}(nil)
		if len(contact.Photos) > 0 {
			photos = contact.Photos[0].Url
		}

		skills := interface{}(nil)
		if len(contact.Skills) > 0 {
			skills = contact.Skills[0].Value
		}

		urls := interface{}(nil)
		if len(contact.Urls) > 0 {
			urls = contact.Urls[0].Value
		}

		rows = append(rows, []interface{}{
			contact.ResourceName,
			adress,
			ageRange,
			biographies,
			birthdays,
			calendarUrls,
			serializeJSON(clientData),
			coverPhoto,
			emailAdresss,
			serializeJSON(events),
			gender,
			serializeJSON(imClients),
			interests,
			locales,
			location,
			name,
			nicknames,
			occupations,
			organizations,
			phoneNumbers,
			photos,
			serializeJSON(relations),
			serializeJSON(sipAddresses),
			skills,
			urls,
			serializeJSON(userDefined),
		})

	}

	// Check if there are more pages
	t.nextPage = connections.NextPageToken

	return rows, t.nextPage == "", nil
}

// Create a new cursor that will be used to read rows
func (t *google_contacts_flatTable) CreateReader() rpc.ReaderInterface {
	return &google_contacts_flatCursor{
		srv: t.srv,
	}
}

// A slice of rows to insert
func (t *google_contacts_flatTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *google_contacts_flatTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *google_contacts_flatTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *google_contacts_flatTable) Close() error {
	return nil
}
