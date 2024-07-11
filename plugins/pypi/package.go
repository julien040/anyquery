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
			},
			{
				Name: "url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "author",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "author_email",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "description",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "home_page",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "keywords",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "license",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "maintainer",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "maintainer_email",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "documentation_url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "source_code_url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "current_version",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "version_count",
				Type: rpc.ColumnTypeInt,
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

// A slice of rows to insert
func (t *packageTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *packageTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *packageTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *packageTable) Close() error {
	return nil
}
