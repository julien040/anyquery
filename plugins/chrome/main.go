package main

import (
	"github.com/julien040/anyquery/rpc"
)

func main() {
	plugin := rpc.NewPlugin(tabsCreator, historyCreator, downloadsCreator)
	plugin.Serve()
}
