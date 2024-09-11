package helper_test

import (
	"os"
	"testing"
	"time"

	"github.com/julien040/anyquery/rpc/helper"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	// Delete the cache directory
	path := helper.GetCachePath("test")
	err := os.RemoveAll(path)
	require.NoError(t, err)

	t.Run("Simple cache", func(t *testing.T) {
		cache1, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"test", "cache1"},
			EncryptionKey: []byte("abcdefghijklmnop"), // A 16 bytes key
		})
		require.NoError(t, err)
		defer cache1.Close()

		// Set the value in the cache
		rows := [][]interface{}{
			{"a", "b"},
			{"c", "d"},
		}

		metadata := map[string]interface{}{
			"hello":  "world",
			"foo":    "bar",
			"number": 42,
		}

		err = cache1.Set("key", rows, metadata, time.Hour)
		require.NoError(t, err)

		// Get the value from the cache
		rows2, metadata2, err := cache1.Get("key")
		require.NoError(t, err)

		require.Equal(t, rows, rows2)
		require.Equal(t, metadata, metadata2)
	})

	// Two different caches for the same paths that must be independent
	t.Run("Two caches for the same paths", func(t *testing.T) {
		cache1, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"test", "cache2"},
			EncryptionKey: []byte("abcdefghijklmnop"), // A 16 bytes key
		})
		require.NoError(t, err)
		defer cache1.Close()

		cache2, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"test", "cache2"},
			EncryptionKey: []byte("abcdefghijklmnop"), // A 16 bytes key
		})
		require.NoError(t, err)
		defer cache2.Close()

		// Set the value in the cache
		rows := [][]interface{}{
			{"a", "b"},
			{"c", "d"},
		}

		metadata := map[string]interface{}{
			"hello":  "world",
			"foo":    "bar",
			"number": 42,
		}

		// Set the value in the cache for the first cache
		err = cache1.Set("key", rows, metadata, time.Hour)
		require.NoError(t, err)

		rows2, metadata2, err := cache1.Get("key")
		require.NoError(t, err)

		require.Equal(t, rows, rows2)
		require.Equal(t, metadata, metadata2)

		// The second cache must be independent and not have the value
		rows2, metadata2, err = cache2.Get("key")
		require.Error(t, err)

		require.Nil(t, rows2)
		require.Nil(t, metadata2)
	})

	t.Run("Cache values expires", func(t *testing.T) {
		cache1, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"test", "cache3"},
			EncryptionKey: []byte("abcdefghijklmnop"), // A 16 bytes key
		})
		require.NoError(t, err)
		defer cache1.Close()

		// Set the value in the cache
		rows := [][]interface{}{
			{"a", "b"},
			{"c", "d"},
		}

		metadata := map[string]interface{}{
			"hello":  "world",
			"foo":    "bar",
			"number": 42,
		}

		// Set the value in the cache
		err = cache1.Set("key", rows, metadata, time.Second)
		require.NoError(t, err)

		// Check the value is in the cache
		rows2, metadata2, err := cache1.Get("key")
		require.NoError(t, err)
		require.Equal(t, rows, rows2)
		require.Equal(t, metadata, metadata2)

		// Wait for the value to expire
		time.Sleep(time.Second)

		// The value must not be in the cache
		rows2, metadata2, err = cache1.Get("key")
		require.Error(t, err)

		require.Nil(t, rows2)
		require.Nil(t, metadata2)
	})

	t.Run("Cache values can be deleted", func(t *testing.T) {
		cache1, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"test", "cache4"},
			EncryptionKey: []byte("abcdefghijklmnop"), // A 16 bytes key
		})
		require.NoError(t, err)
		defer cache1.Close()

		// Set the value in the cache
		rows := [][]interface{}{
			{"a", "b"},
			{"c", "d"},
		}

		metadata := map[string]interface{}{
			"hello":  "world",
			"foo":    "bar",
			"number": 42,
		}

		// Set the value in the cache
		err = cache1.Set("key", rows, metadata, time.Hour)
		require.NoError(t, err)

		// Check the value is in the cache
		rows2, metadata2, err := cache1.Get("key")
		require.NoError(t, err)
		require.Equal(t, rows, rows2)
		require.Equal(t, metadata, metadata2)

		// Delete the value from the cache
		err = cache1.Delete("key")
		require.NoError(t, err)

		// The value must not be in the cache
		rows2, metadata2, err = cache1.Get("key")
		require.Error(t, err)

		require.Nil(t, rows2)
		require.Nil(t, metadata2)
	})

	t.Run("Cache values can be deleted with a prefix", func(t *testing.T) {
		cache1, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"test", "cache5"},
			EncryptionKey: []byte("abcdefghijklmnop"), // A 16 bytes key
			MaxSize:       2 << 20,                    // 2MB
			MaxMemSize:    2 << 20,                    // 2MB
		})
		require.NoError(t, err)
		defer cache1.Close()

		// Set the value in the cache
		rows := [][]interface{}{
			{"a", "b"},
			{"c", "d"},
		}

		metadata := map[string]interface{}{
			"hello":  "world",
			"foo":    "bar",
			"number": 42,
		}

		// Set the value in the cache
		err = cache1.Set("key-hello", rows, metadata, time.Hour)
		require.NoError(t, err)

		err = cache1.Set("key-world", rows, metadata, time.Hour)
		require.NoError(t, err)

		err = cache1.Set("not-the-prefix", rows, metadata, time.Hour)
		require.NoError(t, err)

		// Check the value is in the cache
		rows2, metadata2, err := cache1.Get("key-hello")
		require.NoError(t, err)
		require.Equal(t, rows, rows2)
		require.Equal(t, metadata, metadata2)

		// Delete the value from the cache
		err = cache1.ClearWithPrefix("key")
		require.NoError(t, err)

		// key-hello and key-world must not be in the cache
		// but not-the-prefix must still be in the cache
		_, _, err = cache1.Get("key-hello")
		require.Error(t, err)

		_, _, err = cache1.Get("key-world")
		require.Error(t, err)

		rows2, metadata2, err = cache1.Get("not-the-prefix")
		require.NoError(t, err)
		require.Equal(t, rows, rows2)
		require.Equal(t, metadata, metadata2)

		// Clear the whole cache
		err = cache1.Clear()
		require.NoError(t, err)

		// All the values must be deleted
		_, _, err = cache1.Get("not-the-prefix")
		require.Error(t, err)

	})

	t.Run("A closed cache cannot be used and free the lock on the directory", func(t *testing.T) {
		cache1, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"test", "cache6"},
			EncryptionKey: []byte("abcdefghijklmnop"), // A 16 bytes key
		})
		require.NoError(t, err)

		// Set the value in the cache
		rows := [][]interface{}{
			{"a", "b"},
			{"c", "d"},
		}

		metadata := map[string]interface{}{
			"hello":  "world",
			"foo":    "bar",
			"number": 42,
		}

		// Set the value in the cache
		err = cache1.Set("key", rows, metadata, time.Hour)
		require.NoError(t, err)

		// Close the cache
		err = cache1.Close()
		require.NoError(t, err)

		// The cache is closed, it must not be possible to use it
		_, _, err = cache1.Get("key")
		require.Error(t, err)

		// The cache directory must be free
		cache2, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"test", "cache6"},
			EncryptionKey: []byte("abcdefghijklmnop"), // A 16 bytes key
		})
		require.NoError(t, err)
		defer cache2.Close()

		// The cache must be usable
		rows2, metadata2, err := cache2.Get("key")
		require.NoError(t, err)
		require.Equal(t, rows, rows2)
		require.Equal(t, metadata, metadata2)
	})

}
