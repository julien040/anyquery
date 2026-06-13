package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

// Without a cap the job store would grow forever; only the most recent jobs
// are kept (the draft of the prediction stays available on Atlas Cloud)
const maxStoredJobs = 1000

const jobKeyPrefix = "job:"

// jobRecord is one submitted video generation job, persisted locally so jobs
// survive across anyquery sessions
type jobRecord struct {
	PredictionID string   `json:"prediction_id"`
	Model        string   `json:"model"`
	Prompt       string   `json:"prompt"`
	ImageURL     string   `json:"image_url,omitempty"`
	ExtraParams  string   `json:"extra_params,omitempty"`
	Status       string   `json:"status"`
	Outputs      []string `json:"outputs,omitempty"`
	Error        string   `json:"error,omitempty"`
	CreatedAt    string   `json:"created_at"`
}

// jobStore is a small embedded store (Badger) in the plugin cache directory
type jobStore struct {
	db  *badger.DB
	dir string
}

// Several connections of the same plugin process share one Badger handle per
// directory: Badger holds an exclusive lock on its directory, so a second
// open from the same process would fail
var jobStoreRegistry = struct {
	sync.Mutex
	entries map[string]*jobStoreEntry
}{entries: map[string]*jobStoreEntry{}}

type jobStoreEntry struct {
	store *jobStore
	refs  int
}

func acquireJobStore(dir string, encryptionKey []byte) (*jobStore, error) {
	jobStoreRegistry.Lock()
	defer jobStoreRegistry.Unlock()

	if entry, ok := jobStoreRegistry.entries[dir]; ok {
		entry.refs++
		return entry.store, nil
	}

	options := badger.DefaultOptions(dir).
		WithNumVersionsToKeep(1).
		WithCompactL0OnClose(true).
		WithValueLogFileSize(1 << 26).
		WithIndexCacheSize(2 << 23).
		WithEncryptionKey(encryptionKey)

	db, err := badger.Open(options)
	if err != nil {
		if strings.Contains(err.Error(), "Cannot acquire directory lock") {
			return nil, fmt.Errorf("the video job store is locked by another anyquery instance; close it and retry")
		}
		return nil, fmt.Errorf("failed to open the video job store: %w", err)
	}

	store := &jobStore{db: db, dir: dir}
	jobStoreRegistry.entries[dir] = &jobStoreEntry{store: store, refs: 1}
	return store, nil
}

func releaseJobStore(store *jobStore) error {
	if store == nil {
		return nil
	}
	jobStoreRegistry.Lock()
	defer jobStoreRegistry.Unlock()

	entry, ok := jobStoreRegistry.entries[store.dir]
	if !ok {
		return nil
	}
	entry.refs--
	if entry.refs <= 0 {
		delete(jobStoreRegistry.entries, store.dir)
		return entry.store.db.Close()
	}
	return nil
}

func (s *jobStore) put(job *jobRecord) error {
	if job.PredictionID == "" {
		return errors.New("cannot store a job without a prediction ID")
	}
	serialized, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to serialize the job: %w", err)
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(jobKeyPrefix+job.PredictionID), serialized)
	})
}

func (s *jobStore) delete(predictionID string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(jobKeyPrefix + predictionID))
	})
}

// list returns all stored jobs, newest first
func (s *jobStore) list() ([]*jobRecord, error) {
	jobs := []*jobRecord{}
	err := s.db.View(func(txn *badger.Txn) error {
		options := badger.DefaultIteratorOptions
		options.Prefix = []byte(jobKeyPrefix)
		it := txn.NewIterator(options)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			err := it.Item().Value(func(value []byte) error {
				job := &jobRecord{}
				if err := json.Unmarshal(value, job); err != nil {
					// Skip corrupted records instead of failing the query
					return nil
				}
				jobs = append(jobs, job)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list the video jobs: %w", err)
	}

	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].CreatedAt != jobs[j].CreatedAt {
			return jobs[i].CreatedAt > jobs[j].CreatedAt
		}
		return jobs[i].PredictionID > jobs[j].PredictionID
	})
	return jobs, nil
}

// prune deletes the oldest jobs so that at most maxJobs remain
func (s *jobStore) prune(maxJobs int) error {
	jobs, err := s.list()
	if err != nil {
		return err
	}
	if len(jobs) <= maxJobs {
		return nil
	}
	// jobs is sorted newest first: everything past maxJobs is the oldest
	for _, job := range jobs[maxJobs:] {
		if err := s.delete(job.PredictionID); err != nil {
			return err
		}
	}
	return nil
}
