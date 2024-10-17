package main

import (
	"fmt"

	"github.com/julien040/anyquery/plugins/salesforce/api"
	"github.com/julien040/anyquery/rpc"
)

func salesforce_customCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get the sObject name from the arguments
	sObject := args.UserConfig.GetString("sObject")
	if sObject == "" {
		return nil, nil, fmt.Errorf("sObject must be set in the table configuration")
	}

	return api.TableFactory(sObject)(args)

}
