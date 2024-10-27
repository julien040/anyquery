package main

type Properties struct {
	Results []Property `json:"results"`
}

type Property struct {
	UpdatedAt            string               `json:"updatedAt,omitempty"`
	CreatedAt            string               `json:"createdAt,omitempty"`
	Name                 string               `json:"name"`
	Label                string               `json:"label"`
	Type                 string               `json:"type"`
	FieldType            string               `json:"fieldType"`
	Description          string               `json:"description"`
	GroupName            string               `json:"groupName"`
	Options              []Option             `json:"options"`
	DisplayOrder         int64                `json:"displayOrder"`
	Calculated           bool                 `json:"calculated"`
	ExternalOptions      bool                 `json:"externalOptions"`
	HasUniqueValue       bool                 `json:"hasUniqueValue"`
	Hidden               bool                 `json:"hidden"`
	HubspotDefined       bool                 `json:"hubspotDefined"`
	ShowCurrencySymbol   *bool                `json:"showCurrencySymbol,omitempty"`
	ModificationMetadata ModificationMetadata `json:"modificationMetadata"`
	FormField            bool                 `json:"formField"`
	DataSensitivity      string               `json:"dataSensitivity"`
	CalculationFormula   *string              `json:"calculationFormula,omitempty"`
	ReferencedObjectType *string              `json:"referencedObjectType,omitempty"`
}

type ModificationMetadata struct {
	Archivable         bool  `json:"archivable"`
	ReadOnlyDefinition bool  `json:"readOnlyDefinition"`
	ReadOnlyValue      bool  `json:"readOnlyValue"`
	ReadOnlyOptions    *bool `json:"readOnlyOptions,omitempty"`
}

type Option struct {
	Label        string  `json:"label"`
	Value        string  `json:"value"`
	DisplayOrder int64   `json:"displayOrder"`
	Hidden       bool    `json:"hidden"`
	Description  *string `json:"description,omitempty"`
}

type Objects struct {
	Paging  ObjectsPaging `json:"paging"`
	Results []Object      `json:"results"`
}

type ObjectsPaging struct {
	Next Next `json:"next"`
}

type Next struct {
	Link  string `json:"link"`
	After string `json:"after"`
}

type Object struct {
	CreatedAt  string                 `json:"createdAt"`
	Archived   bool                   `json:"archived"`
	ArchivedAt string                 `json:"archivedAt"`
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
	UpdatedAt  string                 `json:"updatedAt"`
}

// Create/update/delete an object

type CreateUpdateBody struct {
	Inputs []InputUpdate `json:"inputs"`
}

type InputUpdate struct {
	ObjectWriteTraceID string                 `json:"objectWriteTraceId,omitempty"`
	Properties         map[string]interface{} `json:"properties,omitempty"`
	ID                 string                 `json:"id,omitempty"`
}
