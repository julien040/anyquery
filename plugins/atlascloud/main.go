package main

import (
	"github.com/julien040/anyquery/rpc"
)

func main() {
	// The order must match the tables field of the manifest
	plugin := rpc.NewPlugin(modelsCreator, llmCreator, imageCreator, videoJobsCreator)
	plugin.Serve()
}
