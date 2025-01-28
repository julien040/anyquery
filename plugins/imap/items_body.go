package main

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func itemsBodyCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	config, err := getArgs(args.UserConfig)
	if err != nil {
		return nil, nil, err
	}

	dialer, err := client.DialTLS(fmt.Sprintf("%s:%d", config.Host, config.Port), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to imap server: %v", err)
	}

	err = dialer.Login(config.Username, config.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to login to imap server: %v", err)
	}

	// Create the cache folder
	hashedUserConf := md5.Sum([]byte(fmt.Sprintf("%s-%s:%d", config.Username, config.Host, config.Port)))

	cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "imap", fmt.Sprintf("body-%x", hashedUserConf[:]))
	err = os.MkdirAll(cacheFolder, 0755)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cache folder: %w", err)
	}

	// Open the badger database encrypted with the toke
	options := badger.DefaultOptions(cacheFolder).WithEncryptionKey(hashedUserConf[:]).
		WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 29).
		WithIndexCacheSize(2 << 26) // Up to 1GB of cache
	db, err := badger.Open(options)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open badger database: %w", err)
	}

	return &itemsBodyTable{
			db:          db,
			dialer:      dialer,
			username:    config.Username,
			password:    config.Password,
			dialerMutex: new(sync.Mutex),
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "uid",
					Type:        rpc.ColumnTypeInt,
					Description: "The unique identifier of the email",
				},
				{
					Name:        "subject",
					Type:        rpc.ColumnTypeString,
					Description: "The subject of the email",
				},
				{
					Name:        "sent_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date the email was sent (RFC3339)",
				},
				{
					Name:        "received_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date the email was received (RFC3339)",
				},
				{
					Name:        "_from",
					Type:        rpc.ColumnTypeJSON,
					Description: "An array of JSON objects with the email and name of the sender. ([{\"email\": \"john@example.com\", \"name\": \"John Doe\"}])",
				},
				{
					Name:        "to",
					Type:        rpc.ColumnTypeString,
					Description: "An array of JSON objects with the email and name of the sender. ([{\"email\": \"john@example.com\", \"name\": \"John Doe\"}])",
				},
				{
					Name:        "reply_to",
					Type:        rpc.ColumnTypeString,
					Description: "An array of JSON objects with the email and name of the sender. ([{\"email\": \"john@example.com\", \"name\": \"John Doe\"}])",
				},
				{
					Name:        "cc",
					Type:        rpc.ColumnTypeString,
					Description: "An array of JSON objects with the email and name of the sender. ([{\"email\": \"john@example.com\", \"name\": \"John Doe\"}])",
				},
				{
					Name:        "bcc",
					Type:        rpc.ColumnTypeString,
					Description: "An array of JSON objects with the email and name of the sender. ([{\"email\": \"john@example.com\", \"name\": \"John Doe\"}])",
				},
				{
					Name:        "message_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the email",
				},
				{
					Name:        "flags",
					Type:        rpc.ColumnTypeJSON,
					Description: "An array of flags of the email. Flags are: Seen, Answered, Flagged, Deleted, Draft, Recent",
				},
				{
					Name:        "size",
					Type:        rpc.ColumnTypeInt,
					Description: "The size of the email in bytes",
				},
				{
					Name:        "folder",
					Type:        rpc.ColumnTypeString,
					Description: "The folder of the email",
				},
				{
					Name:        "body",
					Type:        rpc.ColumnTypeString,
					Description: "The HTML body of the email",
				},
			},
		}, nil
}

type itemsBodyTable struct {
	dialer          *client.Client
	dialerMutex     *sync.Mutex
	db              *badger.DB
	mailCountFolder map[string]uint32 // To iterate over all the UIDs
	folders         []string
	username        string
	password        string
}

type itemsBodyCursor struct {
	folderIndex     int
	folders         *[]string
	folderSelected  bool
	pageSize        uint32
	offset          uint32
	dialer          *client.Client
	dialerMutex     *sync.Mutex
	db              *badger.DB
	mailCountFolder *map[string]uint32 // To iterate over all the UIDs
	username        string
	password        string
}

func (t *itemsBodyCursor) fillFolders() error {
	// Get the list of folders if we haven't already
	if *t.folders == nil {
		mailboxes := make(chan *imap.MailboxInfo, 10)
		done := make(chan error, 1)
		go func() {
			done <- t.dialer.List("", "*", mailboxes)
		}()

		for m := range mailboxes {
			if m == nil {
				continue
			}
			*t.folders = append(*t.folders, m.Name)
		}
		err := <-done
		if err != nil {
			return fmt.Errorf("failed to get folders: %v", err)
		}
	}

	if *t.mailCountFolder == nil {
		*t.mailCountFolder = make(map[string]uint32)
	}

	return nil
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *itemsBodyCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the list of UIDs if we haven't already
	if *t.mailCountFolder == nil {
		err := t.fillFolders()
		if err != nil {
			return nil, true, err
		}
	}

	// If we are over the number of folders, return no rows
	if t.folderIndex >= len(*t.folders) {
		return nil, true, nil
	}

	// Get the current folder
	folder := (*t.folders)[t.folderIndex]

	// Get the count of emails in the folder
	if !t.folderSelected {
		mbox, err := t.dialer.Select(folder, true) // Select the folder as read-only

		if err != nil {
			if err.Error() == "Not logged in" {
				t.dialerMutex.Lock()
				err = t.dialer.Login(t.username, t.password)
				t.dialerMutex.Unlock()
				if err != nil {
					return nil, true, fmt.Errorf("failed to login to imap server: %v", err)
				}
			} else {
				// Skip the folder
				t.folderIndex++
				t.folderSelected = false
				return nil, t.folderIndex >= len(*t.folders), nil
			}
		}
		// Save the number of emails in the folder
		(*t.mailCountFolder)[folder] = mbox.Messages
		// Move to the next folder that has emails
		for (*t.mailCountFolder)[folder] == 0 {
			t.folderIndex++
			if t.folderIndex >= len(*t.folders) {
				return nil, true, nil
			}
			folder = (*t.folders)[t.folderIndex]
			mbox, err = t.dialer.Select(folder, true) // Select the folder as read-only
			if err != nil {
				return nil, true, fmt.Errorf("failed to select folder %s: %v", folder, err)
			}
			// Save the number of emails in the folder
			(*t.mailCountFolder)[folder] = mbox.Messages
		}
	}
	t.folderSelected = true

	// Check if any emails are to check in the folder
	mailCount := (*t.mailCountFolder)[folder]
	if t.offset >= mailCount || mailCount == 0 {
		t.folderIndex++
		t.folderSelected = false
		t.offset = 0
		return nil, t.folderIndex >= len(*t.folders), nil
	}

	rows := make([][]interface{}, 0, t.pageSize)
	// Fetch the next page of emails
	// Try from cache first and if not found, fetch from the server
	err := t.db.View(func(txn *badger.Txn) error {
		// Check if the page is in the cache
		key := fmt.Sprintf("%s-%d-%d", folder, t.offset, t.pageSize)
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		// Decode the page
		return item.Value(func(val []byte) error {
			decoder := gob.NewDecoder(bytes.NewReader(val))
			return decoder.Decode(&rows)
		})
	})

	if err != nil {
		seqSet := new(imap.SeqSet)
		// Use min to avoid going over the number of emails
		// resulting in The specified message set is invalid.
		seqSet.AddRange(uint32(t.offset+1), min(mailCount, t.offset+t.pageSize))

		tempBody := &imap.BodySectionName{}
		messages := make(chan *imap.Message, 10)
		done := make(chan error, 1)
		go func() {
			done <- t.dialer.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822Size, imap.FetchInternalDate,
				imap.FetchUid, tempBody.FetchItem(),
			}, messages)
		}()

		for msg := range messages {
			if msg == nil {
				log.Printf("Message is nil")
				continue
			}
			subject := interface{}(nil)
			sent_at := interface{}(nil)
			received_at := interface{}(nil)
			from := interface{}(nil)
			to := interface{}(nil)
			reply_to := interface{}(nil)
			cc := interface{}(nil)
			bcc := interface{}(nil)
			if msg.Envelope != nil {
				subject = msg.Envelope.Subject
				sent_at = msg.Envelope.Date.Format(time.RFC3339)
				received_at = msg.InternalDate.Format(time.RFC3339)
				from = serializeJSON(msg.Envelope.From)
				to = serializeJSON(msg.Envelope.To)
				reply_to = serializeJSON(msg.Envelope.ReplyTo)
				cc = serializeJSON(msg.Envelope.Cc)
				bcc = serializeJSON(msg.Envelope.Bcc)
			}

			r := msg.GetBody(tempBody)
			if r == nil {
				log.Printf("Failed to get body for message %d", msg.Uid)
				continue
			}

			// Read the body
			mailReader, err := mail.CreateReader(r)
			if err != nil {
				log.Printf("Failed to create mail reader for message %d: %v", msg.Uid, err)
				continue
			}

			// Get the HTML part
			body := ""
			for {
				part, err := mailReader.NextPart()
				if err != nil {
					break
				}

				// Check if the part is text/plain or text/html
				if strings.HasPrefix(part.Header.Get("Content-Type"), "text/") {
					buf := new(bytes.Buffer)
					_, err := buf.ReadFrom(part.Body)
					if err != nil {
						log.Printf("Failed to read body for message %d: %v", msg.Uid, err)
						continue
					}
					body = buf.String()
					break
				}

			}

			rows = append(rows, []interface{}{
				msg.Uid,
				subject,
				sent_at,
				received_at,
				from,
				to,
				reply_to,
				cc,
				bcc,
				msg.Envelope.MessageId,
				msg.Flags,
				msg.Size,
				folder,
				body,
			})
		}

		err := <-done

		if err != nil {
			if err.Error() == "Not logged in" {
				t.dialerMutex.Lock()
				err = t.dialer.Login(t.username, t.password)
				if err != nil {
					return nil, true, fmt.Errorf("failed to login to imap server: %v", err)
				}
				t.dialerMutex.Unlock()
				return nil, true, nil // Retry the query
			}
			return nil, true, fmt.Errorf("failed to fetch emails: %v", err)
		}

		log.Printf("Requested %d emails, got %d", t.pageSize, len(rows))

		// Save the page in the cache unless it's the last page
		// This is to avoid saving the last page that might not be full
		// and also, if a new email is received, it will be visible in the next query
		if t.offset+t.pageSize < mailCount {
			err := t.db.Update(func(txn *badger.Txn) error {
				key := fmt.Sprintf("%s-%d-%d", folder, t.offset, t.pageSize)
				var buf bytes.Buffer
				encoder := gob.NewEncoder(&buf)
				err := encoder.Encode(rows)
				if err != nil {
					return err
				}
				e := badger.NewEntry([]byte(key), buf.Bytes()).WithTTL(30 * 24 * time.Hour) // We cache the page for 30 days because emails don't change
				return txn.SetEntry(e)
			})
			if err != nil {
				log.Printf("Failed to save page in cache: %v", err)
			}
		}

	}

	// Update the offset
	t.offset += t.pageSize

	// If we are over the number of UIDs, move to the next folder
	if t.offset >= (*t.mailCountFolder)[folder] {
		t.folderIndex++
		t.offset = 0
		t.folderSelected = false
	}

	return rows, t.folderIndex >= len(*t.folders), nil
}

// Create a new cursor that will be used to read rows
func (t *itemsBodyTable) CreateReader() rpc.ReaderInterface {
	if t.dialerMutex == nil {
		t.dialerMutex = &sync.Mutex{}
	}
	return &itemsBodyCursor{
		folders:         &t.folders,
		dialer:          t.dialer,
		dialerMutex:     t.dialerMutex,
		db:              t.db,
		mailCountFolder: &t.mailCountFolder,
		pageSize:        50, // Default page size resulting in 350ms per page in France with outlook.com
		username:        t.username,
		password:        t.password,
	}
}

// A destructor to clean up resources
func (t *itemsBodyTable) Close() error {
	t.db.Close()
	t.dialer.Logout()
	return nil
}
