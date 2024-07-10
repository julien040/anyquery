package main

type imageObjectAPI struct {
	Height int    `json:"height"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
}

type simplifiedArtistAPI struct {
	Href string `json:"href"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	URI  string `json:"uri"`
}

type artistAPI struct {
	Followers struct {
		Href  interface{} `json:"href"`
		Total int         `json:"total"`
	} `json:"followers"`
	Genres     []string         `json:"genres"`
	Href       string           `json:"href"`
	ID         string           `json:"id"`
	Images     []imageObjectAPI `json:"images"`
	Name       string           `json:"name"`
	Popularity int              `json:"popularity"`
	Type       string           `json:"type"`
	URI        string           `json:"uri"`
}

type simplifiedPlaylistAPI struct {
	Collaborative bool   `json:"collaborative"`
	Description   string `json:"description"`
	Href          string `json:"href"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Owner         struct {
		DisplayName string `json:"display_name"`
		Href        string `json:"href"`
		ID          string `json:"id"`
		Type        string `json:"type"`
		URI         string `json:"uri"`
	} `json:"owner"`
	Public bool `json:"public"`
	Tracks struct {
		Href  string `json:"href"`
		Total int    `json:"total"`
	} `json:"tracks"`
	Type string `json:"type"`
	URI  string `json:"uri"`
}

type albumAPI struct {
	AlbumType            string                `json:"album_type"`
	TotalTracks          int                   `json:"total_tracks"`
	Href                 string                `json:"href"`
	ID                   string                `json:"id"`
	Images               []imageObjectAPI      `json:"images"`
	Name                 string                `json:"name"`
	ReleaseDate          string                `json:"release_date"`
	ReleaseDatePrecision string                `json:"release_date_precision"`
	Type                 string                `json:"type"`
	URI                  string                `json:"uri"`
	Artists              []simplifiedArtistAPI `json:"artists"`
	Tracks               struct {
		Href     string `json:"href"`
		Items    []simplifiedTrackAPI
		Limit    int `json:"limit"`
		Next     string
		Offset   int `json:"offset"`
		Previous string
		Total    int `json:"total"`
	} `json:"tracks"`
	Copyright []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"copyright"`
	Popularity int      `json:"popularity"`
	Genres     []string `json:"genres"`
	Label      string   `json:"label"`
}

type simplifiedAlbumAPI struct {
	AlbumType            string                `json:"album_type"`
	Href                 string                `json:"href"`
	ID                   string                `json:"id"`
	Images               []imageObjectAPI      `json:"images"`
	Name                 string                `json:"name"`
	ReleaseDate          string                `json:"release_date"`
	ReleaseDatePrecision string                `json:"release_date_precision"`
	Type                 string                `json:"type"`
	URI                  string                `json:"uri"`
	Artists              []simplifiedArtistAPI `json:"artists"`
	TotalTracks          int                   `json:"total_tracks"`
}

type simplifiedTrackAPI struct {
	Artists     []simplifiedArtistAPI `json:"artists"`
	DurationMs  int                   `json:"duration_ms"`
	DiscNumber  int                   `json:"disc_number"`
	Explicit    bool                  `json:"explicit"`
	Href        string                `json:"href"`
	ID          string                `json:"id"`
	IsPlayable  bool                  `json:"is_playable"`
	Name        string                `json:"name"`
	PreviewURL  string                `json:"preview_url"`
	TrackNumber int                   `json:"track_number"`
	Type        string                `json:"type"`
	URI         string                `json:"uri"`
	IsLocal     bool                  `json:"is_local"`
}

type trackAPI struct {
	Album       albumAPI    `json:"album"`
	Artists     []artistAPI `json:"artists"`
	DiscNumber  int         `json:"disc_number"`
	DurationMs  int         `json:"duration_ms"`
	Explicit    bool        `json:"explicit"`
	Href        string      `json:"href"`
	ID          string      `json:"id"`
	IsPlayable  bool        `json:"is_playable"`
	Name        string      `json:"name"`
	Popularity  int         `json:"popularity"`
	PreviewURL  string      `json:"preview_url"`
	TrackNumber int         `json:"track_number"`
	Type        string      `json:"type"`
	URI         string      `json:"uri"`
	IsLocal     bool        `json:"is_local"`
}
