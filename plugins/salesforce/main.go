package main

import (
	"github.com/julien040/anyquery/plugins/salesforce/api"
	"github.com/julien040/anyquery/rpc"
)

var supportedSObjects = []string{
	"account",
	"contact",
	"lead",
	"opportunity",
	"case",
	"task",
	"event",
	"campaign",
	"user",
	"campaignmember",
	"asset",
	"contract",
	"contractlineitem",
	"servicecontract",
	"solution",
	"pricebook2",
	"product2",
	"productitem",
	"pricebookentry",
	"quote",
	"quotelineitem",
	"order",
	"orderitem",
	"invoice",
	"invoiceline",
	"report",
	"dashboard",
	"document",
}

func main() {
	plugin := rpc.NewPlugin()
	// Create a new table for each supported SObject
	for i, sObject := range supportedSObjects {
		plugin.RegisterTable(i, api.TableFactory(sObject))
	}
	plugin.Serve()
}
