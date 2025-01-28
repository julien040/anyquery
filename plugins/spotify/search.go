package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func searchCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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

	db, err := openDB("search", refreshToken.(string))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &searchTable{
			accessToken: accessToken,
			db:          db,
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The id of the result item",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the result item",
				},
				{
					Name:        "type",
					Type:        rpc.ColumnTypeString,
					Description: "The type of the result item. One of track, artist, album, playlist, show, episode, audiobook",
				},
				{
					Name: "href",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "query",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The query to search for",
				},
				{
					Name:        "object_type",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					Description: "The type of the object to search for. One of track, artist, album, playlist, show, episode, audiobook",
				},
				{
					Name:        "author",
					Type:        rpc.ColumnTypeString,
					Description: "The author of the result item",
				},
			},
		}, nil
}

type searchTable struct {
	accessToken string
	db          *badger.DB
}

type searchCursor struct {
	accessToken string
	db          *badger.DB
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *searchCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	endpoint := "https://api.spotify.com/v1/search"
	query := ""
	ok := false
	objectType := "track,playlist,album,artist"
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 4 {
			if query, ok = constraint.Value.(string); !ok {
				return nil, true, fmt.Errorf("query is not a string")
			}
		}
		if constraint.ColumnID == 5 {
			if objectType, ok = constraint.Value.(string); !ok {
				return nil, true, fmt.Errorf("object type is not a string")
			}
		}
	}
	if query == "" {
		return nil, true, fmt.Errorf("query is missing")
	}

	data := struct {
		Tracks struct {
			Items []trackAPI `json:"items"`
		} `json:"tracks"`
		Playlists struct {
			Items []simplifiedPlaylistAPI `json:"items"`
		} `json:"playlists"`
		Albums struct {
			Items []simplifiedAlbumAPI `json:"items"`
		} `json:"albums"`
		Artists struct {
			Items []simplifiedArtistAPI `json:"items"`
		} `json:"artists"`
	}{}

	// Check if the key exists in the database
	key := fmt.Sprintf("%s\x1c%s", query, objectType)
	err := t.db.View(func(txn *badger.Txn) error {
		res, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		// If the key exists, json decode the value
		return res.Value(func(val []byte) error {
			return json.Unmarshal(val, &data)
		})
	})

	// If the key doesn't exist, fetch the data from the API
	if err != nil {
		res, err := restyClient.R().SetHeader("Authorization", "Bearer "+t.accessToken).
			SetResult(&data).SetQueryParam("q", query).SetQueryParam("type", objectType).
			SetQueryParam("limit", "50").
			Get(endpoint)

		if err != nil {
			return nil, true, err
		}

		if res.StatusCode() != 200 {
			return nil, true, fmt.Errorf("failed to search: %s", res.String())
		}
	}

	rows := [][]interface{}{}
	for _, track := range data.Tracks.Items {
		artists := []string{}
		for _, artist := range track.Artists {
			artists = append(artists, artist.Name)
		}
		rows = append(rows, []interface{}{track.ID, track.Name, track.Type, track.Href, artists})
	}
	for _, playlist := range data.Playlists.Items {
		author := []string{playlist.Owner.DisplayName}
		rows = append(rows, []interface{}{playlist.ID, playlist.Name, playlist.Type, playlist.Href, author})
	}

	for _, album := range data.Albums.Items {
		artists := []string{}
		for _, artist := range album.Artists {
			artists = append(artists, artist.Name)
		}
		rows = append(rows, []interface{}{album.ID, album.Name, album.Type, album.Href, artists})
	}

	for _, artist := range data.Artists.Items {
		rows = append(rows, []interface{}{artist.ID, artist.Name, artist.Type, artist.Href, nil})
	}

	// Store the data in the database and ignore the error (it's not important, we can fetch it again later)
	t.db.Update(func(txn *badger.Txn) error {
		val, err := json.Marshal(data)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(key), val)
		return txn.SetEntry(e.WithTTL(1 * time.Hour))
	})

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *searchTable) CreateReader() rpc.ReaderInterface {
	return &searchCursor{
		accessToken: t.accessToken,
		db:          t.db,
	}
}

// A slice of rows to insert
func (t *searchTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *searchTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *searchTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *searchTable) Close() error {
	return nil
}
