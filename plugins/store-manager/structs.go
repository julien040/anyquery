package main

type PluginsAPIPocketbase struct {
	Items      []PluginPocketbase `json:"items"`
	Page       int64              `json:"page"`
	PerPage    int64              `json:"perPage"`
	TotalItems int64              `json:"totalItems"`
	TotalPages int64              `json:"totalPages"`
}

type PluginPocketbase struct {
	Author         string   `json:"author"`
	CollectionID   string   `json:"collectionId"`
	CollectionName string   `json:"collectionName"`
	Created        string   `json:"created"`
	Description    string   `json:"description"`
	DisplayName    string   `json:"display_name"`
	Expand         Expand   `json:"expand"`
	Hidden         bool     `json:"hidden"`
	Homepage       string   `json:"homepage"`
	Icon           string   `json:"icon"`
	ID             string   `json:"id"`
	License        string   `json:"license"`
	Name           string   `json:"name"`
	PageContent    string   `json:"pageContent"`
	Type           string   `json:"type"`
	Updated        string   `json:"updated"`
	Versions       []string `json:"versions"`
}

type Expand struct {
	Versions []Version `json:"versions"`
}

type Version struct {
	CollectionID   string      `json:"collectionId"`
	CollectionName string      `json:"collectionName"`
	Created        string      `json:"created"`
	Files          []string    `json:"files"`
	ID             string      `json:"id"`
	MinimumVersion string      `json:"minimum_version"`
	Tables         []string    `json:"tables"`
	TablesMetadata interface{} `json:"tablesMetadata"`
	Updated        string      `json:"updated"`
	Version        string      `json:"version"`
}
