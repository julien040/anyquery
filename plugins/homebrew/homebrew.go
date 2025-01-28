package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

type BrewFormula struct {
	Name              string   `json:"name"`
	FullName          string   `json:"full_name"`
	Tap               string   `json:"tap"`
	Oldnames          []string `json:"oldnames"`
	Aliases           []string `json:"aliases"`
	VersionedFormulae []string `json:"versioned_formulae"`
	Desc              string   `json:"desc"`
	License           string   `json:"license"`
	Homepage          string   `json:"homepage"`
	Versions          struct {
		Stable string `json:"stable"`
		Head   any    `json:"head"`
		Bottle bool   `json:"bottle"`
	} `json:"versions"`
	Urls struct {
		Stable struct {
			URL      string `json:"url"`
			Tag      any    `json:"tag"`
			Revision any    `json:"revision"`
			Using    any    `json:"using"`
			Checksum string `json:"checksum"`
		} `json:"stable"`
	} `json:"urls"`
	BuildDependencies       []string `json:"build_dependencies"`
	Dependencies            []string `json:"dependencies"`
	TestDependencies        []string `json:"test_dependencies"`
	RecommendedDependencies []string `json:"recommended_dependencies"`
	OptionalDependencies    []string `json:"optional_dependencies"`
	Revision                int      `json:"revision"`
	VersionScheme           int      `json:"version_scheme"`
	Install30days           int      `json:"install_30_days"`
	Install90days           int      `json:"install_90_days"`
	Install365days          int      `json:"install_365_days"`
}

type BrewAnalyticsFormulae struct {
	Category   string `json:"category"`
	TotalItems int    `json:"total_items"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	TotalCount int    `json:"total_count"`
	Items      []struct {
		Number  int    `json:"number"`
		Formula string `json:"formula"`
		Count   string `json:"count"`
		Percent string `json:"percent"`
	} `json:"items"`
}

type BrewAnalyticsCask struct {
	Category   string `json:"category"`
	TotalItems int    `json:"total_items"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	TotalCount int    `json:"total_count"`
	Formulae   map[string][]struct {
		Cask  string `json:"cask"`
		Count string `json:"count"`
	} `json:"formulae"`
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func homebrewFormulaCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {

	formulae, err := requestFormulae()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to request formulae from API: %w", err)
	}

	return &brewFormulaeTable{
			formulae: formulae,
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the formula. Example: 'git'",
				},
				{
					Name:        "full_name",
					Type:        rpc.ColumnTypeString,
					Description: "The full name of the formula. Example: 'git'",
				},
				{
					Name:        "tap",
					Type:        rpc.ColumnTypeString,
					Description: "The tap of the formula. Example: 'homebrew/core'",
				},
				{
					Name:        "oldnames",
					Type:        rpc.ColumnTypeJSON,
					Description: "An array of old names of the formula. Example: ['git']",
				},
				{
					Name:        "aliases",
					Type:        rpc.ColumnTypeJSON,
					Description: "An array of aliases of the formula. Example: ['git']",
				},
				{
					Name:        "versioned_formulae",
					Type:        rpc.ColumnTypeJSON,
					Description: "An array of versioned formulae of the formula. Example: ['git']",
				},
				{
					Name:        "description",
					Type:        rpc.ColumnTypeString,
					Description: "The description of the formula",
				},
				{
					Name:        "license",
					Type:        rpc.ColumnTypeString,
					Description: "The license of the formula",
				},
				{
					Name: "versions",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "build_dependencies",
					Type:        rpc.ColumnTypeJSON,
					Description: "An array of build dependencies of the formula",
				},
				{
					Name:        "dependencies",
					Type:        rpc.ColumnTypeJSON,
					Description: "An array of dependencies of the formula",
				},
				{
					Name: "test_dependencies",
					Type: rpc.ColumnTypeJSON,
				},
				{
					Name: "recommended_dependencies",
					Type: rpc.ColumnTypeJSON,
				},
				{
					Name: "optional_dependencies",
					Type: rpc.ColumnTypeJSON,
				},
				{
					Name: "revision",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "install_30_days",
					Type:        rpc.ColumnTypeString,
					Description: "The number of installs in the last 30 days",
				},
				{
					Name:        "install_90_days",
					Type:        rpc.ColumnTypeString,
					Description: "The number of installs in the last 90 days",
				},
				{
					Name:        "install_365_days",
					Type:        rpc.ColumnTypeString,
					Description: "The number of installs in the last 365 days",
				},
			},
		}, nil
}

type brewFormulaeTable struct {
	formulae map[string]BrewFormula
}

type brewFormulaeCursor struct {
	formulae map[string]BrewFormula
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *brewFormulaeCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Convert the formulae to a slice of rows
	rows := make([][]interface{}, 0, len(t.formulae))

	for _, formula := range t.formulae {
		rows = append(rows, []interface{}{
			formula.Name,
			formula.FullName,
			formula.Tap,
			formula.Oldnames,
			formula.Aliases,
			formula.VersionedFormulae,
			formula.Desc,
			formula.License,
			formula.Versions.Stable,
			formula.BuildDependencies,
			formula.Dependencies,
			formula.TestDependencies,
			formula.RecommendedDependencies,
			formula.OptionalDependencies,
			formula.Revision,
			formula.Install30days,
			formula.Install90days,
			formula.Install365days,
		})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *brewFormulaeTable) CreateReader() rpc.ReaderInterface {
	return &brewFormulaeCursor{
		t.formulae,
	}
}

// A destructor to clean up resources
func (t *brewFormulaeTable) Close() error {
	return nil
}
