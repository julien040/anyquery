package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

var retryableClient = retryablehttp.Client{
	HTTPClient: http.DefaultClient,
	Backoff:    retryablehttp.DefaultBackoff,
	RetryMax:   12,
	CheckRetry: retryablehttp.DefaultRetryPolicy,
	Logger:     log.Default(),
}

var restyClient = resty.NewWithClient(retryableClient.StandardClient())

func main() {
	plugin := rpc.NewPlugin(albumTableCreator, trackCreator, playlistCreator, searchCreator, historyCreator, savedTrackCreator)
	plugin.Serve()
}

func getAccessToken(refreshToken string, client_id, client_secret string) (string, error) {
	if refreshToken == "" {
		return "", fmt.Errorf("refresh token is empty")
	}
	if client_id == "" {
		return "", fmt.Errorf("client_id is empty")
	}
	if client_secret == "" {
		return "", fmt.Errorf("client_secret is empty")
	}
	content := url.Values{}
	content.Set("grant_type", "refresh_token")
	content.Set("refresh_token", refreshToken)
	urlReq := "https://accounts.spotify.com/api/token"

	token := base64.StdEncoding.EncodeToString([]byte(client_id + ":" + client_secret))

	var data map[string]interface{}
	res, err := restyClient.R().SetHeaderMultiValues(map[string][]string{
		"Content-Type":  {"application/x-www-form-urlencoded"},
		"Authorization": {"Basic " + token},
	}).SetBody(content.Encode()).SetResult(&data).Post(urlReq) // &data and not data
	if err != nil {
		return "", err
	}

	if res.StatusCode() != 200 {
		log.Printf("Failed to get access token: %d\n", res.StatusCode())
		return "", fmt.Errorf("failed to get access token: %s(status code: %d)", res.String(), res.StatusCode())
	}

	accessToken, ok := data["access_token"]
	if !ok {
		log.Printf("Failed to find access token in body: %+v\n", data)
		return "", fmt.Errorf("failed to find access token in body: %s", res.String())
	}

	return accessToken.(string), nil

}

// Open a badger database in the cache folder in the specified type
func openDB(Type, refresh_token string) (*badger.DB, error) {
	// Hash the token to get a key
	// This key will be used to encrypt the database
	hash := sha256.Sum256([]byte(refresh_token))
	hexHash := fmt.Sprintf("%x", hash)

	cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "spotify", Type, hexHash)
	err := os.MkdirAll(cacheFolder, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache folder: %w", err)
	}

	// Open the badger database encrypted with the toke
	options := badger.DefaultOptions(cacheFolder).WithNumVersionsToKeep(1).WithEncryptionKey(hash[:]).
		WithCompactL0OnClose(true).WithValueLogFileSize(2 << 21).WithIndexCacheSize(2 << 21)
	db, err := badger.Open(options)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger database: %w", err)
	}
	return db, nil
}
