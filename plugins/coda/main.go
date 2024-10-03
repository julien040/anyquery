package main

import (
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

var retry = retryablehttp.NewClient()
var client = resty.NewWithClient(retry.StandardClient())

func main() {
	retry.RetryMax = 5
	retry.RetryWaitMin = time.Second
	plugin := rpc.NewPlugin(tableCreator)
	plugin.Serve()
}
