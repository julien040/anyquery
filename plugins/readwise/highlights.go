package main

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"go.uber.org/ratelimit"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func highlightsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get a token from the user configuration
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	md5Sum := md5.Sum([]byte(token))
	sha256Sum := sha256.Sum256([]byte(token))

	// Open a cache connection
	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"readwise", "highlights", fmt.Sprintf("%x", md5Sum)},
		EncryptionKey: []byte(sha256Sum[:]),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cache: %w", err)
	}

	ratelimiter := ratelimit.New(240, ratelimit.Per(time.Minute))

	return &highlightsTable{
			token:       token,
			cache:       cache,
			ratelimiter: ratelimiter,
		}, &rpc.DatabaseSchema{
			PrimaryKey:    0,
			PartialUpdate: true,
			Description:   "Highlights from Readwise",
			BufferInsert:  100,
			Columns: []rpc.DatabaseSchemaColumn{
				// Highlight fields
				{
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					Description: "The unique identifier of the highlight",
				},
				{
					Name:        "text",
					Type:        rpc.ColumnTypeString,
					Description: "The highlight text (maximum length: 8191 characters)",
				},
				{
					Name:        "note",
					Type:        rpc.ColumnTypeString,
					Description: "Annotation note attached to the specific highlight, can include inline tags (maximum length: 8191 characters)",
				},
				{
					Name:        "location",
					Type:        rpc.ColumnTypeInt,
					Description: "Highlight's location in the source text, used to order highlights",
				},
				{
					Name:        "location_type",
					Type:        rpc.ColumnTypeString,
					Description: "Type of location: page, location, none, order, offset or time_offset",
				},
				{
					Name:        "color",
					Type:        rpc.ColumnTypeString,
					Description: "The color of the highlight. One of yellow, blue, pink, orange, green, purple",
				},
				{
					Name:        "highlighted_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "A datetime representing when the highlight was created in ISO 8601 format",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "A datetime when the highlight was created in Readwise",
				},
				{
					Name:        "updated_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "A datetime when the highlight was last updated in Readwise",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "Unique URL of the specific highlight (e.g. a concrete tweet or a podcast snippet)",
				},
				{
					Name:        "is_favorite",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the highlight is marked as favorite",
				},
				{
					Name:        "is_discard",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the highlight is marked as discard",
				},
				{
					Name:        "tags",
					Type:        rpc.ColumnTypeJSON,
					Description: "Tags associated with this highlight",
				},
				// Book metadata
				{
					Name:        "book_id",
					Type:        rpc.ColumnTypeInt,
					Description: "The unique identifier of the book/article/source",
				},
				{
					Name:        "book_title",
					Type:        rpc.ColumnTypeString,
					Description: "Title of the book/article/podcast (maximum length: 511 characters)",
				},
				{
					Name:        "book_author",
					Type:        rpc.ColumnTypeString,
					Description: "Author of the book/article/podcast (maximum length: 1024 characters)",
				},
				{
					Name:        "book_source",
					Type:        rpc.ColumnTypeString,
					Description: "Source of the content (e.g. kindle, article, etc.)",
				},
				{
					Name:        "book_category",
					Type:        rpc.ColumnTypeString,
					Description: "Category of the source: books, articles, tweets or podcasts",
				},
				{
					Name:        "book_cover_image_url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL of a cover image for the source (maximum length: 2047 characters)",
				},
				{
					Name:        "book_summary",
					Type:        rpc.ColumnTypeString,
					Description: "A summary of the book/article content",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type highlightsTable struct {
	token       string
	cache       *helper.Cache
	ratelimiter ratelimit.Limiter
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from highlightsTable, an offset, a cursor, etc.)
type highlightsCursor struct {
	token          string
	cache          *helper.Cache
	nextPageCursor string
	ratelimiter    ratelimit.Limiter
}

// Create a new cursor that will be used to read rows
func (t *highlightsTable) CreateReader() rpc.ReaderInterface {
	return &highlightsCursor{
		token:       t.token,
		cache:       t.cache,
		ratelimiter: t.ratelimiter,
	}
}

func (t *highlightsTable) clearCache() {
	err := t.cache.ClearWithPrefix("highlights")
	if err != nil {
		log.Printf("failed to clear cache: %s", err)
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *highlightsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Define cache key
	cacheKey := fmt.Sprintf("highlights_%s", t.nextPageCursor)

	// Try to get data from cache first
	cachedData, metadata, err := t.cache.Get(cacheKey)
	if err == nil {
		// Get the nextPageCursor from the metadata
		nextPageCursor, ok := metadata["nextPageCursor"].(string)
		if ok {
			t.nextPageCursor = nextPageCursor
		}

		return cachedData, t.nextPageCursor == "", nil
	}

	// Data not in cache, we fetch from API
	var responseData Highlights

	req := client.R().
		SetHeader("Authorization", "Token "+t.token).SetResult(&responseData)

	if t.nextPageCursor != "" {
		req.SetQueryParam("pageCursor", t.nextPageCursor)
	}

	t.ratelimiter.Take()
	resp, err := req.Get("https://readwise.io/api/v2/export/")
	if err != nil {
		return nil, true, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to make request (%d): %s", resp.StatusCode(), resp.String())
	}

	rows := make([][]interface{}, 0, len(responseData.Results))
	for _, result := range responseData.Results {
		for _, highlight := range result.Highlights {
			rows = append(rows, []interface{}{
				highlight.ID,
				highlight.Text,
				highlight.Note,
				highlight.Location,
				highlight.LocationType,
				highlight.Color,
				highlight.HighlightedAt,
				highlight.CreatedAt,
				highlight.UpdatedAt,
				highlight.URL,
				highlight.IsFavorite,
				highlight.IsDiscard,
				helper.Serialize(highlight.Tags),
				result.UserBookID,
				result.Title,
				result.Author,
				result.Source,
				result.Category,
				result.CoverImageURL,
				result.Summary,
			})
		}
	}

	// Store the nextPageCursor in the cache
	t.nextPageCursor = responseData.NextPageCursor
	err = t.cache.Set(cacheKey,
		rows,
		map[string]interface{}{"nextPageCursor": t.nextPageCursor},
		time.Hour*24)
	if err != nil {
		log.Printf("Failed to cache rows: %v\n", err)
	}

	return rows, t.nextPageCursor == "", nil

}

// A destructor to clean up resourcesreturn nil
func (t *highlightsTable) Close() error {
	return t.cache.Close()
}

var insertEndpoint = "https://readwise.io/api/v2/highlights/" // POST

func (t *highlightsTable) Insert(rows [][]interface{}) error {
	highlights := make([]HighlightCreate, 0, len(rows))
	for _, row := range rows {
		if row[1] == nil { // Text column is missing
			continue
		}

		highlight := HighlightCreate{}
		highlight.SourceType = "anyquery"
		if text, ok := row[1].(string); ok {
			highlight.Text = text
		}

		// Book related
		if title, ok := row[14].(string); ok {
			highlight.Title = &title
		}

		if author, ok := row[15].(string); ok {
			highlight.Author = &author
		}

		if sourceURL, ok := row[16].(string); ok {
			highlight.SourceURL = &sourceURL
		}

		if category, ok := row[17].(string); ok {
			highlight.Category = &category
		}

		if coverImageURL, ok := row[18].(string); ok {
			highlight.ImageURL = &coverImageURL
		}

		// Highlight related

		if note, ok := row[2].(string); ok {
			highlight.Note = &note
		}

		if location, ok := row[3].(int); ok {
			highlight.Location = &location
		}
		if locationType, ok := row[4].(string); ok {
			highlight.LocationType = &locationType
		}

		if highlightedAt, ok := row[6].(string); ok {
			highlight.HighlightedAt = &highlightedAt
		}

		highlights = append(highlights, highlight)
	}

	body := map[string]interface{}{
		"highlights": highlights,
	}

	t.ratelimiter.Take()
	resp, err := client.R().
		SetHeader("Authorization", "Token "+t.token).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(insertEndpoint)

	if err != nil {
		return fmt.Errorf("failed to insert highlights: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to insert highlights(%d): %s", resp.StatusCode(), resp.String())
	}

	t.clearCache()

	return nil

}

var updateDeleteEndpoint = "https://readwise.io/api/v2/highlights/{highlight_id}/" // PATCH or DELETE

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *highlightsTable) Update(rows [][]interface{}) error {
	for _, row := range rows {
		id, ok := row[0].(string)
		if !ok {
			if idInt, ok := row[0].(int64); ok {
				id = fmt.Sprintf("%d", idInt)
			} else {
				continue
			}
		}

		canBeUpdated := false

		highlight := HighlightUpdate{}
		if text, ok := row[2].(string); ok {
			highlight.Text = &text
			canBeUpdated = true
		}

		if note, ok := row[3].(string); ok {
			highlight.Note = &note
			canBeUpdated = true
		}

		if location, ok := row[4].(int); ok {
			highlight.Location = &location
			canBeUpdated = true
		}

		if url, ok := row[5].(string); ok {
			highlight.URL = &url
			canBeUpdated = true
		}

		if color, ok := row[6].(string); ok {
			if color == "yellow" || color == "blue" || color == "pink" || color == "orange" || color == "green" || color == "purple" {
				highlight.Color = &color
				canBeUpdated = true
			}
		}

		// Check if we can update the highlight
		if !canBeUpdated {
			log.Printf("cannot update highlight %s because it has nothing to update", id)
			continue
		}

		t.ratelimiter.Take()

		resp, err := client.R().
			SetHeader("Authorization", "Token "+t.token).
			SetHeader("Content-Type", "application/json").
			SetPathParam("highlight_id", id).
			SetBody(highlight).
			Patch(updateDeleteEndpoint)

		if err != nil {
			return fmt.Errorf("failed to update highlights: %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("failed to update highlights(%d): %s", resp.StatusCode(), resp.String())
		}
	}

	t.clearCache()

	return nil
}

func (t *highlightsTable) Delete(primaryKeys []interface{}) error {
	for _, primaryKey := range primaryKeys {
		if primaryKey == nil {
			continue
		}

		var pk string
		// Check if the primary key is a string
		if pkInt, ok := primaryKey.(int64); ok {
			pk = fmt.Sprintf("%d", pkInt)
		}

		if pkStr, ok := primaryKey.(string); ok {
			pk = pkStr
		}

		if pk == "" {
			log.Printf("cannot delete highlight because the primary key is not a string or an int")
			continue
		}

		t.ratelimiter.Take()
		resp, err := client.R().
			SetHeader("Authorization", "Token "+t.token).
			SetHeader("Content-Type", "application/json").
			SetPathParam("highlight_id", pk).
			Delete(updateDeleteEndpoint)

		if err != nil {
			return fmt.Errorf("failed to delete highlights: %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("failed to delete highlights(%d): %s", resp.StatusCode(), resp.String())
		}
	}

	t.clearCache()
	return nil
}
