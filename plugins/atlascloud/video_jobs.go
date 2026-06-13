package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/julien040/anyquery/rpc"
)

func videoJobsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}
	baseURL := getBaseURL(args.UserConfig.GetString("base_url"))

	// One job store per profile: the API key (and base URL) identify the
	// profile from the plugin's point of view
	profileHash := md5.Sum([]byte(apiKey + baseURL))
	dir := path.Join(xdg.CacheHome, "anyquery", "plugins", "atlascloud", "video_jobs", fmt.Sprintf("%x", profileHash))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, nil, fmt.Errorf("failed to create the video job store directory: %w", err)
	}
	store, err := acquireJobStore(dir, profileHash[:])
	if err != nil {
		return nil, nil, err
	}

	return &videoJobsTable{
			client: newAtlasClient(apiKey, baseURL),
			store:  store,
		}, &rpc.DatabaseSchema{
			PrimaryKey:    0,
			HandlesInsert: true,
			HandlesDelete: true,
			// Each INSERT must submit immediately; batching video jobs makes
			// no sense
			BufferInsert: 1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "prediction_id",
					Type:        rpc.ColumnTypeString,
					Description: "The Atlas Cloud prediction ID of the job. Generated on INSERT",
				},
				{
					Name:        "model",
					Type:        rpc.ColumnTypeString,
					Description: "The model used (e.g. bytedance/seedance-2.0/text-to-video). List video models with SELECT * FROM atlascloud_models WHERE modality = 'video'. Required on INSERT",
				},
				{
					Name:        "prompt",
					Type:        rpc.ColumnTypeString,
					Description: "The text description of the video, as submitted. Required on INSERT",
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
					Description: "The error message when the job failed. NULL otherwise",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeString,
					Description: "When the job was submitted (RFC3339 format)",
				},
				{
					Name:        "image_url",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  false,
					Description: "An optional source image URL for image-to-video models. Only used on INSERT",
				},
				{
					Name:        "extra_params",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  false,
					Description: "An optional JSON object of model-specific parameters merged into the request (e.g. '{\"duration\": 5}'). Only used on INSERT",
				},
			},
		}, nil
}

type videoJobsTable struct {
	client *atlasClient
	store  *jobStore
}

type videoJobsCursor struct {
	client *atlasClient
	store  *jobStore
}

func (t *videoJobsTable) CreateReader() rpc.ReaderInterface {
	return &videoJobsCursor{
		client: t.client,
		store:  t.store,
	}
}

// Query returns all stored jobs. Jobs still processing are polled once (no
// waiting loop); jobs in a terminal state are returned from local storage
// without re-polling.
func (t *videoJobsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	jobs, err := t.store.list()
	if err != nil {
		return nil, true, err
	}

	rows := make([][]interface{}, 0, len(jobs))
	for _, job := range jobs {
		if job.Status == statusProcessing {
			polled, err := t.client.getPrediction(job.PredictionID)
			if err != nil {
				// Return the stored state; the next SELECT will retry
				log.Printf("failed to poll the video job %s: %v", job.PredictionID, err)
			} else {
				job.Status = normalizeStatus(polled.Status)
				job.Outputs = polled.outputURLs()
				job.Error = string(polled.Error)
				if err := t.store.put(job); err != nil {
					log.Printf("failed to update the video job %s: %v", job.PredictionID, err)
				}
			}
		}

		var outputs interface{}
		if len(job.Outputs) > 0 {
			if serialized, err := json.Marshal(job.Outputs); err == nil {
				outputs = string(serialized)
			}
		}

		rows = append(rows, []interface{}{
			job.PredictionID,
			job.Model,
			job.Prompt,
			job.Status,
			outputs,
			stringOrNil(job.Error),
			job.CreatedAt,
		})
	}

	return rows, true, nil
}

// Insert submits one video generation job per row and persists it locally.
// Row layout: [prediction_id, model, prompt, status, outputs, error,
// created_at, image_url, extra_params] — parameter columns are included on
// INSERT, and unspecified columns are nil.
func (t *videoJobsTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		model := getString(row, 1)
		if model == "" {
			return fmt.Errorf("model must be set (e.g. INSERT INTO atlascloud_video_jobs(model, prompt) VALUES ('bytedance/seedance-2.0/text-to-video', '...'))")
		}
		prompt := getString(row, 2)
		if prompt == "" {
			return fmt.Errorf("prompt must be set (e.g. INSERT INTO atlascloud_video_jobs(model, prompt) VALUES ('...', 'Ocean waves at sunset'))")
		}
		imageURL := getString(row, 7)
		extraParams := getString(row, 8)

		body, err := buildGenerationBody(model, prompt, imageURL, extraParams)
		if err != nil {
			return err
		}

		prediction, err := t.client.submitGeneration(endpointGenerateVideo, body)
		if err != nil {
			return fmt.Errorf("failed to submit the video job: %w", err)
		}

		job := &jobRecord{
			PredictionID: prediction.ID,
			Model:        model,
			Prompt:       prompt,
			ImageURL:     imageURL,
			ExtraParams:  extraParams,
			Status:       normalizeStatus(prediction.Status),
			Error:        string(prediction.Error),
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		}
		if err := t.store.put(job); err != nil {
			return fmt.Errorf("the video job %s was submitted but could not be stored locally: %w", prediction.ID, err)
		}
		if err := t.store.prune(maxStoredJobs); err != nil {
			log.Printf("failed to prune the video job store: %v", err)
		}
	}
	return nil
}

// Delete removes jobs from local storage. It does not cancel the generation
// on Atlas Cloud.
func (t *videoJobsTable) Delete(primaryKeys []interface{}) error {
	for _, primaryKey := range primaryKeys {
		id, ok := primaryKey.(string)
		if !ok {
			return fmt.Errorf("the primary key is not a string")
		}
		if err := t.store.delete(id); err != nil {
			return err
		}
	}
	return nil
}

func (t *videoJobsTable) Close() error {
	return releaseJobStore(t.store)
}
