package api

type DescribeResp struct {
	Fields []DescribeField `json:"fields"`
}

type DescribeField struct {
	Filterable bool   `json:"filterable"`
	Updateable bool   `json:"updateable"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Label      string `json:"label"`
}

type Rows struct {
	TotalSize      int64  `json:"totalSize"`
	Done           bool   `json:"done"`
	Records        []Row  `json:"records"`
	NextRecordsURL string `json:"nextRecordsUrl"`
}

type Row map[string]interface{}

// Insert

type InsertObject map[string]interface{}

type InsertUpdateRequest struct {
	Records   []InsertObject `json:"records"`
	AllOrNone bool           `json:"allOrNone"`
}

type InsertUpdateObjectType struct {
	Type string `json:"type"`
}

type InsertUpdateResponses []InsertUpdateResponse

type InsertUpdateResponse struct {
	Success bool                `json:"success"`
	Errors  []InsertUpdateError `json:"errors"`
	ID      *string             `json:"id,omitempty"`
}

type InsertUpdateError struct {
	StatusCode string        `json:"statusCode"`
	Message    string        `json:"message"`
	Fields     []interface{} `json:"fields"`
}
