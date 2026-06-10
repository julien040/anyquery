package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFlexStrings(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single string", `"https://example.com/a.png"`, []string{"https://example.com/a.png"}},
		{"array", `["https://a.png","https://b.png"]`, []string{"https://a.png", "https://b.png"}},
		{"null", `null`, nil},
		{"empty string", `""`, nil},
		{"unknown shape ignored", `{"weird": true}`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var parsed flexStrings
			if err := json.Unmarshal([]byte(c.input), &parsed); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual([]string(parsed), c.expected) {
				t.Fatalf("expected %v, got %v", c.expected, parsed)
			}
		})
	}
}

func TestFlexString(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"string", `"insufficient balance"`, "insufficient balance"},
		{"null", `null`, ""},
		{"object kept as JSON", `{"message":"oops"}`, `{"message":"oops"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var parsed flexString
			if err := json.Unmarshal([]byte(c.input), &parsed); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(parsed) != c.expected {
				t.Fatalf("expected %q, got %q", c.expected, parsed)
			}
		})
	}
}

func TestPredictionResponseParsing(t *testing.T) {
	// Shape documented for GET /api/v1/model/prediction/{id}
	payload := `{"data": {"id": "pred-1", "status": "completed", "outputs": ["https://out.png"], "error": null}}`
	parsed := predictionResponse{}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Data.ID != "pred-1" || normalizeStatus(parsed.Data.Status) != statusCompleted {
		t.Fatalf("unexpected prediction: %+v", parsed.Data)
	}
	if urls := parsed.Data.outputURLs(); len(urls) != 1 || urls[0] != "https://out.png" {
		t.Fatalf("unexpected outputs: %v", parsed.Data.outputURLs())
	}

	// Some models return "output" (string) instead of "outputs"
	payload = `{"data": {"id": "pred-2", "status": "succeeded", "output": "https://single.mp4"}}`
	parsed = predictionResponse{}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if urls := parsed.Data.outputURLs(); len(urls) != 1 || urls[0] != "https://single.mp4" {
		t.Fatalf("unexpected outputs: %v", parsed.Data.outputURLs())
	}
	if normalizeStatus(parsed.Data.Status) != statusCompleted {
		t.Fatalf("succeeded should normalize to completed")
	}

	// "outputs" wins over "output" when both are present
	payload = `{"data": {"id": "pred-3", "status": "processing", "output": "https://old.png", "outputs": ["https://new.png"]}}`
	parsed = predictionResponse{}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if urls := parsed.Data.outputURLs(); len(urls) != 1 || urls[0] != "https://new.png" {
		t.Fatalf("unexpected outputs: %v", parsed.Data.outputURLs())
	}
}

func TestNormalizeStatus(t *testing.T) {
	cases := map[string]string{
		"completed":  statusCompleted,
		"succeeded":  statusCompleted,
		"Succeeded":  statusCompleted,
		"failed":     statusFailed,
		"canceled":   statusFailed,
		"processing": statusProcessing,
		"starting":   statusProcessing,
		"queued":     statusProcessing,
		"":           statusProcessing,
	}
	for input, expected := range cases {
		if got := normalizeStatus(input); got != expected {
			t.Errorf("normalizeStatus(%q) = %q, expected %q", input, got, expected)
		}
	}
}

func TestBuildGenerationBody(t *testing.T) {
	// Explicit columns always win over extra_params
	body, err := buildGenerationBody("kling-v2.0", "waves", "https://img.png", `{"duration": 5, "model": "evil-override"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["model"] != "kling-v2.0" || body["prompt"] != "waves" || body["image_url"] != "https://img.png" {
		t.Fatalf("unexpected body: %v", body)
	}
	if body["duration"] != float64(5) {
		t.Fatalf("extra_params not merged: %v", body)
	}

	// No image_url key when the column is not set
	body, err = buildGenerationBody("kling-v2.0", "waves", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := body["image_url"]; exists {
		t.Fatalf("image_url should not be set: %v", body)
	}

	// Invalid extra_params is a user error
	if _, err := buildGenerationBody("m", "p", "", "not json"); err == nil {
		t.Fatalf("expected an error for invalid extra_params")
	}
}

func TestChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing auth header")
		}
		request := chatRequest{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("invalid request body: %v", err)
		}
		if request.Stream {
			t.Errorf("stream must always be false")
		}
		if len(request.Messages) != 2 || request.Messages[0].Role != "system" {
			t.Errorf("unexpected messages: %v", request.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"model": "deepseek-v3",
			"choices": [{"message": {"role": "assistant", "content": "Hello!"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 12, "completion_tokens": 3}
		}`))
	}))
	defer server.Close()

	client := newAtlasClient("test-key", server.URL)
	response, err := client.chatCompletion(chatRequest{
		Model: "deepseek-v3",
		Messages: []chatMessage{
			{Role: "system", Content: "Be brief."},
			{Role: "user", Content: "Say hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Choices[0].Message.Content != "Hello!" || response.Choices[0].FinishReason != "stop" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Usage.PromptTokens != 12 || response.Usage.CompletionTokens != 3 {
		t.Fatalf("unexpected usage: %+v", response.Usage)
	}
}

func TestChatCompletionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"code":402,"msg":"insufficient balance"}`))
	}))
	defer server.Close()

	client := newAtlasClient("test-key", server.URL)
	_, err := client.chatCompletion(chatRequest{Model: "m", Messages: []chatMessage{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatalf("expected an error")
	}
	expected := "chat completion failed (HTTP 402): insufficient balance"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestSubmitAndPollGeneration(t *testing.T) {
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/model/generateImage":
			body := map[string]interface{}{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("invalid request body: %v", err)
			}
			if body["model"] != "seedream-3.0" || body["prompt"] != "a garden" {
				t.Errorf("unexpected body: %v", body)
			}
			w.Write([]byte(`{"data": {"id": "abc123", "status": "processing"}}`))
		case "/api/v1/model/prediction/abc123":
			polls++
			if polls < 2 {
				w.Write([]byte(`{"data": {"id": "abc123", "status": "processing"}}`))
			} else {
				w.Write([]byte(`{"data": {"id": "abc123", "status": "completed", "outputs": ["https://img.png"], "error": null}}`))
			}
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newAtlasClient("test-key", server.URL)
	body, err := buildGenerationBody("seedream-3.0", "a garden", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prediction, err := client.submitGeneration(endpointGenerateImage, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prediction.ID != "abc123" || normalizeStatus(prediction.Status) != statusProcessing {
		t.Fatalf("unexpected prediction: %+v", prediction)
	}

	polled, err := client.getPrediction(prediction.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalizeStatus(polled.Status) != statusProcessing {
		t.Fatalf("first poll should still be processing")
	}
	polled, err = client.getPrediction(prediction.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalizeStatus(polled.Status) != statusCompleted || polled.outputURLs()[0] != "https://img.png" {
		t.Fatalf("unexpected prediction: %+v", polled)
	}
}

func TestSubmitGenerationWithoutID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": {}}`))
	}))
	defer server.Close()

	client := newAtlasClient("test-key", server.URL)
	if _, err := client.submitGeneration(endpointGenerateImage, map[string]interface{}{}); err == nil {
		t.Fatalf("expected an error when no prediction ID is returned")
	}
}

func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code": 200, "data": [
			{"model": "deepseek-ai/DeepSeek-V3", "type": "Text", "displayName": "DeepSeek V3", "organization": "DEEPSEEK", "profile": "An LLM", "price": {"actual": {"input_price": "1"}}},
			{"model": "kling/kling-v2.0", "type": "Video", "displayName": "Kling 2.0", "display_console": false}
		]}`))
	}))
	defer server.Close()

	client := newAtlasClient("test-key", server.URL)
	models, err := client.listModels()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Model != "deepseek-ai/DeepSeek-V3" || string(models[0].Price) != `{"actual": {"input_price": "1"}}` {
		t.Fatalf("unexpected model: %+v", models[0])
	}
	if models[1].DisplayConsole == nil || *models[1].DisplayConsole {
		t.Fatalf("display_console should be false")
	}
}

func TestModalityFromType(t *testing.T) {
	cases := map[string]interface{}{
		"Text":  "llm",
		"Image": "image",
		"Video": "video",
		"Audio": "audio",
		"":      nil,
	}
	for input, expected := range cases {
		if got := modalityFromType(input); got != expected {
			t.Errorf("modalityFromType(%q) = %v, expected %v", input, got, expected)
		}
	}
}

func TestMemoStore(t *testing.T) {
	memo := newMemoStore()
	key := memoKey("llm", "model", "prompt")
	if _, hit := memo.get(key); hit {
		t.Fatalf("unexpected hit on an empty store")
	}
	row := []interface{}{"response", "stop", int64(1), int64(2)}
	memo.set(key, row)
	cached, hit := memo.get(key)
	if !hit || !reflect.DeepEqual(cached, row) {
		t.Fatalf("expected a hit with %v, got %v (hit=%v)", row, cached, hit)
	}
	// Different tuples must not collide
	if _, hit := memo.get(memoKey("llm", "model", "other prompt")); hit {
		t.Fatalf("unexpected hit for a different tuple")
	}
	// ("ab", "c") must not collide with ("a", "bc")
	if memoKey("ab", "c") == memoKey("a", "bc") {
		t.Fatalf("memo keys collide across part boundaries")
	}
}

func TestGetString(t *testing.T) {
	row := []interface{}{nil, "text", int64(42), 3.14, true}
	if getString(row, 0) != "" {
		t.Errorf("nil should map to an empty string")
	}
	if getString(row, 1) != "text" {
		t.Errorf("unexpected string value")
	}
	if getString(row, 2) != "42" {
		t.Errorf("unexpected int value")
	}
	if getString(row, 10) != "" || getString(row, -1) != "" {
		t.Errorf("out-of-bounds access should return an empty string")
	}
}

func TestGetBaseURL(t *testing.T) {
	if getBaseURL("") != defaultBaseURL {
		t.Errorf("empty base_url should fall back to the default")
	}
	if getBaseURL("https://example.com/") != "https://example.com" {
		t.Errorf("trailing slash should be trimmed")
	}
}
