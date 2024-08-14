package main

type Contacts struct {
	Connections   []Connection `json:"connections"`
	TotalPeople   int64        `json:"totalPeople"`
	TotalItems    int64        `json:"totalItems"`
	NextPageToken string       `json:"nextPageToken"`
}

type Connection struct {
	ResourceName   string             `json:"resourceName"`
	Etag           string             `json:"etag"`
	Metadata       ConnectionMetadata `json:"metadata"`
	Names          []Name             `json:"names,omitempty"`
	Photos         []Photo            `json:"photos"`
	EmailAddresses []EmailAddress     `json:"emailAddresses,omitempty"`
	PhoneNumbers   []EmailAddress     `json:"phoneNumbers,omitempty"`
	Birthdays      []Birthday         `json:"birthdays,omitempty"`
	Organizations  []Organization     `json:"organizations,omitempty"`
	Urls           []URL              `json:"urls,omitempty"`
	ClientData     []ClientDatum      `json:"clientData,omitempty"`
	Addresses      []Address          `json:"addresses,omitempty"`
	Relations      []Relation         `json:"relations,omitempty"`
	UserDefined    []ClientDatum      `json:"userDefined,omitempty"`
}

type Address struct {
	Metadata       AddressMetadata `json:"metadata"`
	FormattedValue string          `json:"formattedValue"`
	Type           *string         `json:"type,omitempty"`
	FormattedType  *string         `json:"formattedType,omitempty"`
	StreetAddress  string          `json:"streetAddress"`
	City           string          `json:"city"`
	PostalCode     string          `json:"postalCode"`
	Country        string          `json:"country"`
	CountryCode    *string         `json:"countryCode,omitempty"`
}

type AddressMetadata struct {
	Primary       *bool        `json:"primary,omitempty"`
	Source        PurpleSource `json:"source"`
	SourcePrimary *bool        `json:"sourcePrimary,omitempty"`
}

type PurpleSource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Birthday struct {
	Metadata BirthdayMetadata `json:"metadata"`
	Date     Date             `json:"date"`
}

type Date struct {
	Year  *int64 `json:"year,omitempty"`
	Month int64  `json:"month"`
	Day   int64  `json:"day"`
}

type BirthdayMetadata struct {
	Primary bool         `json:"primary"`
	Source  PurpleSource `json:"source"`
}

type ClientDatum struct {
	Metadata BirthdayMetadata `json:"metadata"`
	Key      string           `json:"key"`
	Value    string           `json:"value"`
}

type EmailAddress struct {
	Metadata      AddressMetadata `json:"metadata"`
	Value         string          `json:"value"`
	Type          *string         `json:"type,omitempty"`
	FormattedType *string         `json:"formattedType,omitempty"`
	CanonicalForm *string         `json:"canonicalForm,omitempty"`
}

type ConnectionMetadata struct {
	Sources    []SourceElement `json:"sources"`
	ObjectType string          `json:"objectType"`
}

type SourceElement struct {
	Type            string           `json:"type"`
	ID              string           `json:"id"`
	Etag            string           `json:"etag"`
	UpdateTime      string           `json:"updateTime"`
	ProfileMetadata *ProfileMetadata `json:"profileMetadata,omitempty"`
}

type ProfileMetadata struct {
	ObjectType string   `json:"objectType"`
	UserTypes  []string `json:"userTypes"`
}

type Name struct {
	Metadata             AddressMetadata `json:"metadata"`
	DisplayName          string          `json:"displayName"`
	FamilyName           *string         `json:"familyName,omitempty"`
	GivenName            string          `json:"givenName"`
	DisplayNameLastFirst string          `json:"displayNameLastFirst"`
	UnstructuredName     string          `json:"unstructuredName"`
}

type Organization struct {
	Metadata BirthdayMetadata `json:"metadata"`
	Name     string           `json:"name"`
	Title    *string          `json:"title,omitempty"`
}

type Photo struct {
	Metadata BirthdayMetadata `json:"metadata"`
	URL      string           `json:"url"`
	Default  *bool            `json:"default,omitempty"`
}

type Relation struct {
	Metadata      BirthdayMetadata `json:"metadata"`
	Person        string           `json:"person"`
	Type          string           `json:"type"`
	FormattedType string           `json:"formattedType"`
}

type URL struct {
	Metadata      AddressMetadata `json:"metadata"`
	Value         string          `json:"value"`
	Type          string          `json:"type"`
	FormattedType string          `json:"formattedType"`
}
