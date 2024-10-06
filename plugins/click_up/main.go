package main

import (
	"log"
	"strconv"
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
	plugin := rpc.NewPlugin(tasksCreator, docsCreator, docs_pagesCreator,
		foldersCreator, listsCreator, whoamiCreator)
	plugin.Serve()
}

func convertTime(val interface{}) interface{} {
	log.Printf("val: %v %T", val, val)
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case float64:
		// Parse the unix timestamp as a float
		return time.Unix(int64(v)/1000, 0).Format(time.RFC3339)
	case int64:
		// Parse the unix timestamp as a int
		return time.Unix(v/1000, 0).Format(time.RFC3339)
	case string:
		// Parse the unix timestamp as a int string
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil
		}
		return time.Unix(parsed/1000, 0).Format(time.RFC3339)
	case *string:
		if v == nil {
			return nil
		}
		return convertTime(*v)
	default:
		return nil
	}
}
