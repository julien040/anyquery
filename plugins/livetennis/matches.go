package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julien040/anyquery/rpc"
)

const matchesURL = "https://api.livetennisapi.com/api/public/v1/matches?status=live&limit=200&offset=0"

func matchesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	key, ok := args.UserConfig["api_key"].(string)
	if !ok || strings.TrimSpace(key) == "" {
		return nil, nil, fmt.Errorf("api_key must be a nonempty string")
	}
	key = strings.TrimSpace(key)
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, nil, fmt.Errorf("locate cache directory: %w", err)
	}
	client := &snapshotClient{
		key:      key,
		cacheDir: filepath.Join(cacheDir, "anyquery", "plugins", "livetennis"),
		endpoint: matchesURL,
		http: &http.Client{
			Timeout: 15 * time.Second,
			// Fresh HTTP/1 connections avoid transparent retries within a reserved attempt.
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				DisableKeepAlives:   true,
				TLSHandshakeTimeout: 10 * time.Second,
				TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
			},
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now,
	}
	return &matchesTable{client: client}, &rpc.DatabaseSchema{
		PrimaryKey: 0,
		Columns: []rpc.DatabaseSchemaColumn{
			{Name: "id", Type: rpc.ColumnTypeInt},
			{Name: "tournament", Type: rpc.ColumnTypeString},
			{Name: "tour", Type: rpc.ColumnTypeString},
			{Name: "scheduled_at", Type: rpc.ColumnTypeDateTime},
			{Name: "status", Type: rpc.ColumnTypeString},
			{Name: "players", Type: rpc.ColumnTypeJSON},
			{Name: "score", Type: rpc.ColumnTypeJSON},
			{Name: "fetched_at", Type: rpc.ColumnTypeDateTime},
		},
	}, nil
}

type matchesTable struct {
	client *snapshotClient
}

func (t *matchesTable) CreateReader() rpc.ReaderInterface {
	return &matchesCursor{client: t.client}
}

func (t *matchesTable) Close() error { return nil }

type matchesCursor struct {
	client *snapshotClient
}

func (c *matchesCursor) Query(_ rpc.QueryConstraint) ([][]interface{}, bool, error) {
	snapshot, err := c.client.load()
	if err != nil {
		return nil, true, err
	}
	rows := make([][]interface{}, 0, len(snapshot.Data))
	for _, match := range snapshot.Data {
		rows = append(rows, []interface{}{
			match.ID, match.Tournament, nullableString(match.Tour),
			nullableString(match.ScheduledTime), match.Status,
			nullableJSON(match.Players), nullableJSON(match.Score),
			snapshot.FetchedAt.UTC().Format(time.RFC3339),
		})
	}
	return rows, true, nil
}

type match struct {
	ID            int64           `json:"id"`
	Tournament    string          `json:"tournament"`
	Tour          *string         `json:"tour"`
	ScheduledTime *string         `json:"scheduled_time"`
	Status        string          `json:"status"`
	Players       json.RawMessage `json:"players"`
	Score         json.RawMessage `json:"score"`
}

func nullableString(value *string) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableJSON(value json.RawMessage) interface{} {
	if len(value) == 0 || strings.TrimSpace(string(value)) == "null" {
		return nil
	}
	return string(value)
}
