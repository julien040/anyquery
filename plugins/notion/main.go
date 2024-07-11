package main

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

func main() {
	plugin := rpc.NewPlugin(databaseTable)
	plugin.Serve()
}

func clearCache(db *badger.DB) error {
	keys := make([][]byte, 0)
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			keys = append(keys, item.Key())
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			err := txn.Delete(key)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}
