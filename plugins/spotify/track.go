package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func trackCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	refreshToken, ok := args.UserConfig["token"]
	if !ok {
		return nil, nil, fmt.Errorf("token is missing")
	}
	clientID, ok := args.UserConfig["client_id"]
	if !ok {
		return nil, nil, fmt.Errorf("client_id is missing")
	}
	clientSecret, ok := args.UserConfig["client_secret"]
	if !ok {
		return nil, nil, fmt.Errorf("client_secret is missing")
	}

	accessToken, err := getAccessToken(refreshToken.(string), clientID.(string), clientSecret.(string))
	if err != nil {
		return nil, nil, err
	}
	return &trackTable{
			accessToken: accessToken,
		},
		&rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name: "album_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "album_release_date",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "artist_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "track_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "href",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "popularity",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "duration_ms",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "explicit",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "preview_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "track_number",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

type trackTable struct {
	accessToken string
}

type trackCursor struct {
	accessToken string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *trackCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	endpoint := "https://api.spotify.com/v1/tracks/"

	trackID := ""
	var ok bool
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 0 {
			if trackID, ok = constraint.Value.(string); !ok {
				return nil, true, fmt.Errorf("track id is missing")
			}
			break
		}
	}
	if trackID == "" {
		return nil, true, fmt.Errorf("track id is missing")
	}

	endpoint += trackID

	var data trackAPI
	res, err := restyClient.R().SetHeader("Authorization", "Bearer "+t.accessToken).SetResult(&data).Get(endpoint)
	if err != nil {
		return nil, true, err
	}

	if res.StatusCode() != 200 {
		return nil, true, fmt.Errorf("failed to get track: %s", res.String())
	}

	artists := []string{}
	for _, artist := range data.Artists {
		artists = append(artists, artist.Name)
	}

	return [][]interface{}{
		{
			data.Album.Name,
			data.Album.ReleaseDate,
			artists,
			data.Name,
			data.Href,
			data.Popularity,
			data.DurationMs,
			data.Explicit,
			data.PreviewURL,
			data.TrackNumber,
		},
	}, true, nil
}

// Create a new cursor that will be used to read rows
func (t *trackTable) CreateReader() rpc.ReaderInterface {
	return &trackCursor{
		accessToken: t.accessToken,
	}
}

// A slice of rows to insert
func (t *trackTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *trackTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *trackTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *trackTable) Close() error {
	return nil
}
