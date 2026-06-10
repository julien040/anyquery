package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const defaultBaseURL = "https://api.atlascloud.ai"

const (
	endpointModels        = "/api/v1/models"
	endpointChat          = "/v1/chat/completions"
	endpointGenerateImage = "/api/v1/model/generateImage"
	endpointGenerateVideo = "/api/v1/model/generateVideo"
	endpointPrediction    = "/api/v1/model/prediction/{id}"
)

// atlasClient is a thin wrapper around resty for the Atlas Cloud API.
// All requests authenticate with "Authorization: Bearer <api_key>".
type atlasClient struct {
	http *resty.Client
}

func newAtlasClient(apiKey string, baseURL string) *atlasClient {
	client := resty.New().
		SetBaseURL(baseURL).
		SetAuthToken(apiKey).
		SetTimeout(3 * time.Minute).
		SetRetryCount(3).
		SetRetryMaxWaitTime(60 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Only retry rate limits: a 429 means the request was rejected, so
			// replaying it is safe. A 5xx on a generation POST could have
			// already created a billable task, so it must not be retried.
			return err == nil && r != nil && r.StatusCode() == http.StatusTooManyRequests
		}).
		SetRetryAfter(func(c *resty.Client, r *resty.Response) (time.Duration, error) {
			if r != nil {
				if retryAfter := r.Header().Get("Retry-After"); retryAfter != "" {
					if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 && secs <= 60 {
						return time.Duration(secs) * time.Second, nil
					}
				}
			}
			return 2 * time.Second, nil
		})

	return &atlasClient{http: client}
}

// getBaseURL returns the base_url from the user config (without a trailing
// slash), or the default Atlas Cloud endpoint
func getBaseURL(rawBaseURL string) string {
	baseURL := strings.TrimSuffix(strings.TrimSpace(rawBaseURL), "/")
	if baseURL == "" {
		return defaultBaseURL
	}
	return baseURL
}

// errorMessage extracts a human-readable message from an API error response.
// It must never include the request (which carries the API key).
func errorMessage(resp *resty.Response) string {
	parsed := struct {
		Msg     string          `json:"msg"`
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}{}
	if err := json.Unmarshal(resp.Body(), &parsed); err == nil {
		if parsed.Msg != "" && parsed.Msg != "succeed" {
			return parsed.Msg
		}
		if parsed.Message != "" {
			return parsed.Message
		}
		if len(parsed.Error) > 0 && string(parsed.Error) != "null" {
			// The error field can be a string or an object {message: ...}
			var str string
			if json.Unmarshal(parsed.Error, &str) == nil && str != "" {
				return str
			}
			var obj struct {
				Message string `json:"message"`
			}
			if json.Unmarshal(parsed.Error, &obj) == nil && obj.Message != "" {
				return obj.Message
			}
		}
	}

	// Fall back to the raw body, truncated
	raw := strings.TrimSpace(string(resp.Body()))
	if len(raw) > 300 {
		raw = raw[:300] + "…"
	}
	if raw == "" {
		raw = resp.Status()
	}
	return raw
}

/* ------------------------------- Catalog -------------------------------- */

// catalogModel is one entry of GET /api/v1/models
type catalogModel struct {
	Model          string          `json:"model"`
	Type           string          `json:"type"` // Text, Image, Video, Audio
	DisplayName    string          `json:"displayName"`
	Profile        string          `json:"profile"`
	Organization   string          `json:"organization"`
	DisplayConsole *bool           `json:"display_console"`
	Price          json.RawMessage `json:"price"`
}

type catalogResponse struct {
	Data []catalogModel `json:"data"`
}

func (c *atlasClient) listModels() ([]catalogModel, error) {
	body := catalogResponse{}
	resp, err := c.http.R().
		SetResult(&body).
		Get(endpointModels)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the model catalog: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("failed to fetch the model catalog (HTTP %d): %s", resp.StatusCode(), errorMessage(resp))
	}
	return body.Data, nil
}

// modalityFromType maps the API's model type to the modality column.
// "Text" is exposed as "llm" so that it matches the atlascloud_llm table name.
func modalityFromType(modelType string) interface{} {
	if modelType == "" {
		return nil
	}
	if strings.EqualFold(modelType, "Text") {
		return "llm"
	}
	return strings.ToLower(modelType)
}

/* --------------------------- Chat completions --------------------------- */

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int64        `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

type chatUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage *chatUsage `json:"usage"`
}

func (c *atlasClient) chatCompletion(request chatRequest) (*chatResponse, error) {
	body := chatResponse{}
	resp, err := c.http.R().
		SetBody(request).
		SetResult(&body).
		Post(endpointChat)
	if err != nil {
		return nil, fmt.Errorf("failed to call the chat completion API: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("chat completion failed (HTTP %d): %s", resp.StatusCode(), errorMessage(resp))
	}
	return &body, nil
}

/* ------------------------------ Predictions ----------------------------- */

// flexStrings unmarshals a JSON string, an array of strings, or null.
// The prediction endpoint returns outputs either as "output" (string or
// array) or "outputs" (array) depending on the model.
type flexStrings []string

func (f *flexStrings) UnmarshalJSON(data []byte) error {
	*f = nil
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "" {
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single != "" {
			*f = []string{single}
		}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err == nil {
		*f = many
		return nil
	}
	// Unknown shape: ignore rather than failing the whole response
	return nil
}

// flexString unmarshals a JSON string, null, or any other value
// (serialized back to JSON). Used for the error field whose shape is
// weakly documented.
type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*f = flexString(str)
		return nil
	}
	if strings.TrimSpace(string(data)) == "null" {
		*f = ""
		return nil
	}
	*f = flexString(data)
	return nil
}

type predictionData struct {
	ID      string      `json:"id"`
	Status  string      `json:"status"`
	Output  flexStrings `json:"output"`
	Outputs flexStrings `json:"outputs"`
	Error   flexString  `json:"error"`
}

// outputURLs returns the output URLs, whichever field they were returned in
func (p *predictionData) outputURLs() []string {
	if len(p.Outputs) > 0 {
		return p.Outputs
	}
	return p.Output
}

type predictionResponse struct {
	Data predictionData `json:"data"`
}

// statusProcessing/Completed/Failed are the normalized statuses exposed in
// the tables. The API also uses values like "starting" or "succeeded".
const (
	statusProcessing = "processing"
	statusCompleted  = "completed"
	statusFailed     = "failed"
	statusTimeout    = "timeout"
)

// normalizeStatus maps the API's status vocabulary to the values documented
// in the tables. Unknown statuses are treated as still processing so that
// polling continues until a timeout.
func normalizeStatus(status string) string {
	switch strings.ToLower(status) {
	case "completed", "succeeded", "success":
		return statusCompleted
	case "failed", "error", "canceled", "cancelled":
		return statusFailed
	default:
		return statusProcessing
	}
}

// buildGenerationBody builds the request body for generateImage and
// generateVideo. The optional extra_params JSON object is merged into the
// body; the explicit columns (model, prompt, image_url) always win.
func buildGenerationBody(model string, prompt string, imageURL string, extraParams string) (map[string]interface{}, error) {
	body := map[string]interface{}{}
	if extraParams != "" {
		if err := json.Unmarshal([]byte(extraParams), &body); err != nil {
			return nil, fmt.Errorf("extra_params must be a JSON object: %w", err)
		}
	}
	body["model"] = model
	body["prompt"] = prompt
	if imageURL != "" {
		body["image_url"] = imageURL
	}
	return body, nil
}

// submitGeneration submits an async generation request (image or video) and
// returns the created prediction
func (c *atlasClient) submitGeneration(endpoint string, body map[string]interface{}) (*predictionData, error) {
	result := predictionResponse{}
	resp, err := c.http.R().
		SetBody(body).
		SetResult(&result).
		Post(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to submit the generation request: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("generation request failed (HTTP %d): %s", resp.StatusCode(), errorMessage(resp))
	}
	if result.Data.ID == "" {
		return nil, fmt.Errorf("Atlas Cloud did not return a prediction ID")
	}
	return &result.Data, nil
}

// getPrediction polls the status of a prediction once
func (c *atlasClient) getPrediction(id string) (*predictionData, error) {
	result := predictionResponse{}
	resp, err := c.http.R().
		SetResult(&result).
		SetPathParam("id", id).
		Get(endpointPrediction)
	if err != nil {
		return nil, fmt.Errorf("failed to poll the prediction: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("failed to poll the prediction (HTTP %d): %s", resp.StatusCode(), errorMessage(resp))
	}
	return &result.Data, nil
}

/* ------------------------------- Row helpers ---------------------------- */

// getString returns the string at the given index of an INSERT row.
// It is nil-safe and bounds-safe because users can omit columns.
func getString(row []interface{}, index int) string {
	if index < 0 || index >= len(row) || row[index] == nil {
		return ""
	}
	switch val := row[index].(type) {
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	}
	return ""
}

// stringOrNil returns nil for empty strings so they show as NULL
func stringOrNil(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
