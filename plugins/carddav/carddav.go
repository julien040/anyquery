package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
)

func newCardDAVClient(config map[string]any) (*carddav.Client, error) {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	username, ok := config["username"].(string)
	if !ok || username == "" {
		return nil, fmt.Errorf("username is required")
	}

	password, ok := config["password"].(string)
	if !ok || password == "" {
		return nil, fmt.Errorf("password is required")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	httpClientWithAuth := webdav.HTTPClientWithBasicAuth(httpClient, username, password)

	client, err := carddav.NewClient(httpClientWithAuth, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create CardDAV client: %w", err)
	}

	return client, nil
}
