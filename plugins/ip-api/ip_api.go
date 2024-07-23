package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julien040/anyquery/rpc"
)

type Responses struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"query"`
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func ip_apiCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &ip_apiTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "ip",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
			},
			{
				Name: "country",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "country_code",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "region",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "region_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "city",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "zip",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "lat",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "lon",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "timezone",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "isp",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "org",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "org_as",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "query",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

type ip_apiTable struct {
}

type ip_apiCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *ip_apiCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	ip := ""
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 0 {
			str, ok := constraint.Value.(string)
			if !ok {
				return nil, true, fmt.Errorf("ip is not a string")
			}
			ip = str
			break
		}
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	res, err := http.Get(url)
	if err != nil {
		return nil, true, fmt.Errorf("failed to get ip-api response: %s", err)
	}

	defer res.Body.Close()

	var response Responses
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, true, fmt.Errorf("failed to decode ip-api response")
	}

	return [][]interface{}{
		{
			response.Country,
			response.CountryCode,
			response.Region,
			response.RegionName,
			response.City,
			response.Zip,
			response.Lat,
			response.Lon,
			response.Timezone,
			response.ISP,
			response.Org,
			response.As,
			response.Query,
		},
	}, true, nil

}

// Create a new cursor that will be used to read rows
func (t *ip_apiTable) CreateReader() rpc.ReaderInterface {
	return &ip_apiCursor{}
}

// A slice of rows to insert
func (t *ip_apiTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *ip_apiTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *ip_apiTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *ip_apiTable) Close() error {
	return nil
}
