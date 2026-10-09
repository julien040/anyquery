package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofrs/flock"
)

// 24 hours / 900 seconds permits at most 96 attempts per day, below the free limit of 100.
const requestInterval = 15 * time.Minute
const maxResponseSize = 16 << 20

type snapshotClient struct {
	key      string
	cacheDir string
	endpoint string
	http     *http.Client
	now      func() time.Time
}

type cachedSnapshot struct {
	NextAttempt time.Time `json:"next_attempt"`
	FetchedAt   time.Time `json:"fetched_at"`
	Data        []match   `json:"data"`
	Error       string    `json:"error,omitempty"`
}

func (c *snapshotClient) load() (*cachedSnapshot, error) {
	if err := os.MkdirAll(c.cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("create snapshot cache: %w", err)
	}
	path := filepath.Join(c.cacheDir, fmt.Sprintf("%x.json", sha256.Sum256([]byte(c.key))))
	lock := flock.New(path + ".lock")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	locked, err := lock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil || !locked {
		return nil, fmt.Errorf("snapshot cache is busy")
	}
	defer lock.Close()

	snapshot := &cachedSnapshot{}
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read snapshot cache: %w", err)
	}
	if err == nil {
		if err := json.Unmarshal(content, snapshot); err != nil || snapshot.NextAttempt.IsZero() {
			return nil, fmt.Errorf("invalid snapshot cache")
		}
	}
	now := c.now().UTC()
	if now.Before(snapshot.NextAttempt) {
		if snapshot.Error != "" {
			return nil, fmt.Errorf("%s; next attempt at %s", snapshot.Error, snapshot.NextAttempt.Format(time.RFC3339))
		}
		return snapshot, nil
	}

	// Reserve before making the request so failed attempts and process restarts count.
	snapshot = &cachedSnapshot{
		NextAttempt: now.Add(requestInterval),
		Error:       "previous snapshot request did not finish",
	}
	if err := saveSnapshot(path, snapshot); err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create snapshot request")
	}
	request.Header.Set("X-API-Key", c.key)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		snapshot.Error = "snapshot request failed"
	} else {
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			snapshot.Error = fmt.Sprintf("snapshot HTTP status %d", response.StatusCode)
			if response.StatusCode == http.StatusTooManyRequests {
				retryAt := retryAfter(response.Header.Get("Retry-After"), c.now().UTC())
				if retryAt.After(snapshot.NextAttempt) {
					snapshot.NextAttempt = retryAt
				}
			}
		} else {
			snapshot.Data, err = decodeMatches(response.Body)
			if err != nil {
				snapshot.Error = "invalid snapshot response"
			} else {
				snapshot.Error = ""
				snapshot.FetchedAt = c.now().UTC()
			}
		}
	}
	if err := saveSnapshot(path, snapshot); err != nil {
		return nil, err
	}
	if snapshot.Error != "" {
		return nil, fmt.Errorf("%s; next attempt at %s", snapshot.Error, snapshot.NextAttempt.Format(time.RFC3339))
	}
	return snapshot, nil
}

func saveSnapshot(path string, snapshot *cachedSnapshot) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".snapshot-*")
	if err != nil {
		return fmt.Errorf("write snapshot cache: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := json.NewEncoder(file).Encode(snapshot); err != nil {
		return fmt.Errorf("encode snapshot cache: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("save snapshot cache: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close snapshot cache: %w", err)
	}
	if err := publishSnapshot(file.Name(), path); err != nil {
		return fmt.Errorf("publish snapshot cache: %w", err)
	}
	return nil
}

func decodeMatches(body io.Reader) ([]match, error) {
	content, err := io.ReadAll(io.LimitReader(body, maxResponseSize+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxResponseSize {
		return nil, fmt.Errorf("snapshot response exceeds size limit")
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(content, &envelope); err != nil {
		return nil, err
	}
	if data := bytes.TrimSpace(envelope.Data); len(data) == 0 || data[0] != '[' {
		return nil, fmt.Errorf("snapshot data must be an array")
	}
	var decoded []*match
	if err := json.Unmarshal(envelope.Data, &decoded); err != nil {
		return nil, err
	}
	matches := make([]match, 0, len(decoded))
	for _, item := range decoded {
		if item == nil {
			return nil, fmt.Errorf("snapshot contains a null match")
		}
		matches = append(matches, *item)
	}
	return matches, nil
}

func retryAfter(value string, now time.Time) time.Time {
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		if seconds > int64((1<<63-1)/time.Second) {
			return now.Add(time.Duration(1<<63 - 1))
		}
		return now.Add(time.Duration(seconds) * time.Second)
	}
	date, _ := http.ParseTime(value)
	return date
}
