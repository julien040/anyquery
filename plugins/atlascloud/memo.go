package main

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// SQLite may call a virtual table's filter function more than once for the
// same logical row set (joins, OR-clause decomposition, re-scans). For the
// llm and image tables every call costs money, so results are memoized
// in-memory, keyed on the full input tuple.
//
// The plugin cannot observe statement boundaries, so entries expire after a
// short TTL instead: long enough to cover re-scans within a statement, short
// enough that generations are not silently re-served across unrelated
// queries. Nothing is ever persisted to disk.
const (
	memoTTL = 5 * time.Minute
	// A submit failure is never billed (the request was rejected before a task
	// was created), so it is cached only briefly: long enough to absorb
	// re-scans within one statement, short enough that a manual retry is not
	// blocked for the full memo TTL.
	memoFailureTTL = 30 * time.Second
	memoMaxEntries = 1024
)

type memoEntry struct {
	row     []interface{}
	expires time.Time
}

type memoStore struct {
	mutex   sync.Mutex
	entries map[string]memoEntry
}

func newMemoStore() *memoStore {
	return &memoStore{entries: map[string]memoEntry{}}
}

func (m *memoStore) get(key string) ([]interface{}, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	entry, ok := m.entries[key]
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry.row, true
}

func (m *memoStore) set(key string, row []interface{}) {
	m.setWithTTL(key, row, memoTTL)
}

func (m *memoStore) setWithTTL(key string, row []interface{}, ttl time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	now := time.Now()

	// Evict expired entries once the store is full
	if len(m.entries) >= memoMaxEntries {
		for k, entry := range m.entries {
			if now.After(entry.expires) {
				delete(m.entries, k)
			}
		}
	}
	// Still full: drop the entry closest to expiry
	if len(m.entries) >= memoMaxEntries {
		oldestKey := ""
		oldestExpiry := time.Time{}
		for k, entry := range m.entries {
			if oldestKey == "" || entry.expires.Before(oldestExpiry) {
				oldestKey = k
				oldestExpiry = entry.expires
			}
		}
		delete(m.entries, oldestKey)
	}

	m.entries[key] = memoEntry{row: row, expires: now.Add(ttl)}
}

// memoKey hashes the input tuple of a generation call. A separator byte
// between parts prevents ("ab", "c") from colliding with ("a", "bc").
func memoKey(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
