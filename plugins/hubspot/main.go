package main

import (
	"github.com/julien040/anyquery/rpc"
)

var supportedObjectsList = []string{
	"companies", "contacts", "deals", "feedback_submissions", "goal_targets", "leads", "tickets", "carts", "discounts", "fees",
	"invoices", "line_items", "orders", "commerce_payments", "products", "quotes", "subscriptions", "taxes", "calls", "communications",
	"emails", "meetings", "notes", "postal_mail", "tasks"}

var supportedObjectsIsReadOnly = []bool{
	false, false, false, true, true, false, false, false, false, false, true, false, false, true, false, false, true, false,
	false, false, false, false, false, false, false}

func main() {
	// Create a new plugin
	plugin := rpc.NewPlugin()

	// Register all supported objects
	for i, object := range supportedObjectsList {
		plugin.RegisterTable(i, factory(object, supportedObjectsIsReadOnly[i]))
	}

	// Start the plugin
	plugin.Serve()
}
