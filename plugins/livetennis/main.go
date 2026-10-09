package main

import "github.com/julien040/anyquery/rpc"

func main() {
	rpc.NewPlugin(matchesCreator).Serve()
}
