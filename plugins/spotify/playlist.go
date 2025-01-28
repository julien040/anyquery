package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func playlistCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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
	return &playlistTable{
			accessToken: accessToken,
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					Description: "The ID of the playlist to search for. In https://open.spotify.com/playlist/37rI2gAtakAmSFtbIE9THq, the id is 37rI2gAtakAmSFtbIE9THq",
				},
				{
					Name:        "playlist_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the playlist",
				},
				{
					Name:        "playlist_followers",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of followers the playlist has",
				},
				{
					Name:        "playlist_owner",
					Type:        rpc.ColumnTypeString,
					Description: "The owner of the playlist",
				},
				{
					Name:        "playlist_href",
					Type:        rpc.ColumnTypeString,
					Description: "A link to the Web API endpoint providing full details of the playlist",
				},
				{
					Name:        "is_public",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the playlist is public or not",
				},
				{
					Name:        "album_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the album of the track in the playlist",
				},
				{
					Name:        "album_release_date",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The release date of the album of the track in the playlist",
				},
				{
					Name:        "artist_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the artist of the track in the playlist",
				},
				{
					Name:        "track_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the track in the playlist",
				},
				{
					Name:        "track_href",
					Type:        rpc.ColumnTypeString,
					Description: "A link to the Web API endpoint providing full details of the track",
				},
				{
					Name:        "track_popularity",
					Type:        rpc.ColumnTypeInt,
					Description: "The popularity of the track in the playlist (0-100)",
				},
				{
					Name:        "track_duration_ms",
					Type:        rpc.ColumnTypeInt,
					Description: "The duration of the track in milliseconds",
				},
				{
					Name:        "track_explicit",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether or not the track has explicit lyrics",
				},
				{
					Name:        "track_preview_url",
					Type:        rpc.ColumnTypeString,
					Description: "A link to a 30 second preview (MP3 format) of the track",
				},
				{
					Name:        "track_number",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of the track in the playlist",
				},
			},
		}, nil
}

type playlistTable struct {
	accessToken string
}

type playlistCursor struct {
	accessToken  string
	playlistID   string
	playlistName string
	followers    int
	owner        string
	href         string
	public       bool
	inited       bool
	nextQueryURL string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *playlistCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Check that the playlist ID is present
	playlistID := ""
	var ok bool
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 0 {
			if playlistID, ok = constraint.Value.(string); !ok {
				return nil, true, fmt.Errorf("playlist id is missing")
			}
			break
		}
	}

	if playlistID == "" {
		return nil, true, fmt.Errorf("playlist id is missing")
	}

	tracks := []trackAPI{}
	// Check if the cursor has been initialized
	if !t.inited {
		endpoint := "https://api.spotify.com/v1/playlists/" + playlistID + "?limit=50"
		data := struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Followers struct {
				Total int `json:"total"`
			} `json:"followers"`
			Owner struct {
				DisplayName interface{} `json:"display_name"` // can be null
			} `json:"owner"`
			Href   string `json:"href"`
			Public bool   `json:"public"`
			Tracks struct {
				Next  interface{} `json:"next"` // can be null
				Items []struct {
					Track trackAPI `json:"track"`
				} `json:"items"`
			} `json:"tracks"`
		}{}

		res, err := restyClient.R().SetHeader("Authorization", "Bearer "+t.accessToken).SetResult(&data).Get(endpoint)
		if err != nil {
			return nil, true, err
		}

		if res.StatusCode() != 200 {
			return nil, true, fmt.Errorf("failed to get playlist: %s", res.String())
		}

		t.playlistID = data.ID
		t.playlistName = data.Name
		t.followers = data.Followers.Total
		if data.Owner.DisplayName != nil {
			t.owner = data.Owner.DisplayName.(string)
		}
		t.href = data.Href
		t.public = data.Public
		if data.Tracks.Next != nil {
			t.nextQueryURL = data.Tracks.Next.(string)
		} else {
			t.nextQueryURL = ""
		}
		t.inited = true

		for _, item := range data.Tracks.Items {
			tracks = append(tracks, item.Track)
		}
	} else {
		// If the cursor has been initialized, we can fetch the next page
		if t.nextQueryURL != "" {
			data := struct {
				Items []struct {
					Track trackAPI `json:"track"`
				} `json:"items"`
				Next interface{} `json:"next"`
			}{}

			res, err := restyClient.R().SetHeader("Authorization", "Bearer "+t.accessToken).SetResult(&data).Get(t.nextQueryURL)
			if err != nil {
				return nil, true, err
			}

			if res.StatusCode() != 200 {
				return nil, true, fmt.Errorf("failed to get playlist: %s", res.String())
			}

			if data.Next != nil {
				t.nextQueryURL = data.Next.(string)
			} else {
				t.nextQueryURL = ""
			}

			for _, item := range data.Items {
				tracks = append(tracks, item.Track)
			}
		}
	}

	rows := [][]interface{}{}
	for _, track := range tracks {
		artists := []string{}
		for _, artist := range track.Artists {
			artists = append(artists, artist.Name)
		}

		rows = append(rows, []interface{}{
			t.playlistName,
			t.followers,
			t.owner,
			t.href,
			t.public,
			track.Album.Name,
			track.Album.ReleaseDate,
			artists,
			track.Name,
			track.Href,
			track.Popularity,
			track.DurationMs,
			track.Explicit,
			track.PreviewURL,
			track.TrackNumber,
		})
	}

	return rows, t.nextQueryURL == "", nil
}

// Create a new cursor that will be used to read rows
func (t *playlistTable) CreateReader() rpc.ReaderInterface {
	return &playlistCursor{
		accessToken: t.accessToken,
	}
}

// A destructor to clean up resources
func (t *playlistTable) Close() error {
	return nil
}
