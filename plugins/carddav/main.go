package main

import (
	"github.com/julien040/anyquery/rpc"
)

func main() {
	plugin := rpc.NewPlugin(contactsCreator, addressBooksCreator)
	plugin.Serve()
}
