package main

type Highlights struct {
	Count          int64    `json:"count"`
	NextPageCursor string   `json:"nextPageCursor"`
	Results        []Result `json:"results"`
}

type Result struct {
	UserBookID    int64       `json:"user_book_id"`
	IsDeleted     bool        `json:"is_deleted"`
	Title         string      `json:"title"`
	Author        string      `json:"author"`
	ReadableTitle string      `json:"readable_title"`
	Source        string      `json:"source"`
	CoverImageURL string      `json:"cover_image_url"`
	UniqueURL     string      `json:"unique_url"`
	Summary       string      `json:"summary"`
	BookTags      []Tag       `json:"book_tags"`
	Category      string      `json:"category"`
	DocumentNote  string      `json:"document_note"`
	ReadwiseURL   string      `json:"readwise_url"`
	SourceURL     string      `json:"source_url"`
	ExternalID    string      `json:"external_id"`
	Asin          string      `json:"asin"`
	Highlights    []Highlight `json:"highlights"`
}

type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Highlight struct {
	ID            int64       `json:"id"`
	IsDeleted     bool        `json:"is_deleted"`
	Text          string      `json:"text"`
	Location      int64       `json:"location"`
	LocationType  string      `json:"location_type"`
	Note          string      `json:"note"`
	Color         string      `json:"color"`
	HighlightedAt string      `json:"highlighted_at"`
	CreatedAt     string      `json:"created_at"`
	UpdatedAt     string      `json:"updated_at"`
	ExternalID    string      `json:"external_id"`
	EndLocation   interface{} `json:"end_location"`
	URL           string      `json:"url"`
	BookID        int64       `json:"book_id"`
	Tags          []Tag       `json:"tags"`
	IsFavorite    bool        `json:"is_favorite"`
	IsDiscard     bool        `json:"is_discard"`
	ReadwiseURL   string      `json:"readwise_url"`
}

type HighlightCreate struct {
	Text          string  `json:"text"` // required
	Title         *string `json:"title,omitempty"`
	Author        *string `json:"author,omitempty"`
	ImageURL      *string `json:"image_url,omitempty"`
	SourceURL     *string `json:"source_url,omitempty"`
	SourceType    string  `json:"source_type,omitempty"`
	Category      *string `json:"category,omitempty"`
	Note          *string `json:"note,omitempty"`
	Location      *int    `json:"location,omitempty"`
	LocationType  *string `json:"location_type,omitempty"`
	HighlightedAt *string `json:"highlighted_at,omitempty"` // ISO-8601 string
	HighlightURL  *string `json:"highlight_url,omitempty"`
}

type HighlightUpdate struct {
	Text     *string `json:"text,omitempty"`
	Note     *string `json:"note,omitempty"`
	Location *int    `json:"location,omitempty"`
	URL      *string `json:"url,omitempty"`
	Color    *string `json:"color,omitempty"` // yellow, blue, pink, orange, green, purple
}

type DocumentList struct {
	Count          int64      `json:"count"`
	NextPageCursor string     `json:"nextPageCursor"`
	Results        []Document `json:"results"`
}

type Document struct {
	ID              string      `json:"id"`
	URL             string      `json:"url"`
	Title           string      `json:"title"`
	Author          string      `json:"author"`
	Source          string      `json:"source"`
	Category        string      `json:"category"`
	Location        string      `json:"location"`
	Tags            interface{} `json:"tags"`
	SiteName        string      `json:"site_name"`
	WordCount       int64       `json:"word_count"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
	PublishedDate   interface{} `json:"published_date"`
	Summary         string      `json:"summary"`
	ImageURL        string      `json:"image_url"`
	Content         string      `json:"content"`
	SourceURL       string      `json:"source_url"`
	Notes           string      `json:"notes"`
	ParentID        string      `json:"parent_id"`
	ReadingProgress float64     `json:"reading_progress"`
	FirstOpenedAt   string      `json:"first_opened_at"`
	LastOpenedAt    string      `json:"last_opened_at"`
	SavedAt         string      `json:"saved_at"`
	LastMovedAt     string      `json:"last_moved_at"`
}

type Tags struct {
}
