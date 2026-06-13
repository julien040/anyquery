package main

import (
	"github.com/julien040/anyquery/rpc"
)

// pluginVersion must match the version field in manifest.toml. It is stamped
// into cache keys so a new release never reads a previous version's cached
// rows (the v0.1.6 helper cache does not version entries itself).
const pluginVersion = "0.1.0"

func main() {
	// The order must match the tables field of the manifest
	plugin := rpc.NewPlugin(modelsCreator, llmCreator, imageCreator, videoJobsCreator, predictionsCreator)
	plugin.Serve()
}
