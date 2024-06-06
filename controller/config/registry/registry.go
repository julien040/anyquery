package registry

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/julien040/anyquery/controller/config/model"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schema_registry.json
var schema string

var DefaultRegistryBasePath = "https://registry.anyquery.dev/"

var anyqueryVersion = "0.0.1"

// Validates the given JSON string against the schema_registry.json schema.
func validateSchema(registry []byte) error {
	sch, err := jsonschema.CompileString("registry.json", schema)
	if err != nil {
		return fmt.Errorf("error compiling schema: %v", err)
	}

	var unmarshaled map[string]interface{}

	json.Unmarshal(registry, &unmarshaled)

	if err := sch.Validate(unmarshaled); err != nil {
		return fmt.Errorf("error validating registry: %v", err)
	}

	return nil
}

// Downloads the registry at the given URL, ensures it is a valid registry, and returns its contents.
func downloadRegistry(url string) ([]byte, error) {
	// Download registry
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, fmt.Errorf("error downloading registry from %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Read registry
	registry, err := io.ReadAll(resp.Body)

	if err != nil {
		return []byte{}, fmt.Errorf("error reading registry from %s: %v", url, err)
	}

	// Validate registry
	if err := validateSchema(registry); err != nil {
		return []byte{}, fmt.Errorf("error validating registry from %s: %v", url, err)
	}

	return registry, nil
}

// hashBytes returns the SHA256 hash of the given byte slice.
func hashBytes(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func registryExists(queries *model.Queries, name string) bool {
	ctx := context.Background()
	_, err := queries.GetRegistry(ctx, name)
	return err == nil
}

// Add a new registry to the list of registries in the database.
//
// It ensures the registry is valid, does not already exist, and then adds it to the list of registries.
func AddNewRegistry(queries *model.Queries, name string, url string) error {
	// Check if registry already exists
	ctx := context.Background()
	if registryExists(queries, name) {
		return fmt.Errorf("registry %s already exists", name)
	}

	// Download registry
	registry, err := downloadRegistry(url)
	if err != nil {
		return fmt.Errorf("error downloading registry: %v", err)
	}

	// Add registry
	err = queries.AddRegistry(ctx, model.AddRegistryParams{
		Name:             name,
		Url:              url,
		Registryjson:     string(registry),
		Lastupdated:      time.Now().Unix(),
		Checksumregistry: hashBytes(registry),
	})
	if err != nil {
		return fmt.Errorf("error adding registry: %v", err)
	}

	return nil
}

// Update the JSON content in the database of the given registry.
//
// If the registry does not exist, it returns an error.
// If the registry has not changed, it does nothing.
func UpdateRegistry(queries *model.Queries, name string) error {
	// Check if registry exists
	ctx := context.Background()
	if !registryExists(queries, name) {
		return fmt.Errorf("registry %s does not exist", name)
	}

	// Get registry
	registry, err := queries.GetRegistry(ctx, name)
	if err != nil {
		return fmt.Errorf("error getting registry: %v", err)
	}

	// Download registry
	newRegistry, err := downloadRegistry(registry.Url)
	if err != nil {
		return fmt.Errorf("error downloading registry: %v", err)
	}

	// Hash registry
	checksum := hashBytes(newRegistry)

	// Check if registry has changed
	if checksum == registry.Checksumregistry {
		// Set the last updated time
		return queries.UpdateRegistryFetched(ctx, name)
	}

	// Update registry
	err = queries.UpdateRegistry(ctx, model.UpdateRegistryParams{
		Name:             name,
		Registryjson:     string(newRegistry),
		Lastupdated:      time.Now().Unix(),
		Checksumregistry: checksum,
		Url:              registry.Url,
	})
	if err != nil {
		return fmt.Errorf("error updating registry: %v", err)
	}

	return nil
}

func LoadDefaultRegistry(queries *model.Queries) error {
	// Compute the path to the default registry
	path, err := url.JoinPath(DefaultRegistryBasePath, "v0", "registry.json")
	if err != nil {
		return fmt.Errorf("error joining path while loading default registry: %v", err)
	}

	// Add default registry
	return AddNewRegistry(queries, "default", path)
}

func LoadRegistry(queries *model.Queries, name string) (model.Registry, Registry, error) {
	ctx := context.Background()
	registry, err := queries.GetRegistry(ctx, name)
	if err != nil {
		return model.Registry{}, Registry{}, fmt.Errorf("error getting registry: %v", err)
	}

	var unmarshaled Registry
	err = json.Unmarshal([]byte(registry.Registryjson), &unmarshaled)
	if err != nil {
		return model.Registry{}, Registry{}, fmt.Errorf("error unmarshaling registry: %v", err)
	}

	return registry, unmarshaled, nil
}
