package main

type GetColsResp struct {
	Items []Column `json:"items"`
	Href  string   `json:"href"`
}

type Column struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Name       string  `json:"name"`
	Href       string  `json:"href"`
	Display    *bool   `json:"display,omitempty"`
	Format     Format  `json:"format"`
	Calculated *bool   `json:"calculated,omitempty"`
	Formula    *string `json:"formula,omitempty"`
}

type Format struct {
	Type                  string  `json:"type"`
	IsArray               bool    `json:"isArray"`
	Precision             *int64  `json:"precision,omitempty"`
	UseThousandsSeparator *bool   `json:"useThousandsSeparator,omitempty"`
	CurrencyCode          *string `json:"currencyCode,omitempty"`
	Format                *string `json:"format,omitempty"`
	Minimum               *int64  `json:"minimum,omitempty"`
	Maximum               *int64  `json:"maximum,omitempty"`
	Step                  *int64  `json:"step,omitempty"`
	DisplayType           *string `json:"displayType,omitempty"`
	Icon                  *string `json:"icon,omitempty"`
	DateFormat            *string `json:"dateFormat,omitempty"`
	TimeFormat            *string `json:"timeFormat,omitempty"`
	Table                 *Table  `json:"table,omitempty"`
	Display               *string `json:"display,omitempty"`
	Width                 *int64  `json:"width,omitempty"`
	Height                *int64  `json:"height,omitempty"`
	Style                 *string `json:"style,omitempty"`
}

type Table struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	TableType   string `json:"tableType"`
	Href        string `json:"href"`
	BrowserLink string `json:"browserLink"`
	Name        string `json:"name"`
}

// Get rows
type GetRows struct {
	Items         []Row  `json:"items"`
	Href          string `json:"href"`
	NextSyncToken string `json:"nextSyncToken"`
	NextPageToken string `json:"nextPageToken"`
}

type Row struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Href        string                 `json:"href"`
	Name        string                 `json:"name"`
	Index       int64                  `json:"index"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
	BrowserLink string                 `json:"browserLink"`
	Values      map[string]interface{} `json:"values"`
}

// Insert row
type InsertRow struct {
	Cells []InsertCell `json:"cells"`
}

type InsertCell struct {
	Column string      `json:"column"`
	Value  interface{} `json:"value"`
}
type InsertRowsBody struct {
	Rows []InsertRow `json:"rows"`

	// To update the rows
	KeyColumns []string `json:"keyColumns,omitempty"`
}

// Update row
type UpdateRow struct {
	Cells []InsertCell `json:"cells"`
}

type UpdateRowsBody struct {
	Row UpdateRow `json:"row"`
}

// Delete row
type DeleteRowBody struct {
	RowIds []string `json:"rowIds"`
}
