package main

import (
	"sync"

	"github.com/julien040/anyquery/rpc"
	"go.uber.org/ratelimit"
)

// We need tableCreator to be part of a bigger struct
// so that we can have a global rate limiter for all connections
//
// This is due to Airtable's rate limiting policy of 5 requests per second
// for each base (and not per token)
type tablePlugin struct {
	mapMutex    sync.Mutex
	rateLimiter map[string]ratelimit.Limiter
}

func main() {
	tablePluginInstance := &tablePlugin{
		rateLimiter: make(map[string]ratelimit.Limiter),
		mapMutex:    sync.Mutex{},
	}
	plugin := rpc.NewPlugin(tablePluginInstance.tableCreator)
	plugin.Serve()
}
