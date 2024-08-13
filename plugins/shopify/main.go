package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
	"golang.org/x/time/rate"
)

var retry = retryablehttp.NewClient()
var client = resty.NewWithClient(retry.StandardClient())

const graphQLEndpoint = "https://{shop}.myshopify.com/admin/api/2024-07/graphql.json"

const tokenPerSecond = 100
const cacheTTL = time.Hour * 1

type rateLimit struct {
	limiters map[string]*rate.Limiter
}

func (r *rateLimit) Init(token string) {
	if r.limiters == nil {
		r.limiters = make(map[string]*rate.Limiter)
	}

	r.limiters[token] = rate.NewLimiter(tokenPerSecond, tokenPerSecond)
}

func (r *rateLimit) Wait(token string, n int) error {
	if limiter, ok := r.limiters[token]; ok {
		return limiter.WaitN(context.Background(), n)
	} else {
		r.Init(token)
		return r.Wait(token, n)
	}
}

func extractKeyUserConf(userConf rpc.PluginConfig, key string) string {
	inter, ok := userConf[key]
	if !ok {
		return ""
	}
	if token, ok := inter.(string); ok {
		return token
	}
	return ""
}

func openCache(element string, token string) (*badger.DB, error) {
	// Find the cache path
	md5sumToken := md5.Sum([]byte(token))
	cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "shopify", fmt.Sprintf("%x", md5sumToken[:]), element)

	// Open the cache
	options := badger.DefaultOptions(cacheFolder).WithEncryptionKey(md5sumToken[:]).
		WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 27).
		WithIndexCacheSize(2 << 24)

	return badger.Open(options)
}

func convertStrToFloat(str string) interface{} {
	if str == "" {
		return nil
	}

	// Convert the string to a float
	float, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return nil
	}
	return float
}

func main() {
	// Here we use a global rate limiter per token that will limit the number of requests per second
	// to 100 query cost per second
	rateLimiter := &rateLimit{}
	retry.RetryMax = 12
	retry.RetryWaitMin = 1 * time.Second
	plugin := rpc.NewPlugin(rateLimiter.ordersCreator, rateLimiter.productsCreator,
		rateLimiter.products_variantCreator, rateLimiter.customersCreator)
	plugin.Serve()
}
