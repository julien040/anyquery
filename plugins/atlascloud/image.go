package main

import (
	"fmt"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// Atlas Cloud recommends polling images every 2 seconds (typical generation
// takes 2–10 s). The 90-second cap keeps a query from blocking forever; some
// MySQL clients in server mode may need their read timeout raised to match.
const (
	imagePollInterval = 2 * time.Second
	imageTimeout      = 90 * time.Second
)

func imageCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}
	baseURL := getBaseURL(args.UserConfig.GetString("base_url"))

	return &imageTable{
			client: newAtlasClient(apiKey, baseURL),
			memo:   newMemoStore(),
		}, &rpc.DatabaseSchema{
			PrimaryKey: -1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "model",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The model to use (e.g. bytedance/seedream-3.0/text-to-image). List image models with SELECT * FROM atlascloud_models WHERE modality = 'image'",
				},
				{
					Name:        "prompt",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The text description of the image to generate",
				},
				{
					Name:        "image_url",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  false,
					Description: "An optional source image URL for image-to-image models",
				},
				{
					Name:        "extra_params",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  false,
					Description: "An optional JSON object of model-specific parameters merged into the request (e.g. '{\"image_size\": \"1024x1024\"}')",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL of the generated image. NULL if the generation failed. Download it promptly: output URLs may expire",
				},
				{
					Name:        "prediction_id",
					Type:        rpc.ColumnTypeString,
					Description: "The Atlas Cloud prediction ID of the generation",
				},
				{
					Name:        "status",
					Type:        rpc.ColumnTypeString,
					Description: "completed, failed or timeout",
				},
				{
					Name:        "error",
					Type:        rpc.ColumnTypeString,
					Description: "The error message when the generation failed. NULL otherwise",
				},
			},
		}, nil
}

type imageTable struct {
	client *atlasClient
	memo   *memoStore
}

type imageCursor struct {
	client *atlasClient
	memo   *memoStore
}

func (t *imageTable) CreateReader() rpc.ReaderInterface {
	return &imageCursor{
		client: t.client,
		memo:   t.memo,
	}
}

func (t *imageCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	model := constraints.GetColumnConstraint(0).GetStringValue()
	if model == "" {
		return nil, true, fmt.Errorf("model must be set (e.g. WHERE model = 'bytedance/seedream-3.0/text-to-image')")
	}
	prompt := constraints.GetColumnConstraint(1).GetStringValue()
	if prompt == "" {
		return nil, true, fmt.Errorf("prompt must be set (e.g. WHERE prompt = 'A serene Japanese garden')")
	}
	imageURL := constraints.GetColumnConstraint(2).GetStringValue()
	extraParams := constraints.GetColumnConstraint(3).GetStringValue()

	// Re-scans of the same input tuple within a statement must not trigger
	// a second paid generation
	key := memoKey("image", model, prompt, imageURL, extraParams)
	if row, hit := t.memo.get(key); hit {
		return [][]interface{}{row}, true, nil
	}

	// An invalid extra_params is a user error caught before any money is
	// spent, so it can fail the query
	body, err := buildGenerationBody(model, prompt, imageURL, extraParams)
	if err != nil {
		return nil, true, err
	}

	// From here on, failures are returned as rows (status/error set) rather
	// than Go errors, so batch generation over a join continues past
	// individual failures
	prediction, err := t.client.submitGeneration(endpointGenerateImage, body)
	if err != nil {
		row := []interface{}{nil, nil, statusFailed, err.Error()}
		t.memo.set(key, row)
		return [][]interface{}{row}, true, nil
	}

	status := normalizeStatus(prediction.Status)
	urls := prediction.outputURLs()
	errMsg := string(prediction.Error)
	deadline := time.Now().Add(imageTimeout)

	for status == statusProcessing {
		if time.Now().After(deadline) {
			status = statusTimeout
			errMsg = fmt.Sprintf("the generation was still processing after %s; it may still complete on Atlas Cloud (prediction id %s)", imageTimeout, prediction.ID)
			break
		}
		time.Sleep(imagePollInterval)

		polled, err := t.client.getPrediction(prediction.ID)
		if err != nil {
			// Transient polling error: keep trying until the deadline
			errMsg = err.Error()
			continue
		}
		status = normalizeStatus(polled.Status)
		urls = polled.outputURLs()
		errMsg = string(polled.Error)
	}

	var url interface{}
	if status == statusCompleted && len(urls) > 0 {
		url = urls[0]
	}
	var errOut interface{}
	if status != statusCompleted && errMsg != "" {
		errOut = errMsg
	}

	row := []interface{}{url, prediction.ID, status, errOut}
	t.memo.set(key, row)

	return [][]interface{}{row}, true, nil
}

func (t *imageTable) Close() error {
	return nil
}
