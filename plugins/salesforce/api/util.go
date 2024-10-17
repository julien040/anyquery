package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/julien040/anyquery/rpc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Get an HTTP client that is authenticated for the Salesforce REST API
//
// It gets the required information from the plugin configuration,
// exchange tokens using Oauth2 and returns an HTTP client that is authenticated
//
// Supported configurations:
//
//   - access_token: Use the token directly until it expires
//   - client_id, client_secret, refresh_token: Use the refresh token to get a new access token, and refresh it when it expires
//   - username, password, client_id, client_secret: Use the username and password to get a new access token, and refresh it when it expires
//   - client_id, client_secret: Use the client_id as a consumer key and client_secret as a consumer secret to get a new access token, and refresh it when it expires
func GetAuthHTTPClient(conf rpc.PluginConfig) (*http.Client, error) {
	if conf.GetString("domain") == "" {
		return nil, fmt.Errorf("domain must be set in the plugin configuration")
	}

	oauthConfig := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("https://%s/services/oauth2/authorize", conf.GetString("domain")),
			TokenURL: fmt.Sprintf("https://%s/services/oauth2/token", conf.GetString("domain")),
		},
	}

	switch {
	// Access token grant type
	case conf.GetString("access_token") != "":
		return oauthConfig.Client(context.Background(), &oauth2.Token{
			AccessToken: conf.GetString("access_token"),
		}), nil

	// Refresh token grant type
	case conf.GetString("client_id") != "" && conf.GetString("client_secret") != "" && conf.GetString("refresh_token") != "":
		oauthConfig.ClientID = conf.GetString("client_id")
		oauthConfig.ClientSecret = conf.GetString("client_secret")
		return oauthConfig.Client(context.Background(), &oauth2.Token{
			RefreshToken: conf.GetString("refresh_token"),
		}), nil

	// Username and password grant type
	case conf.GetString("client_id") != "" && conf.GetString("client_secret") != "" && conf.GetString("username") != "" && conf.GetString("password") != "":
		oauthConfig.ClientID = conf.GetString("client_id")
		oauthConfig.ClientSecret = conf.GetString("client_secret")
		token, err := oauthConfig.PasswordCredentialsToken(context.Background(), conf.GetString("username"), conf.GetString("password"))
		if err != nil {
			return nil, fmt.Errorf("unable to get token using a username and password: %w", err)
		}
		return oauthConfig.Client(context.Background(), token), nil

	// Client credentials grant type
	case conf.GetString("client_id") != "" && conf.GetString("client_secret") != "":
		clientCred := clientcredentials.Config{
			ClientID:     conf.GetString("client_id"),
			ClientSecret: conf.GetString("client_secret"),
			TokenURL:     oauthConfig.Endpoint.TokenURL,
			AuthStyle:    oauth2.AuthStyleInParams,
		}
		return clientCred.Client(context.Background()), nil

	}

	return nil, fmt.Errorf("invalid configuration. Refer to the documentation for supported configurations")
}

type Column struct {
	Name                 string
	Type                 rpc.ColumnType
	SalesforceType       string
	SalesforceFilterable bool
	SalesforceUpdateable bool
	Index                int
}

// Map a column name to column information
type ColMapper map[string]Column

type ColIndex map[int]string

// InspectSchema an sObject definition and return column information, a table schema and an error if any
func InspectSchema(client *resty.Client, sObject string, domain string) (ColMapper, ColIndex, []rpc.DatabaseSchemaColumn, error) {
	// Get the sObject metadata
	endpoint := fmt.Sprintf("https://%s/services/data/v61.0/sobjects/%s/describe", domain, sObject)
	bodyResp := &DescribeResp{}
	resp, err := client.R().SetResult(bodyResp).Get(endpoint)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to get sObject metadata: %w", err)
	}
	if resp.IsError() {
		return nil, nil, nil, fmt.Errorf("unable to get sObject metadata(%d): %s", resp.StatusCode(), resp.String())
	}

	// Map the columns
	cols := make(ColMapper)
	colIndex := make(ColIndex)
	schema := make([]rpc.DatabaseSchemaColumn, 0, len(bodyResp.Fields))
	schema = append(schema, rpc.DatabaseSchemaColumn{
		Name:        "Id",
		Type:        rpc.ColumnTypeString,
		Description: "The unique identifier for the object",
	})
	colIndex[0] = "Id"
	cols["Id"] = Column{
		Name:  "Id",
		Type:  rpc.ColumnTypeString,
		Index: 0,
	}
	j := 1
	for _, field := range bodyResp.Fields {
		// Because the Id field is already added
		if field.Name == "Id" {
			continue
		}
		colIndex[j] = field.Name
		switch field.Type {
		case "double", "percent", "currency", "anyType", "calculated", "int", "long":
			cols[field.Name] = Column{
				Name:                 field.Name,
				Type:                 rpc.ColumnTypeFloat,
				SalesforceType:       field.Type,
				SalesforceFilterable: field.Filterable,
				SalesforceUpdateable: field.Updateable,
				Index:                j,
			}
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: field.Name,
				Type: rpc.ColumnTypeFloat,
			})
		case "boolean":
			cols[field.Name] = Column{
				Name:                 field.Name,
				Type:                 rpc.ColumnTypeBool,
				SalesforceType:       field.Type,
				SalesforceFilterable: field.Filterable,
				SalesforceUpdateable: field.Updateable,
				Index:                j,
			}
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: field.Name,
				Type: rpc.ColumnTypeBool,
			})
		default:
			cols[field.Name] = Column{
				Name:                 field.Name,
				Type:                 rpc.ColumnTypeString,
				SalesforceType:       field.Type,
				SalesforceFilterable: field.Filterable,
				SalesforceUpdateable: field.Updateable,
				Index:                j,
			}
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: field.Name,
				Type: rpc.ColumnTypeString,
			})
		}
		j++
	}

	return cols, colIndex, schema, nil
}
