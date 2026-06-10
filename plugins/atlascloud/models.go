package main

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// The catalog rarely changes, so it is cached on disk for a few hours.
// Users can force a refresh with SELECT clear_plugin_cache('atlascloud')
const catalogCacheTTL = 6 * time.Hour

func modelsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}
	baseURL := getBaseURL(args.UserConfig.GetString("base_url"))

	pathHash := md5.Sum([]byte(apiKey + baseURL))
	encryptionKey := sha256.Sum256([]byte(apiKey))
	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"atlascloud", "models", fmt.Sprintf("%x", pathHash)},
		EncryptionKey: encryptionKey[:],
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the cache: %w", err)
	}

	return &modelsTable{
			client: newAtlasClient(apiKey, baseURL),
			cache:  cache,
		}, &rpc.DatabaseSchema{
			PrimaryKey: 0,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The model identifier to pass to the llm, image and video_jobs tables (e.g. deepseek-ai/DeepSeek-V3-0324)",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The display name of the model",
				},
				{
					Name:        "modality",
					Type:        rpc.ColumnTypeString,
					Description: "The modality of the model: llm, image, video or audio",
				},
				{
					Name:        "provider",
					Type:        rpc.ColumnTypeString,
					Description: "The upstream provider of the model (e.g. BYTEDANCE, GOOGLE)",
				},
				{
					Name:        "description",
					Type:        rpc.ColumnTypeString,
					Description: "A short description of the model. Might be NULL",
				},
				{
					Name:        "price",
					Type:        rpc.ColumnTypeString,
					Description: "A JSON object with the pricing of the model (per-token prices for LLMs, per-generation base price for image/video). Might be NULL",
				},
			},
		}, nil
}

type modelsTable struct {
	client *atlasClient
	cache  *helper.Cache
}

type modelsCursor struct {
	client *atlasClient
	cache  *helper.Cache
}

func (t *modelsTable) CreateReader() rpc.ReaderInterface {
	return &modelsCursor{
		client: t.client,
		cache:  t.cache,
	}
}

func (t *modelsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	const cacheKey = "models"

	rows, _, err := t.cache.Get(cacheKey)
	if err == nil && len(rows) > 0 {
		return rows, true, nil
	}

	models, err := t.client.listModels()
	if err != nil {
		return nil, true, err
	}

	rows = make([][]interface{}, 0, len(models))
	for _, model := range models {
		// Models hidden from the Atlas Cloud console are not usable
		if model.DisplayConsole != nil && !*model.DisplayConsole {
			continue
		}
		price := interface{}(nil)
		if len(model.Price) > 0 && string(model.Price) != "null" {
			price = string(model.Price)
		}
		rows = append(rows, []interface{}{
			model.Model,
			model.DisplayName,
			modalityFromType(model.Type),
			stringOrNil(model.Organization),
			stringOrNil(model.Profile),
			price,
		})
	}

	if err := t.cache.Set(cacheKey, rows, nil, catalogCacheTTL); err != nil {
		log.Printf("failed to cache the model catalog: %v", err)
	}

	return rows, true, nil
}

func (t *modelsTable) Close() error {
	return t.cache.Close()
}
