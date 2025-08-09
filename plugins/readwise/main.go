package main

import (
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

var retry = retryablehttp.NewClient()
var client = resty.NewWithClient(retry.StandardClient())

func main() {
	retry.Backoff = retryablehttp.DefaultBackoff
	retry.RetryMax = 8
	plugin := rpc.NewPlugin(highlightsCreator, documentsCreator)
	plugin.Serve()
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
