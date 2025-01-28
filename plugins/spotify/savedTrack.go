package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func savedTrackCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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

	db, err := openDB("savedTrack", refreshToken.(string))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &savedTrackTable{
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
					Description: "The ID of the track. In https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT, the id is 4cOdK2wGLETKBW3PvgPWqT",
				},
				{
					Name:        "saved_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The time the track was saved",
				},
				{
					Name:        "artist_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the artist",
				},
				{
					Name:        "track_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the track",
				},
				{
					Name:        "album_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the album",
				},
				{
					Name:        "album_release_date",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The release date of the album",
				},
				{
					Name:        "href",
					Type:        rpc.ColumnTypeString,
					Description: "A link to the Web API endpoint providing full details of the track",
				},
				{
					Name:        "popularity",
					Type:        rpc.ColumnTypeInt,
					Description: "The popularity of the track",
				},
				{
					Name:        "duration_ms",
					Type:        rpc.ColumnTypeInt,
					Description: "The duration of the track in milliseconds",
				},
				{
					Name:        "explicit",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether or not the track has explicit lyrics",
				},
				{
					Name:        "preview_url",
					Type:        rpc.ColumnTypeString,
					Description: "A link to a 30 second preview (MP3 format) of the track",
				},
				{
					Name:        "track_number",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of the track",
				},
			},
		}, nil
}

type savedTrackTable struct {
	accessToken string
	db          *badger.DB
}

type savedTrackCursor struct {
	accessToken     string
	db              *badger.DB
	nextURL         string
	cursorExhausted bool
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *savedTrackCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	endpoint := "https://api.spotify.com/v1/me/tracks?limit=50&offset=0"
	if t.nextURL != "" {
		endpoint = t.nextURL
	}

	var data struct {
		Items []struct {
			AddedAt string `json:"added_at"`
			Track   trackAPI
		}
		Next interface{} `json:"next"`
	}

	// Try to fetch the page from the DB
	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(endpoint))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &data)
		})
	})

	if err != nil {
		// Try to fetch the page from the API
		res, err := restyClient.R().SetHeader("Authorization", "Bearer "+t.accessToken).
			SetResult(&data).Get(endpoint)
		if err != nil {
			return nil, true, err
		}

		if res.StatusCode() != 200 {
			return nil, true, fmt.Errorf("failed to fetch data: %s", res.String())
		}

		// Save the data in the DB
		err = t.db.Update(func(txn *badger.Txn) error {
			val, err := json.Marshal(data)
			if err != nil {
				return err
			}
			e := badger.NewEntry([]byte(endpoint), val)
			return txn.SetEntry(e.WithTTL(time.Hour))
		})

		if err != nil {
			log.Printf("Failed to save data: %s\n", err)
		}
	}

	rows := [][]interface{}{}
	for _, item := range data.Items {
		artists := []string{}
		for _, artist := range item.Track.Artists {
			artists = append(artists, artist.Name)
		}

		rows = append(rows, []interface{}{
			item.Track.ID,
			item.AddedAt,
			artists,
			item.Track.Name,
			item.Track.Album.Name,
			item.Track.Album.ReleaseDate,
			item.Track.Href,
			item.Track.Popularity,
			item.Track.DurationMs,
			item.Track.Explicit,
			item.Track.PreviewURL,
			item.Track.TrackNumber,
		})
	}

	if _, ok := data.Next.(string); ok {
		t.nextURL = data.Next.(string)
	} else {
		t.cursorExhausted = true
	}

	if len(rows) == 0 {
		t.cursorExhausted = true
	}

	return rows, t.cursorExhausted, nil
}

// Create a new cursor that will be used to read rows
func (t *savedTrackTable) CreateReader() rpc.ReaderInterface {
	return &savedTrackCursor{
		accessToken: t.accessToken,
		db:          t.db,
	}
}

// A slice of rows to insert
func (t *savedTrackTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *savedTrackTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *savedTrackTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *savedTrackTable) Close() error {
	return nil
}
