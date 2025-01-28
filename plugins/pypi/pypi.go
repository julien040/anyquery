package main

import (
	"fmt"
	"time"

	"github.com/julien040/anyquery/rpc"
)

type APIresponse struct {
	Info struct {
		Author                 string   `json:"author"`
		AuthorEmail            string   `json:"author_email"`
		Classifiers            []string `json:"classifiers"`
		Description            string   `json:"description"`
		DescriptionContentType string   `json:"description_content_type"`
		Downloads              struct {
			LastDay   int `json:"last_day"`
			LastMonth int `json:"last_month"`
			LastWeek  int `json:"last_week"`
		} `json:"downloads"`
		Dynamic         any      `json:"dynamic"`
		HomePage        string   `json:"home_page"`
		Keywords        []string `json:"keywords"`
		License         string   `json:"license"`
		Maintainer      string   `json:"maintainer"`
		MaintainerEmail string   `json:"maintainer_email"`
		Name            string   `json:"name"`
		PackageURL      string   `json:"package_url"`
		Platform        any      `json:"platform"`
		ProjectURL      string   `json:"project_url"`
		ProjectUrls     struct {
			Documentation string `json:"Documentation"`
			Homepage      string `json:"Homepage"`
			Source        string `json:"Source"`
		} `json:"project_urls"`
		ProvidesExtra  []string `json:"provides_extra"`
		ReleaseURL     string   `json:"release_url"`
		RequiresDist   []string `json:"requires_dist"`
		RequiresPython string   `json:"requires_python"`
		Summary        string   `json:"summary"`
		Version        string   `json:"version"`
		Yanked         bool     `json:"yanked"`
		YankedReason   any      `json:"yanked_reason"`
	} `json:"info"`
	LastSerial int `json:"last_serial"`
	Releases   map[string][]struct {
		CommentText string `json:"comment_text"`
		Digests     struct {
			Blake2B256 string `json:"blake2b_256"`
			Md5        string `json:"md5"`
			Sha256     string `json:"sha256"`
		} `json:"digests"`
		Downloads         int       `json:"downloads"`
		Filename          string    `json:"filename"`
		HasSig            bool      `json:"has_sig"`
		Md5Digest         string    `json:"md5_digest"`
		Packagetype       string    `json:"packagetype"`
		PythonVersion     string    `json:"python_version"`
		RequiresPython    any       `json:"requires_python"`
		Size              int       `json:"size"`
		UploadTime        string    `json:"upload_time"`
		UploadTimeIso8601 time.Time `json:"upload_time_iso_8601"`
		URL               string    `json:"url"`
		Yanked            bool      `json:"yanked"`
		YankedReason      any       `json:"yanked_reason"`
	} `json:"releases"`
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func pypiVersionCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &pypiVersionTable{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		PrimaryKey:    -1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "package_name",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
				Description: "The name of the Pypi (pip) package you want to list versions of",
			},
			{
				Name:        "package_url",
				Type:        rpc.ColumnTypeString,
				Description: "The URL of the package to see more information",
			},
			{
				Name:        "package_author",
				Type:        rpc.ColumnTypeString,
				Description: "The author of the package",
			},
			{
				Name:        "version",
				Type:        rpc.ColumnTypeString,
				Description: "One of the versions of the package",
			},
			{
				Name: "md5_digest",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "upload_time",
				Type: rpc.ColumnTypeDateTime,
			},
			{
				Name: "filename",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "version_size",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "python_version",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "yanked",
				Type: rpc.ColumnTypeInt,
			},
		},
	}, nil
}

type pypiVersionTable struct {
}

type pypiVersionCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *pypiVersionCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Find the package name from the constraints
	packageName := ""
	for _, c := range constraints.Columns {
		if c.ColumnID == 0 {
			if parsedString, ok := c.Value.(string); ok {
				packageName = parsedString
			}
		}
	}

	if packageName == "" {
		return nil, true, fmt.Errorf("package_name is required and must be a string")
	}

	// Fetch the data from the API
	var response APIresponse
	endpoint := "https://pypi.org/pypi/" + packageName + "/json"

	res, err := client.R().SetResult(&response).Get(endpoint)
	if err != nil {
		return nil, true, fmt.Errorf("error fetching data from the API: %v", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("error fetching data from the API(code %d): %s", res.StatusCode(), res.String())
	}

	// Prepare the rows
	rows := make([][]interface{}, 0)

	for version, releases := range response.Releases {
		for _, release := range releases {
			rows = append(rows, []interface{}{
				response.Info.PackageURL,
				response.Info.Author,
				version,
				release.Md5Digest,
				release.UploadTime,
				release.Filename,
				release.Size,
				release.PythonVersion,
				release.URL,
				release.Yanked,
			})
		}
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *pypiVersionTable) CreateReader() rpc.ReaderInterface {
	return &pypiVersionCursor{}
}

// A destructor to clean up resources
func (t *pypiVersionTable) Close() error {
	return nil
}
