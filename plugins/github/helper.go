package main

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-github/v63/github"
	"github.com/julien040/anyquery/rpc"
)

const ttl = time.Hour * 1

var regexToken = regexp.MustCompile(`^(gh[ps]_[a-zA-Z0-9]{36}|github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59})$`)

func getClient(args rpc.TableCreatorArgs) (*github.Client, string, error) {
	// Retrieve the token from the arguments
	token := ""

	rawToken, ok := args.UserConfig["token"]
	if !ok {
		return nil, "", fmt.Errorf("token not found")
	}

	token, ok = rawToken.(string)
	if !ok {
		return nil, "", fmt.Errorf("token is not a string")
	}

	// Check if the token is valid
	if !regexToken.MatchString(token) {
		return nil, "", fmt.Errorf("invalid token")
	}

	// Create a new client
	client := github.NewClient(nil).WithAuthToken(token)

	return client, token, nil

}

// Open a database in the cache path encrypted with a hash of the token
func openDatabase(tag, token string) (*badger.DB, error) {
	// Hash the token
	md5sum := md5.Sum([]byte(token))
	hash := fmt.Sprintf("%x", md5sum)

	cachePath := path.Join(xdg.CacheHome, "anyquery", "plugins", "github", tag, hash)

	// Make the directory
	os.MkdirAll(cachePath, 0700)

	// Open the database
	options := badger.DefaultOptions(cachePath).WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 21).
		WithIndexCacheSize(2 << 23).WithEncryptionKey(md5sum[:])

	db, err := badger.Open(options)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nil
}

func saveCache(db *badger.DB, key string, rows [][]interface{}) error {
	// Serialize the rows
	buf := bytes.Buffer{}

	enc := gob.NewEncoder(&buf)

	err := enc.Encode(rows)
	if err != nil {
		return fmt.Errorf("failed to encode cache: %w", err)
	}

	return db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), buf.Bytes()).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

func loadCache(db *badger.DB, key string, rows *[][]interface{}) error {
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			dec := gob.NewDecoder(bytes.NewReader(val))
			return dec.Decode(rows)
		})
	})

	if err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	return nil
}

func retrieveArgString(constraints rpc.QueryConstraint, columnID int) string {
	for _, c := range constraints.Columns {
		if c.ColumnID == columnID {
			switch rawVal := c.Value.(type) {
			case string:
				return rawVal
			case int64:
				return fmt.Sprintf("%d", rawVal)
			case float64:
				return fmt.Sprintf("%f", rawVal)
			}
		}
	}

	return ""

}

func serializeJSON(val interface{}) interface{} {
	if val == nil {
		return nil
	}
	serialized, err := json.Marshal(val)
	if err != nil {
		return nil
	}
	return string(serialized)
}
