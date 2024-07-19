package main

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func itemsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
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

	cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "imap", fmt.Sprintf("%x", hashedUserConf[:]))
	err = os.MkdirAll(cacheFolder, 0755)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cache folder: %w", err)
	}

	// Open the badger database encrypted with the toke
	options := badger.DefaultOptions(cacheFolder).WithEncryptionKey(hashedUserConf[:]).
		WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 26).
		WithIndexCacheSize(2 << 29) // Up to 1GB of cache
	db, err := badger.Open(options)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open badger database: %w", err)
	}

	return &itemsTable{
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
					Name: "uid",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "subject",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "sent_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "received_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "_from",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "to",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "reply_to",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "cc",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "bcc",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "message_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "flags",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "size",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "folder",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type itemsTable struct {
	dialer          *client.Client
	dialerMutex     *sync.Mutex
	db              *badger.DB
	mailCountFolder map[string]uint32 // To iterate over all the UIDs
	folders         []string
	username        string
	password        string
}

type itemsCursor struct {
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

func (t *itemsCursor) fillFolders() error {
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
func (t *itemsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
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

		messages := make(chan *imap.Message, 10)
		done := make(chan error, 1)
		go func() {
			done <- t.dialer.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822Size, imap.FetchInternalDate,
				imap.FetchUid}, messages)
		}()

		for msg := range messages {
			if msg == nil {
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
func (t *itemsTable) CreateReader() rpc.ReaderInterface {
	if t.dialerMutex == nil {
		t.dialerMutex = &sync.Mutex{}
	}
	return &itemsCursor{
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

// A slice of rows to insert
func (t *itemsTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *itemsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *itemsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *itemsTable) Close() error {
	return nil
}

// Serialize a value to JSON and return nil if the value is not serializable
// Therefore, nil will be replaced as NULL in the database
func serializeJSON(v interface{}) interface{} {
	// If of type adressMail, we convert it to an array of struct {email, name}
	if v == nil {
		return nil
	}

	if adressMail, ok := v.([]*imap.Address); ok {
		arrayEmail := make([]struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}, 0, len(adressMail))
		for _, email := range adressMail {
			if email == nil {
				continue
			}
			arrayEmail = append(arrayEmail, struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			}{
				Email: email.Address(),
				Name:  email.PersonalName,
			})
		}

		v = arrayEmail

	}

	serialized, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(serialized)
}
