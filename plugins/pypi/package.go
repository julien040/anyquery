package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func packageCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &packageTable{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "package_name",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
				Description: "The name of the Pypi (pip) package you want to get information about",
			},
			{
				Name:        "url",
				Type:        rpc.ColumnTypeString,
				Description: "The URL of the package to see more information",
			},
			{
				Name:        "author",
				Type:        rpc.ColumnTypeString,
				Description: "The author of the package",
			},
			{
				Name:        "author_email",
				Type:        rpc.ColumnTypeString,
				Description: "The email of the author",
			},
			{
				Name:        "description",
				Type:        rpc.ColumnTypeString,
				Description: "The description of the package",
			},
			{
				Name:        "home_page",
				Type:        rpc.ColumnTypeString,
				Description: "A link to the homepage of the package",
			},
			{
				Name:        "keywords",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON array of keywords",
			},
			{
				Name:        "license",
				Type:        rpc.ColumnTypeString,
				Description: "The license of the package",
			},
			{
				Name:        "maintainer",
				Type:        rpc.ColumnTypeString,
				Description: "The maintainer of the package",
			},
			{
				Name:        "maintainer_email",
				Type:        rpc.ColumnTypeString,
				Description: "The email of the maintainer",
			},
			{
				Name:        "documentation_url",
				Type:        rpc.ColumnTypeString,
				Description: "A link to the documentation of the package",
			},
			{
				Name:        "source_code_url",
				Type:        rpc.ColumnTypeString,
				Description: "A link to the source code of the package",
			},
			{
				Name:        "current_version",
				Type:        rpc.ColumnTypeString,
				Description: "The current version of the package that will be downloaded using pip",
			},
			{
				Name:        "version_count",
				Type:        rpc.ColumnTypeInt,
				Description: "The number of versions available",
			},
		},
	}, nil
}

type packageTable struct {
}

type packageCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *packageCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the package name from the constraints
	packageName := ""
	for _, c := range constraints.Columns {
		if c.ColumnID == 0 {
			if parsedStr, ok := c.Value.(string); ok {
				packageName = parsedStr
			}
		}
	}
	if packageName == "" {
		return nil, true, fmt.Errorf("package name must be a non-empty string")
	}

	// Get the package information
	data := APIresponse{}
	res, err := client.R().SetResult(&data).Get("https://pypi.org/pypi/" + packageName + "/json")
	if err != nil {
		return nil, true, fmt.Errorf("error fetching data: %v", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("error fetching data(code %d): %s", res.StatusCode(), res.String())
	}

	// Prepare the rows
	rows := [][]interface{}{}

	rows = append(rows, []interface{}{
		data.Info.PackageURL,
		data.Info.Author,
		data.Info.AuthorEmail,
		data.Info.Description,
		data.Info.HomePage,
		data.Info.Keywords,
		data.Info.License,
		data.Info.Maintainer,
		data.Info.MaintainerEmail,
		data.Info.ProjectUrls.Documentation,
		data.Info.ProjectUrls.Source,
		data.Info.Version,
		len(data.Releases),
	})

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *packageTable) CreateReader() rpc.ReaderInterface {
	return &packageCursor{}
}

// A destructor to clean up resources
func (t *packageTable) Close() error {
	return nil
}
