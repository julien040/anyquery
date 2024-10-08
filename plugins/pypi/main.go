package main

import (
	"github.com/go-resty/resty/v2"
	"github.com/julien040/anyquery/rpc"
)

var client = resty.New()

func main() {
	plugin := rpc.NewPlugin(pypiVersionCreator, packageCreator)
	plugin.Serve()
}
