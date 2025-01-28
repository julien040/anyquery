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
func historyCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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

	db, err := openDB("history", refreshToken.(string))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &historyTable{
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
					Description: "The ID of the track",
				},
				{
					Name:        "played_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The time the track was played (RFC3339)",
				},
				{
					Name:        "played_from",
					Type:        rpc.ColumnTypeString,
					Description: "The source of the track (artist, album, playlist, show)",
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
					Name: "href",
					Type: rpc.ColumnTypeString,
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
					Description: "Whether the track is explicit",
				},
				{
					Name:        "preview_url",
					Type:        rpc.ColumnTypeString,
					Description: "The preview URL of the track. It's a small MP3 file that you can play in the browser. It's only available for a short time.",
				},
				{
					Name:        "track_number",
					Type:        rpc.ColumnTypeInt,
					Description: "The track number of the track",
				},
			},
		}, nil
}

type historyTable struct {
	accessToken string
	db          *badger.DB
}

type historyCursor struct {
	accessToken string
	db          *badger.DB
	nextCursor  string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *historyCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	data := struct {
		Next    interface{} `json:"next"` // To check if the cursor is at the end. If so, nil is returned
		Total   int         `json:"total"`
		Cursors struct {
			After  string `json:"after"`
			Before string `json:"before"`
		} `json:"cursors"`
		Items []struct {
			PlayedAt string `json:"played_at"`
			Context  struct {
				URI  string `json:"uri"`
				Type string `json:"type"`
			} `json:"context"`
			Track trackAPI `json:"track"`
		} `json:"items"`
	}{}

	cacheKey := fmt.Sprintf("history_cursor_%s", t.nextCursor)

	// Try to check if the cursor is in the database
	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}
		// Try to unmarshal the data
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &data)
		})
	})

	if err != nil {
		// If the cursor is not in the database, fetch it from the API
		urlReq := "https://api.spotify.com/v1/me/player/recently-played?limit=50"
		if t.nextCursor != "" {
			urlReq += "&before=" + t.nextCursor
		}

		res, err := restyClient.R().SetHeader("Authorization", "Bearer "+t.accessToken).SetResult(&data).Get(urlReq)
		if err != nil {
			return nil, true, fmt.Errorf("failed to get recently played: %w", err)
		}

		if res.StatusCode() != 200 {
			return nil, true, fmt.Errorf("failed to get recently played: %s", res.String())
		}

		// Save the cursor in the database and ignore the error
		err = t.db.Update(func(txn *badger.Txn) error {
			val, err := json.Marshal(data)
			if err != nil {
				return err
			}
			e := badger.NewEntry([]byte(cacheKey), val)
			return txn.SetEntry(e.WithTTL(1 * time.Hour))
		})
		if err != nil {
			log.Printf("Failed to save cursor: %v\n", err)
		}

	}
	var rows [][]interface{}
	for _, item := range data.Items {
		artists := []string{}
		for _, artist := range item.Track.Artists {
			artists = append(artists, artist.Name)
		}

		rows = append(rows, []interface{}{
			item.Track.ID,
			item.PlayedAt,
			item.Context.Type,
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

	t.nextCursor = data.Cursors.Before
	return rows, data.Next == nil, nil
}

// Create a new cursor that will be used to read rows
func (t *historyTable) CreateReader() rpc.ReaderInterface {
	return &historyCursor{
		accessToken: t.accessToken,
		db:          t.db,
	}
}

// A destructor to clean up resources
func (t *historyTable) Close() error {
	return nil
}
