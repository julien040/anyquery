package main

import (
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

var retry = retryablehttp.NewClient()
var client = resty.NewWithClient(retry.HTTPClient)

func main() {
	retry.RetryMax = 5
	plugin := rpc.NewPlugin(boardsCreator, cardsCreator, listsCreator)
	plugin.Serve()
}
