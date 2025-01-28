package main

import (
	"context"
	"encoding/json"
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
func google_contactsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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

	return &google_contactsTable{
			srv: srv,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the contact",
				},
				{
					Name:        "addresses",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of The whole address of the contact",
				},
				{
					Name:        "age_range",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of The age range of the contact. One of 'AGE_RANGE_UNSPECIFIED', 'LESS_THAN_EIGHTEEN', 'EIGHTEEN_TO_TWENTY', 'TWENTY_ONE_OR_OLDER'",
				},
				{
					Name:        "biographies",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of A description written by you for the contact",
				},
				{
					Name:        "birthdays",
					Type:        rpc.ColumnTypeDate,
					Description: "A json array of The birthday of the contact in YYYY-MM-DD format",
				},
				{
					Name:        "calendar_urls",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of An URL to the iCal file for the contact's calendar",
				},
				{
					Name:        "client_data",
					Type:        rpc.ColumnTypeJSON,
					Description: "A map of custom data for the contact",
				},
				{
					Name:        "cover_photos",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of The URL of the cover photo of the contact",
				},
				{
					Name:        "email_addresses",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of The email address of the contact",
				},
				{
					Name:        "events",
					Type:        rpc.ColumnTypeString,
					Description: "A map of events for the contact. EventName => YYYY-MM-DD",
				},
				{
					Name:        "gender",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of The gender of the contact",
				},
				{
					Name: "im_clients",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "interests",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of the interests of the contact",
				},
				{
					Name: "locales",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "locations",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of The location of the contact",
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
					Name:        "organizations",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of the organization of the contact",
				},
				{
					Name:        "phone_numbers",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of the phone numbers of the contact",
				},
				{
					Name:        "photos",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of the URLs of the photo of the contact",
				},

				{
					Name:        "relations",
					Type:        rpc.ColumnTypeJSON,
					Description: "A map of relations for the contact. RelationType => PersonName",
				},
				{
					Name: "sip_addresses",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "skills",
					Type:        rpc.ColumnTypeString,
					Description: "A json array of the skill of the contact",
				},
				{
					Name: "urls",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "user_defined",
					Type:        rpc.ColumnTypeJSON,
					Description: "A map of user defined fields for the contact",
				},
			},
		}, nil
}

type google_contactsTable struct {
	srv *people.Service
}

type google_contactsCursor struct {
	nextPage string
	srv      *people.Service
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *google_contactsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

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
		adresses := []string{}
		for _, address := range contact.Addresses {
			adresses = append(adresses, address.FormattedValue)
		}

		ageRanges := []string{}
		for _, ageRange := range contact.AgeRanges {
			ageRanges = append(ageRanges, ageRange.AgeRange)
		}

		biographies := []string{}
		for _, biography := range contact.Biographies {
			biographies = append(biographies, biography.Value)
		}

		birthdays := []string{}
		for _, birthday := range contact.Birthdays {
			if birthday != nil && birthday.Date != nil {
				// Format the date as YYYY-MM-DD
				birthdays = append(birthdays, fmt.Sprintf("%d-%d-%d", birthday.Date.Year, birthday.Date.Month, birthday.Date.Day))
			}
		}

		calendarUrls := []string{}
		for _, calendarUrl := range contact.CalendarUrls {
			calendarUrls = append(calendarUrls, calendarUrl.Url)
		}

		clientData := map[string]string{}
		for _, data := range contact.ClientData {
			clientData[data.Key] = data.Value
		}

		coverPhotos := []string{}
		for _, coverPhoto := range contact.CoverPhotos {
			coverPhotos = append(coverPhotos, coverPhoto.Url)
		}

		emailAddresses := []string{}
		for _, emailAddress := range contact.EmailAddresses {
			emailAddresses = append(emailAddresses, emailAddress.Value)
		}

		events := map[string]string{}
		for _, event := range contact.Events {
			// Format the date as YYYY-MM-DD
			events[event.Type] = fmt.Sprintf("%d-%d-%d", event.Date.Year, event.Date.Month, event.Date.Day)
		}

		genders := []string{}
		for _, gender := range contact.Genders {
			genders = append(genders, gender.Value)
		}

		imClients := map[string]string{}
		for _, imClient := range contact.ImClients {
			imClients[imClient.Protocol] = imClient.Username
		}

		interests := []string{}
		for _, interest := range contact.Interests {
			interests = append(interests, interest.Value)
		}

		locales := []string{}
		for _, locale := range contact.Locales {
			locales = append(locales, locale.Value)
		}

		locations := []string{}
		for _, location := range contact.Locations {
			locations = append(locations, location.Value)
		}

		names := []string{}
		for _, name := range contact.Names {
			names = append(names, name.DisplayName)
		}

		nicknames := []string{}
		for _, nickname := range contact.Nicknames {
			nicknames = append(nicknames, nickname.Value)
		}

		occupations := []string{}
		for _, occupation := range contact.Occupations {
			occupations = append(occupations, occupation.Value)
		}

		organizations := []string{}
		for _, organization := range contact.Organizations {
			organizations = append(organizations, organization.Name)
		}

		phoneNumbers := []string{}
		for _, phoneNumber := range contact.PhoneNumbers {
			phoneNumbers = append(phoneNumbers, phoneNumber.Value)
		}

		photos := []string{}
		for _, photo := range contact.Photos {
			photos = append(photos, photo.Url)
		}

		relations := map[string]string{}
		for _, relation := range contact.Relations {
			relations[relation.Type] = relation.Person
		}

		sipAddresses := map[string]string{}
		for _, sipAddress := range contact.SipAddresses {
			sipAddresses[sipAddress.Type] = sipAddress.Value
		}

		skills := []string{}
		for _, skill := range contact.Skills {
			skills = append(skills, skill.Value)
		}

		urls := []string{}
		for _, url := range contact.Urls {
			urls = append(urls, url.Value)
		}

		userDefined := map[string]string{}
		for _, value := range contact.UserDefined {
			userDefined[value.Key] = value.Value
		}

		rows = append(rows, []interface{}{
			contact.ResourceName,
			serializeJSON(adresses),
			serializeJSON(ageRanges),
			serializeJSON(biographies),
			serializeJSON(birthdays),
			serializeJSON(calendarUrls),
			serializeJSON(clientData),
			serializeJSON(coverPhotos),
			serializeJSON(emailAddresses),
			serializeJSON(events),
			serializeJSON(genders),
			serializeJSON(imClients),
			serializeJSON(interests),
			serializeJSON(locales),
			serializeJSON(locations),
			serializeJSON(names),
			serializeJSON(nicknames),
			serializeJSON(occupations),
			serializeJSON(organizations),
			serializeJSON(phoneNumbers),
			serializeJSON(photos),
			serializeJSON(relations),
			serializeJSON(sipAddresses),
			serializeJSON(skills),
			serializeJSON(urls),
			serializeJSON(userDefined),
		})
	}

	// Check if there are more pages
	t.nextPage = connections.NextPageToken

	return rows, t.nextPage == "", nil
}

// Create a new cursor that will be used to read rows
func (t *google_contactsTable) CreateReader() rpc.ReaderInterface {
	return &google_contactsCursor{
		srv: t.srv,
	}
}

// A destructor to clean up resources
func (t *google_contactsTable) Close() error {
	return nil
}

// Serialize a value to JSON and return nil in case of an error
// or if the value is nil or empty
func serializeJSON(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	// Check if the value is a string slice
	if s, ok := v.([]string); ok {
		if len(s) == 0 {
			return nil
		}
	}

	// Check if the value is a map
	if m, ok := v.(map[string]string); ok {
		if len(m) == 0 {
			return nil
		}
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(b)
}
