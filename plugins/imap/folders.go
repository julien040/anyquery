package main

import (
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func foldersCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	config, err := getArgs(args.UserConfig)
	if err != nil {
		return nil, nil, err
	}

	dialer, err := client.DialTLS(fmt.Sprintf("%s:%d", config.Host, config.Port), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to imap server: %v", err)
	}

	err = dialer.Login(config.Username, config.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to login to imap server: %v", err)
	}

	return &foldersTable{
			dialer: dialer,
		}, &rpc.DatabaseSchema{
			PrimaryKey: -1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "folder",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the folder in the mailbox",
				},
			},
		}, nil
}

type foldersTable struct {
	dialer *client.Client
}

type foldersCursor struct {
	dialer *client.Client
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *foldersCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	folders := make([]string, 0)
	// Get the list of folders
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- t.dialer.List("", "*", mailboxes)
	}()

	for m := range mailboxes {
		if m == nil {
			continue
		}
		folders = append(folders, m.Name)
	}
	err := <-done
	if err != nil {
		return nil, true, fmt.Errorf("failed to get folders: %v", err)
	}

	// Convert the list of folders to a slice of rows
	rows := make([][]interface{}, 0, len(folders))
	for _, folder := range folders {
		rows = append(rows, []interface{}{folder})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *foldersTable) CreateReader() rpc.ReaderInterface {
	return &foldersCursor{
		dialer: t.dialer,
	}
}

// A destructor to clean up resources
func (t *foldersTable) Close() error {
	return nil
}

type userConfig struct {
	Username string
	Password string
	Port     int64
	Host     string
}

func getArgs(args rpc.PluginConfig) (userConfig, error) {
	var config userConfig
	var ok bool
	var rawString string
	var rawInt64 int64

	if rawString, ok = args["username"].(string); !ok {
		return config, fmt.Errorf("username is not a string. Got %T", args["username"])
	} else if config.Username = rawString; config.Username == "" {
		return config, fmt.Errorf("username is empty")
	}

	if rawString, ok = args["password"].(string); !ok {
		return config, fmt.Errorf("password is not a string. Got %T", args["password"])
	} else if config.Password = rawString; config.Password == "" {
		return config, fmt.Errorf("password is empty")
	}

	if rawString, ok = args["host"].(string); !ok {
		return config, fmt.Errorf("host is not a string. Got %T", args["host"])
	} else if config.Host = rawString; config.Host == "" {
		return config, fmt.Errorf("host is empty")
	}

	if rawInt64, ok = args["port"].(int64); !ok {
		// Try to convert it from a float
		if rawFloat, ok := args["port"].(float64); !ok {
			return config, fmt.Errorf("port is not a number. Got %T", args["port"])
		} else {
			config.Port = int64(rawFloat)
		}
	} else if config.Port = rawInt64; config.Port == 0 {
		return config, fmt.Errorf("port is 0")
	}

	return config, nil
}
