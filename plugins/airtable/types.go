package main

type GetSchemaResponses struct {
	Tables []Table `json:"tables"`
}

type Table struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	PrimaryFieldID string  `json:"primaryFieldId"`
	Fields         []Field `json:"fields"`
	Views          []Field `json:"views"`
}

type Field struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListRecordsResponse struct {
	Records []Record `json:"records"`
	Offset  string   `json:"offset"`
}

type Record struct {
	ID          string                 `json:"id"`
	CreatedTime string                 `json:"createdTime"`
	Fields      map[string]interface{} `json:"fields"`
}

// Insert

type InsertRecordRequest struct {
	Records []InsertRecordItem `json:"records"`
}

type InsertRecordItem struct {
	Fields map[string]interface{} `json:"fields"`
}

// Update

type UpdateRecordRequest struct {
	Records []UpdateRecordItem `json:"records"`
}

type UpdateRecordItem struct {
	ID     string                 `json:"id"`
	Fields map[string]interface{} `json:"fields"`
}

// Delete

type DeleteRecordRequest struct {
	Records []string `json:"records"`
}
