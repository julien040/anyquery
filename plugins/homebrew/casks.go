package main

import "github.com/julien040/anyquery/rpc"

type BrewCasks struct {
	Token     string   `json:"token"`
	FullToken string   `json:"full_token"`
	OldTokens []string `json:"old_tokens"`
	Tap       string   `json:"tap"`
	Name      []string `json:"name"`
	Desc      string   `json:"desc"`
	Homepage  string   `json:"homepage"`
	URL       string   `json:"url"`
	URLSpecs  struct {
		Verified string `json:"verified"`
	} `json:"url_specs"`
	Version string `json:"version"`

	Outdated       bool   `json:"outdated"`
	Sha256         string `json:"sha256"`
	Deprecated     bool   `json:"deprecated"`
	Disabled       bool   `json:"disabled"`
	Install30days  int    `json:"install_30_days"`
	Install90days  int    `json:"install_90_days"`
	Install365days int    `json:"install_365_days"`
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func homebrewCasksCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	casks, err := requestCasks()
	if err != nil {
		return nil, nil, err
	}
	return &casksTable{
			casks: casks,
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "token",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the cask. Used to install the cask with 'brew install --cask <token>'",
				},
				{
					Name:        "full_token",
					Type:        rpc.ColumnTypeString,
					Description: "The full name of the cask",
				},
				{
					Name:        "old_tokens",
					Type:        rpc.ColumnTypeJSON,
					Description: "An array of old names of the cask",
				},
				{
					Name:        "tap",
					Type:        rpc.ColumnTypeString,
					Description: "The tap of the cask. Example: 'homebrew/cask'",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the cask",
				},
				{
					Name:        "desc",
					Type:        rpc.ColumnTypeString,
					Description: "The description of the cask",
				},
				{
					Name:        "homepage",
					Type:        rpc.ColumnTypeString,
					Description: "The homepage of the cask",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL of the cask",
				},
				{
					Name: "version",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "sha256",
					Type:        rpc.ColumnTypeString,
					Description: "The SHA256 of the cask",
				},
				{
					Name:        "install_30_days",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of installations in the last 30 days",
				},
				{
					Name:        "install_90_days",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of installations in the last 90 days",
				},
				{
					Name:        "install_365_days",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of installations in the last 365 days",
				},
			},
		}, nil
}

type casksTable struct {
	casks map[string]BrewCasks
}

type casksCursor struct {
	casks map[string]BrewCasks
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *casksCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	rows := make([][]interface{}, 0)

	for _, cask := range t.casks {
		row := []interface{}{
			cask.Token,
			cask.FullToken,
			cask.OldTokens,
			cask.Tap,
			cask.Name,
			cask.Desc,
			cask.Homepage,
			cask.URL,
			cask.Version,
			cask.Sha256,
			cask.Install30days,
			cask.Install90days,
			cask.Install365days,
		}
		rows = append(rows, row)
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *casksTable) CreateReader() rpc.ReaderInterface {
	return &casksCursor{
		casks: t.casks,
	}
}

// A destructor to clean up resources
func (t *casksTable) Close() error {
	return nil
}
