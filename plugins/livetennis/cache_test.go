package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testKey = "local-test-secret"

var testNow = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func newTestClient(dir, endpoint, key string, now func() time.Time) *snapshotClient {
	return &snapshotClient{
		key: key, cacheDir: dir, endpoint: endpoint, now: now,
		http: &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}},
	}
}

func snapshotPath(dir, key string) string {
	return filepath.Join(dir, fmt.Sprintf("%x.json", sha256.Sum256([]byte(key))))
}

func readTestSnapshot(t *testing.T, dir, key string) cachedSnapshot {
	t.Helper()
	content, err := os.ReadFile(snapshotPath(dir, key))
	if err != nil {
		t.Fatal(err)
	}
	var snapshot cachedSnapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), key) {
		t.Fatal("cache contains the API key")
	}
	return snapshot
}

func requireLoadError(t *testing.T, client *snapshotClient) {
	t.Helper()
	if snapshot, err := client.load(); err == nil || snapshot != nil {
		t.Fatalf("load() = %v, %v; expected an error", snapshot, err)
	} else if strings.Contains(err.Error(), client.key) {
		t.Fatal("error contains the API key")
	}
}

func TestSnapshotPersistsCadence(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.RequestURI() != "/api/public/v1/matches?status=live&limit=200&offset=0" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
		if r.Header.Get("X-API-Key") != testKey || r.Header.Get("Accept") != "application/json" {
			t.Error("request headers are incorrect")
		}
		io.WriteString(w, `{"data":[{"id":17,"tournament":"Test","status":"live","score":null}],"pagination":{"total":1}}`)
	}))
	defer server.Close()
	dir, now := t.TempDir(), testNow
	endpoint := server.URL + "/api/public/v1/matches?status=live&limit=200&offset=0"
	client := newTestClient(dir, endpoint, testKey, func() time.Time { return now })
	first, err := client.load()
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Data) != 1 || first.Data[0].ID != 17 || !first.FetchedAt.Equal(now) {
		t.Fatalf("unexpected snapshot: %+v", first)
	}
	if want := testNow.Add(15 * time.Minute); !readTestSnapshot(t, dir, testKey).NextAttempt.Equal(want) {
		t.Fatalf("next attempt must be %s", want)
	}
	for _, elapsed := range []time.Duration{0, time.Minute, 15*time.Minute - time.Nanosecond} {
		now = testNow.Add(elapsed)
		freshClient := newTestClient(dir, endpoint, testKey, func() time.Time { return now })
		snapshot, err := freshClient.load()
		if err != nil || snapshot.Data[0].ID != 17 || !snapshot.FetchedAt.Equal(testNow) {
			t.Fatalf("cached load at %s = %v, %v", elapsed, snapshot, err)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("cache interval made %d requests, want 1", got)
	}
	now = testNow.Add(15 * time.Minute)
	if _, err := client.load(); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("boundary made %d requests, want 2", got)
	}
	info, err := os.Stat(snapshotPath(dir, testKey))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("cache permissions = %o; group and other must have no access", info.Mode().Perm())
	}
}

func TestFailedAttemptsPersistCadence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", http.StatusUnauthorized, testKey},
		{"forbidden", http.StatusForbidden, testKey},
		{"server error", http.StatusInternalServerError, testKey},
		{"rate limited", http.StatusTooManyRequests, testKey},
		{"malformed JSON", http.StatusOK, `{"data":[` + testKey},
		{"missing data", http.StatusOK, `{}`},
		{"wrong data type", http.StatusOK, `{"data":{}}`},
		{"null data", http.StatusOK, `{"data":null}`},
		{"null match", http.StatusOK, `{"data":[null]}`},
		{"invalid field type", http.StatusOK, `{"data":[{"id":"bad"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			dir, now := t.TempDir(), testNow
			client := newTestClient(dir, server.URL, testKey, func() time.Time { return now })
			requireLoadError(t, client)
			snapshot := readTestSnapshot(t, dir, testKey)
			if !snapshot.NextAttempt.Equal(testNow.Add(15*time.Minute)) || snapshot.Error == "" {
				t.Fatalf("failed request reservation: %+v", snapshot)
			}
			now = testNow.Add(15*time.Minute - time.Nanosecond)
			requireLoadError(t, newTestClient(dir, server.URL, testKey, func() time.Time { return now }))
			if requests.Load() != 1 {
				t.Fatal("restarted client retried before the interval")
			}
			now = testNow.Add(15 * time.Minute)
			requireLoadError(t, client)
			if requests.Load() != 2 {
				t.Fatal("failed request was not retried at the interval boundary")
			}
		})
	}
}

type failingTransport struct {
	attempts *atomic.Int32
}

func (transport failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.attempts.Add(1)
	return nil, errors.New("transport failed for " + testKey)
}

func TestNetworkFailurePersistsCadence(t *testing.T) {
	var attempts atomic.Int32
	dir := t.TempDir()
	client := newTestClient(dir, "http://example.invalid", testKey, func() time.Time { return testNow })
	client.http.Transport = failingTransport{&attempts}
	requireLoadError(t, client)
	restarted := newTestClient(dir, client.endpoint, testKey, func() time.Time { return testNow.Add(time.Minute) })
	restarted.http.Transport = failingTransport{&attempts}
	requireLoadError(t, restarted)
	if attempts.Load() != 1 {
		t.Fatal("network failure was retried before the interval")
	}
	if !readTestSnapshot(t, dir, testKey).NextAttempt.Equal(testNow.Add(15 * time.Minute)) {
		t.Fatal("network failure did not reserve the interval")
	}
}

func TestRetryAfterExtendsCadence(t *testing.T) {
	for _, tc := range []struct {
		name, header string
		wait         time.Duration
	}{
		{"seconds", "1800", 30 * time.Minute},
		{"date", testNow.Add(time.Hour).Format(http.TimeFormat), time.Hour},
		{"shorter than floor", "30", 15 * time.Minute},
		{"past date", testNow.Add(-time.Hour).Format(http.TimeFormat), 15 * time.Minute},
		{"invalid", "unknown", 15 * time.Minute},
		{"negative", "-5", 15 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Retry-After", tc.header)
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			defer server.Close()
			dir, now := t.TempDir(), testNow
			client := newTestClient(dir, server.URL, testKey, func() time.Time { return now })
			requireLoadError(t, client)
			if got := readTestSnapshot(t, dir, testKey).NextAttempt; !got.Equal(testNow.Add(tc.wait)) {
				t.Fatalf("next attempt = %s, want %s", got, testNow.Add(tc.wait))
			}
			now = testNow.Add(tc.wait - time.Nanosecond)
			requireLoadError(t, newTestClient(dir, server.URL, testKey, func() time.Time { return now }))
			if requests.Load() != 1 {
				t.Fatal("client ignored persisted Retry-After")
			}
			now = testNow.Add(tc.wait)
			requireLoadError(t, client)
			if requests.Load() != 2 {
				t.Fatal("client did not retry at Retry-After boundary")
			}
		})
	}
}

func TestSnapshotConcurrentClientsShareKey(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()
	dir, start := t.TempDir(), make(chan struct{})
	var wait sync.WaitGroup
	for i := 0; i < 12; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			client := newTestClient(dir, server.URL, testKey, func() time.Time { return testNow })
			if _, err := client.load(); err != nil {
				t.Errorf("concurrent load failed: %v", err)
			}
		}()
	}
	close(start)
	wait.Wait()
	if requests.Load() != 1 {
		t.Fatalf("concurrent clients made %d requests, want 1", requests.Load())
	}
}

func TestSnapshotSeparateKeys(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	for _, key := range []string{testKey, "other-local-test-key", testKey, "other-local-test-key"} {
		client := newTestClient(dir, server.URL, key, func() time.Time { return testNow })
		if _, err := client.load(); err != nil {
			t.Fatal(err)
		}
		readTestSnapshot(t, dir, key)
	}
	if requests.Load() != 2 {
		t.Fatalf("independent keys made %d requests, want 2", requests.Load())
	}
}

func TestSnapshotDailyQuota(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(status)
				io.WriteString(w, `{"data":[]}`)
			}))
			defer server.Close()
			dir := t.TempDir()
			for minute := 0; minute < 24*60; minute++ {
				now := testNow.Add(time.Duration(minute) * time.Minute)
				client := newTestClient(dir, server.URL, testKey, func() time.Time { return now })
				_, err := client.load()
				if (err != nil) != (status != http.StatusOK) {
					t.Fatalf("minute %d: unexpected error %v", minute, err)
				}
			}
			if requests.Load() != 96 {
				t.Fatalf("24 hours made %d attempts, want 96", requests.Load())
			}
		})
	}
}

func TestInvalidCacheFailsClosed(t *testing.T) {
	for _, content := range []string{"{", "{}", `{"next_attempt":"invalid"}`, `{"next_attempt":"2026-01-01T00:15:00Z","data":{}}`} {
		t.Run(content, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				io.WriteString(w, `{"data":[]}`)
			}))
			defer server.Close()
			dir := t.TempDir()
			if err := os.WriteFile(snapshotPath(dir, testKey), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			requireLoadError(t, newTestClient(dir, server.URL, testKey, func() time.Time { return testNow }))
			if requests.Load() != 0 {
				t.Fatal("invalid cache caused a request")
			}
		})
	}
}

func TestUnwritableCacheFailsClosed(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()
	t.Run("cache directory cannot be created", func(t *testing.T) {
		obstacle := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(obstacle, []byte("not a directory"), 0600); err != nil {
			t.Fatal(err)
		}
		client := newTestClient(filepath.Join(obstacle, "cache"), server.URL, testKey, func() time.Time { return testNow })
		requireLoadError(t, client)
	})
	t.Run("snapshot cannot be read or written", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(snapshotPath(dir, testKey), 0700); err != nil {
			t.Fatal(err)
		}
		requireLoadError(t, newTestClient(dir, server.URL, testKey, func() time.Time { return testNow }))
	})
	if requests.Load() != 0 {
		t.Fatal("unwritable cache caused a request")
	}
}

func TestSnapshotReservesBeforeHTTP(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := readTestSnapshot(t, dir, testKey)
		if snapshot.Error == "" || !snapshot.NextAttempt.Equal(testNow.Add(15*time.Minute)) {
			t.Error("attempt was not reserved before HTTP")
		}
		io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()
	if _, err := newTestClient(dir, server.URL, testKey, func() time.Time { return testNow }).load(); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotProcessHelper(t *testing.T) {
	dir := os.Getenv("LIVETENNIS_TEST_CACHE")
	if dir == "" {
		return
	}
	client := newTestClient(dir, os.Getenv("LIVETENNIS_TEST_ENDPOINT"), testKey, func() time.Time { return testNow })
	if _, err := client.load(); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotAcrossProcesses(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(50 * time.Millisecond)
		io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()
	dir, start := t.TempDir(), make(chan struct{})
	var wait sync.WaitGroup
	for i := 0; i < 2; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSnapshotProcessHelper$")
			command.Env = append(os.Environ(), "LIVETENNIS_TEST_CACHE="+dir, "LIVETENNIS_TEST_ENDPOINT="+server.URL)
			if output, err := command.CombinedOutput(); err != nil {
				t.Errorf("child process: %v\n%s", err, output)
			}
		}()
	}
	close(start)
	wait.Wait()
	if requests.Load() != 1 {
		t.Fatalf("separate processes made %d requests, want 1", requests.Load())
	}
}
