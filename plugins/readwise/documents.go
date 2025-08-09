package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
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
func documentsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get a token from the user configuration
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	md5Sum := md5.Sum([]byte(token))
	sha256Sum := sha256.Sum256([]byte(token))

	// Open a cache connection
	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"readwise", "documents", fmt.Sprintf("%x", md5Sum)},
		EncryptionKey: []byte(sha256Sum[:]),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cache: %w", err)
	}

	// Create rate limiter (20 requests per minute as per API documentation)
	ratelimiter := ratelimit.New(20, ratelimit.Per(time.Minute))

	return &documentsTable{
			token:       token,
			cache:       cache,
			ratelimiter: ratelimiter,
		}, &rpc.DatabaseSchema{
			PrimaryKey:    0,
			Description:   "Documents saved in Readwise Reader",
			PartialUpdate: true,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "A unique identifier for the document saved in Readwise Reader",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "URL to view the document in Readwise Reader",
				},
				{
					Name:        "source_url",
					Type:        rpc.ColumnTypeString,
					Description: "Original URL of the document before it was saved to Readwise Reader",
				},
				{
					Name:        "title",
					Type:        rpc.ColumnTypeString,
					Description: "The title of the document",
				},
				{
					Name:        "author",
					Type:        rpc.ColumnTypeString,
					Description: "The author of the document",
				},
				{
					Name:        "source",
					Type:        rpc.ColumnTypeString,
					Description: "How the document was added to Readwise Reader (e.g., 'Reader RSS', 'Reader add from import URL')",
				},
				{
					Name:        "category",
					Type:        rpc.ColumnTypeString,
					Description: "The type of content (e.g., 'article', 'rss', 'book')",
				},
				{
					Name:        "location",
					Type:        rpc.ColumnTypeString,
					Description: "The current location/status of the document in Readwise Reader (e.g., 'feed', 'new', 'archive')",
				},
				{
					Name:        "tags",
					Type:        rpc.ColumnTypeJSON,
					Description: "Tags associated with the document",
				},
				{
					Name:        "site_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the site the document was sourced from",
				},
				{
					Name:        "word_count",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of words in the document",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the document was created in Readwise Reader",
				},
				{
					Name:        "updated_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the document was last updated in Readwise Reader",
				},
				{
					Name:        "published_date",
					Type:        rpc.ColumnTypeDate,
					Description: "The original publication date of the document",
				},
				{
					Name:        "notes",
					Type:        rpc.ColumnTypeString,
					Description: "User notes added to the document",
				},
				{
					Name:        "summary",
					Type:        rpc.ColumnTypeString,
					Description: "A brief summary of the document content",
				},
				{
					Name:        "image_url",
					Type:        rpc.ColumnTypeString,
					Description: "URL to an image associated with the document (e.g., thumbnail or featured image)",
				},
				{
					Name:        "parent_id",
					Type:        rpc.ColumnTypeString,
					Description: "ID of the parent document if this is part of a series or collection",
				},
				{
					Name:        "reading_progress",
					Type:        rpc.ColumnTypeFloat,
					Description: "Reading progress as a decimal between 0 and 1 (0.5 = 50% read)",
				},
				{
					Name:        "first_opened_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the document was first opened by the user",
				},
				{
					Name:        "last_opened_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the document was last opened by the user",
				},
				{
					Name:        "saved_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the document was saved to Readwise Reader",
				},
				{
					Name:        "last_moved_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the document was last moved between locations (e.g., from 'new' to 'archive')",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type documentsTable struct {
	token       string
	cache       *helper.Cache
	ratelimiter ratelimit.Limiter
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from documentsTable, an offset, a cursor, etc.)
type documentsCursor struct {
	token          string
	cache          *helper.Cache
	nextPageCursor string
	ratelimiter    ratelimit.Limiter
}

// Create a new cursor that will be used to read rows
func (t *documentsTable) CreateReader() rpc.ReaderInterface {
	return &documentsCursor{
		token:       t.token,
		cache:       t.cache,
		ratelimiter: t.ratelimiter,
	}
}

func (t *documentsTable) clearCache() {
	err := t.cache.ClearWithPrefix("documents")
	if err != nil {
		log.Printf("failed to clear cache: %s", err)
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *documentsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Define cache key
	cacheKey := fmt.Sprintf("documents_%s", t.nextPageCursor)

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
	var responseData DocumentList

	// Create request
	req := client.R().
		SetHeader("Authorization", "Token "+t.token).
		SetResult(&responseData)

	// Add pagination if we have a cursor
	if t.nextPageCursor != "" {
		req.SetQueryParam("pageCursor", t.nextPageCursor)
	}

	// Apply rate limiting
	t.ratelimiter.Take()

	// Make the request
	resp, err := req.Get("https://readwise.io/api/v3/list/")
	if err != nil {
		return nil, true, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to make request (%d): %s", resp.StatusCode(), resp.String())
	}

	// Process the documents into rows
	rows := make([][]interface{}, 0, len(responseData.Results))
	for _, doc := range responseData.Results {
		// Handle null values for datetime fields
		firstOpenedAt := doc.FirstOpenedAt
		lastOpenedAt := doc.LastOpenedAt
		parentID := doc.ParentID

		rows = append(rows, []interface{}{
			doc.ID,
			doc.URL,
			doc.SourceURL,
			doc.Title,
			doc.Author,
			doc.Source,
			doc.Category,
			doc.Location,
			helper.Serialize(doc.Tags),
			doc.SiteName,
			doc.WordCount,
			doc.CreatedAt,
			doc.UpdatedAt,
			doc.PublishedDate,
			doc.Notes,
			doc.Summary,
			doc.ImageURL,
			parentID,
			doc.ReadingProgress,
			firstOpenedAt,
			lastOpenedAt,
			doc.SavedAt,
			doc.LastMovedAt,
		})
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

// Document API endpoints
var documentCreateEndpoint = "https://readwise.io/api/v3/save/"                 // POST
var documentUpdateEndpoint = "https://readwise.io/api/v3/update/{document_id}/" // PATCH
var documentDeleteEndpoint = "https://readwise.io/api/v3/delete/{document_id}/" // DELETE

var enumLocations = []string{"feed", "new", "latest", "archive"}

// One of: article, email, rss, highlight, note, pdf, epub, tweet or video
var enumCategories = []string{"article", "email", "rss", "highlight", "note", "pdf", "epub", "tweet", "video"}

// A slice of rows to insert
func (t *documentsTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		// Essential fields
		sourceURL, ok := row[2].(string)
		if !ok || sourceURL == "" {
			continue // Skip if source_url is missing or empty
		}

		// Prepare document data
		document := map[string]interface{}{
			"url":         sourceURL,
			"saved_using": "anyquery",
		}

		// Optional fields
		if title, ok := row[3].(string); ok && title != "" {
			document["title"] = title
		}

		if author, ok := row[4].(string); ok && author != "" {
			document["author"] = author
		}

		if category, ok := row[6].(string); ok && category != "" {
			if stringInSlice(category, enumCategories) {
				document["category"] = category
			}
		}

		if location, ok := row[7].(string); ok && location != "" {
			if stringInSlice(location, enumLocations) {
				document["location"] = location
			}
		}

		// Tags processing
		if tags, ok := row[8].(string); ok && tags != "" {
			var tagsList []string
			if err := json.Unmarshal([]byte(tags), &tagsList); err == nil && len(tagsList) > 0 {
				document["tags"] = tagsList
			}
		}

		if notes, ok := row[14].(string); ok && notes != "" {
			document["notes"] = notes
		}

		if summary, ok := row[15].(string); ok && summary != "" {
			document["summary"] = summary
		}

		if imageURL, ok := row[16].(string); ok && imageURL != "" {
			document["image_url"] = imageURL
		}

		if publishedDate, ok := row[13].(string); ok && publishedDate != "" {
			document["published_date"] = publishedDate
		}

		// Apply rate limiting
		t.ratelimiter.Take()

		// Make request
		resp, err := client.R().
			SetHeader("Authorization", "Token "+t.token).
			SetHeader("Content-Type", "application/json").
			SetBody(document).
			Post(documentCreateEndpoint)

		if err != nil {
			return fmt.Errorf("failed to create document: %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("failed to create document (%d): %s", resp.StatusCode(), resp.String())
		}
	}

	// Clear the cache after successful inserts
	t.clearCache()

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *documentsTable) Update(rows [][]interface{}) error {
	for _, row := range rows {
		docID, ok := row[0].(string)
		if !ok || docID == "" {
			continue // Skip if document ID is missing or empty
		}

		// Prepare document update data
		document := map[string]interface{}{}
		hasChanges := false

		// Check for fields to update
		if title, ok := row[4].(string); ok {
			document["title"] = title
			hasChanges = true
		}

		if author, ok := row[5].(string); ok {
			document["author"] = author
			hasChanges = true
		}

		if category, ok := row[7].(string); ok {
			document["category"] = category
			hasChanges = true
		}

		if location, ok := row[8].(string); ok {
			document["location"] = location
			hasChanges = true
		}

		if summary, ok := row[16].(string); ok {
			document["summary"] = summary
			hasChanges = true
		}

		if imageURL, ok := row[17].(string); ok {
			document["image_url"] = imageURL
			hasChanges = true
		}

		if publishedDate, ok := row[14].(string); ok {
			document["published_date"] = publishedDate
			hasChanges = true
		}

		// Skip if there's nothing to update
		if !hasChanges {
			log.Printf("cannot update document %s because it has nothing to update", docID)
			continue
		}

		// Apply rate limiting
		t.ratelimiter.Take()

		// Make request
		resp, err := client.R().
			SetHeader("Authorization", "Token "+t.token).
			SetHeader("Content-Type", "application/json").
			SetPathParam("document_id", docID).
			SetBody(document).
			Patch(documentUpdateEndpoint)

		if err != nil {
			return fmt.Errorf("failed to update document: %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("failed to update document (%d): %s", resp.StatusCode(), resp.String())
		}
	}

	// Clear the cache after successful updates
	t.clearCache()

	return nil
}

// A slice of primary keys to delete
func (t *documentsTable) Delete(primaryKeys []interface{}) error {
	for _, primaryKey := range primaryKeys {
		if primaryKey == nil {
			continue
		}

		docID, ok := primaryKey.(string)
		if !ok || docID == "" {
			log.Printf("cannot delete document because the primary key is not a valid string")
			continue
		}

		// Apply rate limiting
		t.ratelimiter.Take()

		// Make delete request
		resp, err := client.R().
			SetHeader("Authorization", "Token "+t.token).
			SetPathParam("document_id", docID).
			Delete(documentDeleteEndpoint)

		if err != nil {
			return fmt.Errorf("failed to delete document: %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("failed to delete document (%d): %s", resp.StatusCode(), resp.String())
		}
	}

	// Clear the cache after successful deletes
	t.clearCache()

	return nil
}

// A destructor to clean up resources
func (t *documentsTable) Close() error {
	return t.cache.Close()
}
