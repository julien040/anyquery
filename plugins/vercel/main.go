package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

const EntriesPerPage = 100
const ttl = time.Hour * 1

var retryClient = retryablehttp.NewClient()
var client = resty.NewWithClient(retryClient.StandardClient())

func main() {
	retryClient.RetryMax = 5
	plugin := rpc.NewPlugin(projectsCreator, deploymentsCreator)
	plugin.Serve()
}

// Open a database in the cache path encrypted with a hash of the token
func openDatabase(tag, token string) (*badger.DB, error) {
	// Hash the token
	md5sum := md5.Sum([]byte(token))
	hash := fmt.Sprintf("%x", md5sum)

	cachePath := path.Join(xdg.CacheHome, "anyquery", "plugins", "vercel", tag, hash)

	// Make the directory
	os.MkdirAll(cachePath, 0700)

	// Open the database
	options := badger.DefaultOptions(cachePath).WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 21).
		WithIndexCacheSize(2 << 23).WithEncryptionKey(md5sum[:])

	db, err := badger.Open(options)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nil
}

func retrieveArgString(constraints rpc.QueryConstraint, columnID int) string {
	for _, c := range constraints.Columns {
		if c.ColumnID == columnID {
			switch rawVal := c.Value.(type) {
			case string:
				return rawVal
			case int64:
				return fmt.Sprintf("%d", rawVal)
			case float64:
				return fmt.Sprintf("%f", rawVal)
			}
		}
	}

	return ""

}

func getToken(conf rpc.PluginConfig) (string, error) {
	inter, ok := conf["token"]
	if !ok {
		return "", fmt.Errorf("missing token in configuration")
	}

	token, ok := inter.(string)
	if !ok {
		return "", fmt.Errorf("token is not a string")
	}
	return token, nil
}
