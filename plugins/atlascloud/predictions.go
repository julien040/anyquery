package main

import (
	"encoding/json"
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// predictionsCreator exposes a read-only lookup of any Atlas Cloud prediction
// by its ID. It lets users recover an image generation that timed out (the
// generation was already paid for and may finish later) or check the state of
// any prediction without submitting a new — billable — generation.
func predictionsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}
	baseURL := getBaseURL(args.UserConfig.GetString("base_url"))

	return &predictionsTable{
			client: newAtlasClient(apiKey, baseURL),
		}, &rpc.DatabaseSchema{
			PrimaryKey: -1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "prediction_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The Atlas Cloud prediction ID to look up (from atlascloud_image, an image timeout message, or atlascloud_video_jobs)",
				},
				{
					Name:        "status",
					Type:        rpc.ColumnTypeString,
					Description: "processing, completed or failed",
				},
				{
					Name:        "outputs",
					Type:        rpc.ColumnTypeString,
					Description: "A JSON array of output URLs. NULL while processing. Download them promptly: output URLs may expire",
				},
				{
					Name:        "error",
					Type:        rpc.ColumnTypeString,
					Description: "The error message when the prediction failed. NULL otherwise",
				},
			},
		}, nil
}

type predictionsTable struct {
	client *atlasClient
}

type predictionsCursor struct {
	client *atlasClient
}

func (t *predictionsTable) CreateReader() rpc.ReaderInterface {
	return &predictionsCursor{client: t.client}
}

func (t *predictionsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	predictionID := constraints.GetColumnConstraint(0).GetStringValue()
	if predictionID == "" {
		return nil, true, fmt.Errorf("prediction_id must be set (e.g. WHERE prediction_id = 'abc123')")
	}

	// A status lookup is a free GET, so it is never memoized: a processing
	// prediction changes state over time and must always be re-fetched
	prediction, err := t.client.getPrediction(predictionID)
	if err != nil {
		return nil, true, err
	}

	status := normalizeStatus(prediction.Status)

	var outputs interface{}
	if urls := prediction.outputURLs(); len(urls) > 0 {
		if serialized, err := json.Marshal(urls); err == nil {
			outputs = string(serialized)
		}
	}

	var errOut interface{}
	if errMsg := string(prediction.Error); status != statusCompleted && errMsg != "" {
		errOut = errMsg
	}

	row := []interface{}{status, outputs, errOut}
	return [][]interface{}{row}, true, nil
}

func (t *predictionsTable) Close() error {
	return nil
}
