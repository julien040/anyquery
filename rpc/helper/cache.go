package helper

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"golang.org/x/exp/rand"
)

// Cache is a simple key-value store that can store metadata with rows
//
// It is the recommended way to cache data in a plugin because it abstracts away the
// underlying storage and encryption
//
// It also helps the user to clear the cache with the SQL function clear_plugin_cache(plugin_name)
type Cache struct {
	db *badger.DB
}

// Get the value and the metadata of the key in the cache
func (c *Cache) Get(key string) ([][]interface{}, map[string]interface{}, error) {
	if c.db == nil {
		return nil, nil, errors.New("the cache is not initialized. Create a cache with NewCache")
	}

	// Get the value from the cache
	var value [][]interface{}
	var metadata map[string]interface{}
	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		// Deserialize the value
		err = item.Value(func(val []byte) error {
			buf := bytes.NewReader(val)
			dec := gob.NewDecoder(buf)
			return dec.Decode(&value)
		})

		if err != nil {
			return errors.Join(errors.New("failed to deserialize the value"), err)
		}

		// Get the metadata
		item, err = txn.Get([]byte(key + "-metadata"))
		if err != nil {
			return err
		}

		// Deserialize the metadata
		err = item.Value(func(val []byte) error {
			buf := bytes.NewReader(val)
			dec := gob.NewDecoder(buf)
			return dec.Decode(&metadata)
		})

		if err != nil {
			return errors.Join(errors.New("failed to deserialize the metadata"), err)
		}

		return nil
	})

	return value, metadata, err
}

// Save the key  in the cache with the value and the metadata for a duration of ttl (default to time.Hour if zero)
func (c *Cache) Set(key string, value [][]interface{}, metadata map[string]interface{},
	ttl time.Duration) error {
	if c.db == nil {
		return errors.New("the cache is not initialized. Create a cache with NewCache")
	}

	if ttl == 0 {
		ttl = time.Hour
	}

	// Save the key and metadata in the cache
	return c.db.Update(func(txn *badger.Txn) error {
		// Serialize the rows using Gob
		buf := bytes.Buffer{}
		enc := gob.NewEncoder(&buf)
		err := enc.Encode(value)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(key), buf.Bytes()).WithTTL(ttl)
		err = txn.SetEntry(e)
		if err != nil {
			errors.Join(errors.New("failed to save the value in the cache"), err)
		}

		// Serialize the metadata using Gob
		buf = bytes.Buffer{}
		enc = gob.NewEncoder(&buf)
		err = enc.Encode(metadata)
		if err != nil {
			return err
		}

		e = badger.NewEntry([]byte(key+"-metadata"), buf.Bytes()).WithTTL(ttl)
		err = txn.SetEntry(e)

		if err != nil {
			return errors.Join(errors.New("failed to save the metadata in the cache"), err)
		}
		return nil
	})

}

// Clear all the keys in the cache
func (c *Cache) Clear() error {
	if c.db == nil {
		return errors.New("the cache is not initialized. Create a cache with NewCache")
	}

	return c.db.DropAll()
}

func (c *Cache) ClearWithPrefix(prefix string) error {
	if c.db == nil {
		return errors.New("the cache is not initialized. Create a cache with NewCache")
	}

	return c.db.DropPrefix([]byte(prefix))
}

func (c *Cache) Delete(key string) error {
	if c.db == nil {
		return errors.New("the cache is not initialized. Create a cache with NewCache")
	}

	return c.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		if err != nil {
			return err
		}

		return txn.Delete([]byte(key + "-metadata"))
	})
}

func (c *Cache) Close() error {
	if c.db == nil {
		return errors.New("the cache is not initialized. Create a cache with NewCache")
	}

	err := c.db.Close()
	c.db = nil
	return err
}

var alphabet = "abcdefghijklmnopqrstuvwxyz"

func generateRandomString(length int) string {
	result := strings.Builder{}
	for i := 0; i < length; i++ {
		result.WriteByte(alphabet[rand.Intn(len(alphabet))])
	}
	return result.String()
}

type NewCacheArgs struct {
	// Where the cache will be stored (must be a unique set of paths for all running instances of the plugin).
	// Often, it's the plugin name followed by the MD5 hash of current user for an API. The last path should not
	// be a directory but a file.
	//
	// The cache will be stored at $XDG_CACHE_HOME/anyquery/paths[0]/.../paths[n]
	//
	// For example, []string{"trello", "boards", "5f4dcc3b5aa765d61d8327deb882cf99"} will store the cache at
	// $XDG_CACHE_HOME/anyquery/plugins/trello/boards/5f4dcc3b5aa765d61d8327deb882cf99
	Paths []string

	// The maximum on-disk size of the cache in bytes
	//
	// Default to 64MB (1 << 26)
	MaxSize int64

	// The maximum in-memory size of the cache of the cache in bytes
	//
	// Default to 8MB (1 << 23)
	MaxMemSize int64

	// An encryption key to encrypt the cache (required for security)
	//
	// The key must be 16, 24 or 32 bytes long for AES-128, AES-192 and AES-256 respectively
	EncryptionKey []byte
}

// Create a new cache at $XDG_CACHE_HOME/anyquery/paths[0]/.../paths[n]
func NewCache(args NewCacheArgs) (*Cache, error) {
	// Create the cache directory
	pathsSlice := []string{xdg.CacheHome, "anyquery", "plugins"}
	pathsSlice = append(pathsSlice, args.Paths...)
	pathCache := strings.TrimSuffix(path.Join(pathsSlice...), "/")

	// Verification of the arguments
	if args.MaxSize <= 0 {
		args.MaxSize = 1 << 26
	}

	if args.MaxMemSize <= 0 {
		args.MaxMemSize = 1 << 23
	}

	if len(args.EncryptionKey) == 0 {
		return nil, errors.New("encryption key is required")
	}

	if len(args.EncryptionKey) != 16 && len(args.EncryptionKey) != 24 && len(args.EncryptionKey) != 32 {
		return nil, errors.New("encryption key must be 16, 24 or 32 bytes long")
	}

	if len(args.Paths) < 1 {
		return nil, errors.New("paths must have at least one element (the plugin name)")
	}

	// Create the directory
	directory := path.Dir(pathCache)
	err := os.MkdirAll(directory, 0700)
	if err != nil {
		return nil, err
	}

	// Create the cache
	options := badger.DefaultOptions(pathCache).WithEncryptionKey(args.EncryptionKey).
		WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(args.MaxSize).
		WithIndexCacheSize(2 << 24)

	db, err := badger.Open(options)
	if err != nil && strings.HasPrefix(err.Error(), "Cannot acquire directory lock") {
		// If a db is already open, we append a random string to the path
		pathCache = path.Join(path.Dir(pathCache), path.Base(pathCache)+"-"+generateRandomString(8))
		options.Dir = pathCache
		options.ValueDir = pathCache
		db, err = badger.Open(options)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &Cache{db: db}, nil
}
