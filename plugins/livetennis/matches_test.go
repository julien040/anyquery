package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/julien040/anyquery/rpc"
)

func TestMatchesCreatorRejectsMissingKey(t *testing.T) {
	for _, value := range []interface{}{nil, "", " \t\n", 12, false} {
		if table, schema, err := matchesCreator(rpc.TableCreatorArgs{UserConfig: rpc.PluginConfig{"api_key": value}}); err == nil || table != nil || schema != nil {
			t.Fatalf("api_key %v returned table=%v, schema=%v, error=%v", value, table, schema, err)
		}
	}
}

func TestMatchesCreatorReadOnlySchema(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	table, schema, err := matchesCreator(rpc.TableCreatorArgs{UserConfig: rpc.PluginConfig{"api_key": testKey}})
	if err != nil {
		t.Fatal(err)
	}
	if schema.HandlesInsert || schema.HandlesUpdate || schema.HandlesDelete || schema.HandleOffset {
		t.Fatal("schema must be read only and leave offset handling to anyquery")
	}
	if _, ok := table.(rpc.TableInsert); ok {
		t.Fatal("table implements insert")
	}
	if _, ok := table.(rpc.TableUpdate); ok {
		t.Fatal("table implements update")
	}
	if _, ok := table.(rpc.TableDelete); ok {
		t.Fatal("table implements delete")
	}
	wantNames := []string{"id", "tournament", "tour", "scheduled_at", "status", "players", "score", "fetched_at"}
	wantTypes := []rpc.ColumnType{rpc.ColumnTypeInt, rpc.ColumnTypeString, rpc.ColumnTypeString, rpc.ColumnTypeDateTime, rpc.ColumnTypeString, rpc.ColumnTypeJSON, rpc.ColumnTypeJSON, rpc.ColumnTypeDateTime}
	if len(schema.Columns) != len(wantNames) || schema.PrimaryKey != 0 {
		t.Fatalf("unexpected schema: %+v", schema)
	}
	for i, column := range schema.Columns {
		if column.Name != wantNames[i] || column.Type != wantTypes[i] {
			t.Errorf("column %d: %+v, want %s with type %v", i, column, wantNames[i], wantTypes[i])
		}
	}
	endpoint, err := url.Parse(table.(*matchesTable).client.endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Scheme != "https" || endpoint.Host != "api.livetennisapi.com" || endpoint.Path != "/api/public/v1/matches" {
		t.Fatalf("unexpected production endpoint: %s", endpoint)
	}
	if want := (url.Values{"status": {"live"}, "limit": {"200"}, "offset": {"0"}}); !reflect.DeepEqual(endpoint.Query(), want) {
		t.Fatalf("unexpected endpoint parameters: %v", endpoint.Query())
	}
	transport, ok := table.(*matchesTable).client.http.Transport.(*http.Transport)
	if !ok || !transport.DisableKeepAlives || transport.TLSNextProto == nil || len(transport.TLSNextProto) != 0 || transport.ForceAttemptHTTP2 {
		t.Fatal("transport must use fresh HTTP/1 connections without transparent retries")
	}
	if err := table.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMatchesCreatorNormalizesKey(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("X-API-Key") != testKey {
			t.Error("request contains an unnormalized key")
		}
		io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()
	for _, key := range []string{" \t" + testKey + "\n ", testKey} {
		table, _, err := matchesCreator(rpc.TableCreatorArgs{UserConfig: rpc.PluginConfig{"api_key": key}})
		if err != nil {
			t.Fatal(err)
		}
		client := table.(*matchesTable).client
		if client.key != testKey {
			t.Fatal("profile key was not normalized")
		}
		client.endpoint, client.now = server.URL, func() time.Time { return testNow }
		if _, _, err := table.CreateReader().Query(rpc.QueryConstraint{}); err != nil {
			t.Fatal(err)
		}
	}
	if requests.Load() != 1 {
		t.Fatal("whitespace variants did not share the quota cache")
	}
}

func TestMatchesCursorPreservesJSON(t *testing.T) {
	const players = `{"p1":{"id":11,"name":"First"},"p2":{"id":12,"name":"Second"}}`
	const score = `{"sets":[1,0],"games":[[6,3],[4,4]],"points":[null,"40"],"server":null}`
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		io.WriteString(w, `{"data":[{"id":31,"tournament":"Test Open","tour":"atp","scheduled_time":"2026-01-01T12:00:00Z","status":"live","players":`+players+`,"score":`+score+`},{"id":32,"tournament":"Test Open","tour":null,"scheduled_time":null,"status":"live","players":null,"score":null}]}`)
	}))
	defer server.Close()
	table := &matchesTable{client: newTestClient(t.TempDir(), server.URL, testKey, func() time.Time { return testNow })}
	for i := 0; i < 2; i++ {
		rows, done, err := table.CreateReader().Query(rpc.QueryConstraint{Limit: 1, Offset: 1})
		if err != nil {
			t.Fatal(err)
		}
		if !done || len(rows) != 2 || len(rows[0]) != 8 || len(rows[1]) != 8 {
			t.Fatalf("rows=%v, exhausted=%v", rows, done)
		}
		want := [][]interface{}{
			{int64(31), "Test Open", "atp", "2026-01-01T12:00:00Z", "live", players, score, testNow.Format(time.RFC3339)},
			{int64(32), "Test Open", nil, nil, "live", nil, nil, testNow.Format(time.RFC3339)},
		}
		if !reflect.DeepEqual(rows, want) {
			t.Fatalf("cursor rows = %#v, want %#v", rows, want)
		}
		var parsed struct {
			Games  [][]int       `json:"games"`
			Points []interface{} `json:"points"`
			Server interface{}   `json:"server"`
		}
		if err := json.Unmarshal([]byte(rows[0][6].(string)), &parsed); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(parsed.Games, [][]int{{6, 3}, {4, 4}}) || parsed.Points[0] != nil || parsed.Server != nil {
			t.Fatalf("score changed orientation or null fields: %+v", parsed)
		}
	}
	if requests.Load() != 1 {
		t.Fatal("separate cursors caused duplicate HTTP requests")
	}
}

func TestMatchesCursorFailureReturnsNoRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	cursor := &matchesCursor{client: newTestClient(t.TempDir(), server.URL, testKey, func() time.Time { return testNow })}
	rows, done, err := cursor.Query(rpc.QueryConstraint{})
	if rows != nil || !done || err == nil {
		t.Fatalf("failed cursor returned rows=%v, exhausted=%v, error=%v", rows, done, err)
	}
}

func TestMatchesCreatorDoesNotFollowRedirects(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var leaked atomic.Bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked.Store(true)
		io.WriteString(w, `{"data":[]}`)
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusFound)
	}))
	defer source.Close()
	table, _, err := matchesCreator(rpc.TableCreatorArgs{UserConfig: rpc.PluginConfig{"api_key": testKey}})
	if err != nil {
		t.Fatal(err)
	}
	table.(*matchesTable).client.endpoint = source.URL
	_, _, err = table.CreateReader().Query(rpc.QueryConstraint{})
	if err == nil || leaked.Load() {
		t.Fatalf("redirect error=%v, destination contacted=%v", err, leaked.Load())
	}
	if strings.Contains(err.Error(), testKey) {
		t.Fatal("redirect error contains API key")
	}
}

func TestDecodeMatchesRejectsMalformedResponse(t *testing.T) {
	for _, content := range []string{
		``, `{"data":[`, `{"data":[]} {}`, `{}`, `{"data":null}`, `{"data":{}}`, `{"data":[{"id":"bad"}]}`,
	} {
		if _, err := decodeMatches(strings.NewReader(content)); err == nil {
			t.Errorf("accepted malformed response %q", content)
		}
	}
	if _, err := decodeMatches(bytes.NewReader(bytes.Repeat([]byte{' '}, maxResponseSize+1))); err == nil {
		t.Fatal("accepted oversized response")
	}
	if data, err := decodeMatches(strings.NewReader(`{"data":[]}`)); err != nil || len(data) != 0 {
		t.Fatalf("empty data = %v, %v", data, err)
	}
}
