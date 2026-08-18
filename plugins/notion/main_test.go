package main

import (
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestClearCacheRemovesPersistedEntries(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions(t.TempDir()).WithLogger(nil))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("stale"), []byte("value"))
	}); err != nil {
		t.Fatal(err)
	}

	if err := clearCache(db); err != nil {
		t.Fatal(err)
	}

	err = db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte("stale"))
		return err
	})
	if err != badger.ErrKeyNotFound {
		t.Fatalf("expected stale entry to be removed, got %v", err)
	}
}
