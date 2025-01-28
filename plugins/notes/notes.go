//go:build darwin

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

//go:embed scripts/list.js
var listScript string

//go:embed scripts/countNotes.js
var countNotesScript string

type item struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	CreationDate     string `json:"creationDate"`
	ModificationDate string `json:"modificationDate"`
	HTMLBody         string `json:"htmlBody"`
	Folder           string `json:"folder"`
	Account          string `json:"account"`
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func notesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Open the database for the cache
	cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "notes")
	err := os.MkdirAll(cacheFolder, 0755)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cache folder: %w", err)
	}

	// Open the badger database encrypted with the toke
	options := badger.DefaultOptions(cacheFolder).WithNumVersionsToKeep(1).
		WithCompactL0OnClose(true).WithValueLogFileSize(2 << 24).
		WithIndexCacheSize(2 << 29)
	db, err := badger.Open(options)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open badger database: %w", err)
	}

	// Get the count of notes
	count := 0
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", countNotesScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count notes: %w. Output: %s", err, string(output))
	}

	outputTrimmed := strings.Trim(string(output), "\n ,\t\"")

	// Parse the count
	count, err = strconv.Atoi(outputTrimmed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse count: %w. Output: %s", err, string(output))
	}

	// Check if the count is the same as the cache
	// If not, drop the cache (it's not perfect but it's good enough)
	dbCount := 0
	if count != 0 {
		db.View(func(txn *badger.Txn) error {
			iterator := txn.NewIterator(badger.IteratorOptions{
				Prefix:         []byte("note-"),
				PrefetchValues: false,
			})

			for iterator.Rewind(); iterator.Valid(); iterator.Next() {
				dbCount++
			}
			iterator.Close()
			return nil
		})
	}

	if dbCount != count {
		db.DropPrefix([]byte("note-"))
	}

	return &notesTable{
			cache:     db,
			noteCount: count,
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the note",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the note",
				},
				{
					Name:        "creation_date",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The creation date of the note (RFC3339 format)",
				},
				{
					Name:        "modification_date",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The modification date of the note (RFC3339 format)",
				},
				{
					Name:        "html_body",
					Type:        rpc.ColumnTypeString,
					Description: "The HTML body of the note. Images are base64 encoded. Inline css is used for text formatting",
				},
				{
					Name:        "folder",
					Type:        rpc.ColumnTypeString,
					Description: "The folder of the note",
				},
				{
					Name:        "account",
					Type:        rpc.ColumnTypeString,
					Description: "The account of the note",
				},
			},
		}, nil
}

type notesTable struct {
	cache     *badger.DB
	noteCount int
}

type notesCursor struct {
	cache         *badger.DB
	offset        int
	exhausted     bool
	decoder       *json.Decoder
	decodedOffset int
	count         int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *notesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Check if the cursor is exhausted
	if t.exhausted {
		return nil, true, nil
	}

	// Get the next row from the cache
	cacheKey := fmt.Sprintf("note-%d", t.offset)
	val := item{}
	err := t.cache.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}

		return item.Value(func(valBytes []byte) error {
			return json.Unmarshal(valBytes, &val)
		})
	})
	if err != nil {
		log.Printf("failed to get note from cache: %v", err)
		// Fetch the next row from the decoder
		err = t.makeDecoder()
		if err != nil {
			return nil, true, err
		}

		// Reconcile the offset
		for t.decodedOffset <= t.offset {
			err = t.decoder.Decode(&val)
			if err != nil {
				if err.Error() == "EOF" {
					t.exhausted = true
					return nil, true, nil
				}
			}
			t.decodedOffset++
		}
	}

	// Convert the row to a slice of interfaces
	row := []interface{}{
		val.ID,
		val.Name,
		val.CreationDate,
		val.ModificationDate,
		val.HTMLBody,
		val.Folder,
		val.Account,
	}

	// Save the row to the cache
	err = t.cache.Update(func(txn *badger.Txn) error {
		valBytes, err := json.Marshal(val)
		if err != nil {
			return err
		}

		e := badger.NewEntry([]byte(cacheKey), valBytes).WithTTL(1 * time.Hour)
		return txn.SetEntry(e)
	})
	if err != nil {
		log.Printf("failed to save note to cache: %v", err)
	}

	// Increment the offset
	t.offset++

	return [][]interface{}{row}, t.exhausted || t.offset >= t.count, nil
}

// Create a new decoder from the apple script if it doesn't exist
func (t *notesCursor) makeDecoder() error {
	if t.decoder != nil {
		return nil
	}

	// Open the apple script
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", listScript)
	cmdReader, err := cmd.StderrPipe() // osascript writes to stderr
	if err != nil {
		return fmt.Errorf("failed to open stderr pipe: %w", err)
	}

	// Create a new decoder
	t.decoder = json.NewDecoder(cmdReader)

	// Start the command
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	return nil
}

// Create a new cursor that will be used to read rows
func (t *notesTable) CreateReader() rpc.ReaderInterface {
	return &notesCursor{
		cache: t.cache,
		count: t.noteCount,
	}
}

// A destructor to clean up resources
func (t *notesTable) Close() error {
	return nil
}
