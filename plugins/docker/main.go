package main

import (
	"encoding/json"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/julien040/anyquery/rpc"
)

func main() {
	plugin := rpc.NewPlugin(containersCreator, containerCreator,
		imagesCreator, networksCreator)
	plugin.Serve()
}

// Serialize a value to JSON and return nil in case of an error
func serializeJSON(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	b, err := json.Marshal(val)
	if err != nil {
		return nil
	}

	return string(b)
}

func extractHost(constraints rpc.QueryConstraint, colPosition int) string {
	host := ""
	for _, cst := range constraints.Columns {
		if cst.ColumnID == colPosition {
			host, ok := cst.Value.(string)
			if ok {
				return host
			}
		}
	}

	return host
}

func createClient(constraints rpc.QueryConstraint, colPosition int) (*client.Client, error) {
	host := extractHost(constraints, colPosition)
	options := []client.Opt{client.FromEnv}
	if host != "" {
		options = append(options, client.WithHost(host))
	}
	options = append(options, client.WithAPIVersionNegotiation())
	cli, err := client.NewClientWithOpts(options...)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func retrieveArgString(constraints rpc.QueryConstraint, columnID int) string {
	for _, c := range constraints.Columns {
		if c.ColumnID == columnID {
			switch rawVal := c.Value.(type) {
			case string:
				return rawVal
			case int64:
				return fmt.Sprintf("%d", rawVal)
			case float64:
				return fmt.Sprintf("%f", rawVal)
			}
		}
	}

	return ""

}
