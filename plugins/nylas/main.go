/*
Copyright 2024 Julien CAGNIART

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package main

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

var retry = retryablehttp.NewClient()
var client = resty.NewWithClient(retry.StandardClient())

// Get the apiKey, GrantID and ServerHost from the connection arguments
func getCredentials(args rpc.TableCreatorArgs) (string, string, string, error) {
	var apiKey, grantID, serverHost string
	rawInter := interface{}(nil)
	var ok bool
	if rawInter, ok = args.UserConfig["api_key"]; !ok {
		return "", "", "", fmt.Errorf("api_key not found in user config")
	} else if apiKey, ok = rawInter.(string); !ok {
		return "", "", "", fmt.Errorf("api_key is not a string")
	}

	if rawInter, ok = args.UserConfig["grant_id"]; !ok {
		return "", "", "", fmt.Errorf("grant_id not found in user config")
	}
	if grantID, ok = rawInter.(string); !ok {
		return "", "", "", fmt.Errorf("grant_id is not a string")
	}

	if rawInter, ok = args.UserConfig["server_host"]; !ok {
		return "", "", "", fmt.Errorf("server_host not found in user config")
	} else if serverHost, ok = rawInter.(string); !ok {
		return "", "", "", fmt.Errorf("server_host is not a string")
	}

	return apiKey, grantID, serverHost, nil
}

func parseDate(row []interface{}, index int) int64 {
	if index >= len(row) {
		return 0
	}
	switch v := row[index].(type) {
	case int64:
		return v
	case string:
		// Try to parse as RFC3339. If not working, parse the dateonly format
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.Unix()
		} else if t, err := time.Parse(time.DateTime, v); err == nil {
			return t.Unix()
		} else if t, err := time.Parse(time.DateOnly, v); err == nil {
			return t.Unix()
		}
	}

	return 0
}

func getString(row []interface{}, index int) string {
	if index >= len(row) {
		return ""
	}
	if v, ok := row[index].(string); ok {
		return v
	}
	return ""
}

func getInt(row []interface{}, index int) int64 {
	if index >= len(row) {
		return 0
	}
	if v, ok := row[index].(int64); ok {
		return v
	}
	return 0
}

func main() {
	retry.RetryMax = 12
	plugin := rpc.NewPlugin(eventsCreator, emailsCreator)
	plugin.Serve()
}
