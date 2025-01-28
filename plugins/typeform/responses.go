package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

var retryableClient = retryablehttp.NewClient()
var client = resty.NewWithClient(retryableClient.StandardClient())

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func responsesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get the user config
	var formID, token string
	if rawInter, ok := args.UserConfig["form_id"]; ok {
		if formID, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("form_id should be a string")
		}
		if formID == "" {
			return nil, nil, fmt.Errorf("form_id should not be empty")
		}
	}

	if rawInter, ok := args.UserConfig["token"]; ok {
		if token, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("token should be a string")
		}
		if token == "" {
			return nil, nil, fmt.Errorf("token should not be empty")
		}
	}

	// Request the schema from typeform
	data := FormInfo{}

	res, err := client.R().SetHeader("Authorization", "Bearer "+token).SetResult(&data).Get(fmt.Sprintf("https://api.typeform.com/forms/%s", formID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get form schema: %w", err)
	}

	if res.IsError() {
		return nil, nil, fmt.Errorf("failed to get form schema(code: %d): %s", res.StatusCode(), res.String())
	}

	schema := []rpc.DatabaseSchemaColumn{
		{
			Name:        "id",
			Type:        rpc.ColumnTypeString,
			Description: "The ID of the response",
		},
		{
			Name:        "landed_at",
			Type:        rpc.ColumnTypeDateTime,
			Description: "The time the user landed on the form",
		},
		{
			Name:        "submitted_at",
			Type:        rpc.ColumnTypeDateTime,
			Description: "The time the user submitted the form",
		},
		{
			Name:        "user_agent",
			Type:        rpc.ColumnTypeString,
			Description: "The user agent of the user",
		},
		{
			Name:        "response_type",
			Type:        rpc.ColumnTypeString,
			Description: "The type of the response. One of started, partial, completed",
		},
	}

	mapIdColumnIndex := make(map[string]int)
	columnNameDedup := make(map[string]bool)

	supportedField := []string{"short_text", "long_text", "multiple_choice", "picture_choice", "yes_no", "number",
		"date", "rating", "opinion_scale", "email", "website", "file_upload", "dropdown", "legal",
		"ranking", "nps"}
	fieldNumber := []string{"number", "rating", "opinion_scale", "nps"}
	fieldBool := []string{"yes_no", "legal"}
	i := 0
	for _, field := range data.Fields {

		// Add subfields
		// We need to analyze the subfields first because some types are not supported
		// like a matrix, however, the subfields can be supported
		for _, subfield := range field.Properties.Fields {
			if !slices.Contains(supportedField, subfield.Type) {
				continue
			}
			subfield.Title = strings.ToLower(subfield.Title)
			subfield.Title = fmt.Sprintf("%s_%s", field.Title, subfield.Title)
			_, nameAlreadyExist := columnNameDedup[subfield.Title]
			for nameAlreadyExist {
				subfield.Title = subfield.Title + "_"
				_, nameAlreadyExist = columnNameDedup[subfield.Title]
			}

			columnNameDedup[subfield.Title] = true

			if slices.Contains(fieldNumber, subfield.Type) {
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        subfield.Title,
					Type:        rpc.ColumnTypeFloat,
					Description: fmt.Sprintf("The field %s (type: %s) of the form", subfield.Title, subfield.Type),
				})
			} else if slices.Contains(fieldBool, subfield.Type) {
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        subfield.Title,
					Type:        rpc.ColumnTypeInt,
					Description: fmt.Sprintf("The field %s (type: %s) of the form", subfield.Title, subfield.Type),
				})
			} else {
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        subfield.Title,
					Type:        rpc.ColumnTypeString,
					Description: fmt.Sprintf("The field %s (type: %s) of the form", subfield.Title, subfield.Type),
				})
			}

			mapIdColumnIndex[subfield.ID] = i
			i++
		}

		// Add the field
		if !slices.Contains(supportedField, field.Type) {
			continue
		}
		field.Title = strings.ToLower(field.Title)
		_, nameAlreadyExist := columnNameDedup[field.Title]
		for nameAlreadyExist {
			field.Title = field.Title + "_"
			_, nameAlreadyExist = columnNameDedup[field.Title]
		}

		if slices.Contains(fieldNumber, field.Type) {
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: field.Title,
				Type: rpc.ColumnTypeFloat,
			})
		} else if slices.Contains(fieldBool, field.Type) {
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: field.Title,
				Type: rpc.ColumnTypeInt,
			})
		} else {
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: field.Title,
				Type: rpc.ColumnTypeString,
			})
		}

		columnNameDedup[field.Title] = true

		mapIdColumnIndex[field.ID] = i
		i++

	}

	return &responsesTable{
			mapIdColumnIndex: mapIdColumnIndex,
			formID:           formID,
			token:            token,
		}, &rpc.DatabaseSchema{
			Columns: schema,
		}, nil
}

type responsesTable struct {
	mapIdColumnIndex map[string]int
	formID           string
	token            string
}

type responsesCursor struct {
	nextCursor       string
	formID           string
	token            string
	mapIdColumnIndex map[string]int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *responsesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the responses from typeform
	data := Responses{}

	res, err := client.R().SetHeader("Authorization", "Bearer "+t.token).
		SetResult(&data).
		SetQueryParams(map[string]string{
			"page_size": "1000",
			"after":     t.nextCursor,
		}).
		SetPathParam("form_id", t.formID).
		Get("https://api.typeform.com/forms/{form_id}/responses")

	if err != nil {
		return nil, true, fmt.Errorf("failed to get responses: %w", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("failed to get responses(code: %d): %s", res.StatusCode(), res.String())
	}

	// Update the cursor
	if len(data.Items) > 0 {
		t.nextCursor = data.Items[0].Token
	}

	// Prepare the rows
	rows := make([][]interface{}, 0, len(data.Items))
	for _, item := range data.Items {
		row := make([]interface{}, len(t.mapIdColumnIndex)+5)
		row[0] = item.ResponseID
		row[1] = item.LandedAt
		row[2] = item.SubmittedAt
		row[3] = item.Metadata.UserAgent
		row[4] = item.ResponseType
		for _, answer := range item.Answers {
			switch answer.Type {
			case "text":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.Text
			case "email":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.Email
			case "phone_number":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.PhoneNumber
			case "url":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.URL
			case "choice":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.Choice.Label
			case "boolean":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.Boolean
			case "number":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.Number
			case "date":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.Date
			case "choices":
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = answer.Choices.Labels
			default:
				row[t.mapIdColumnIndex[answer.Field.ID]+5] = nil
			}
		}
		rows = append(rows, row)
	}

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *responsesTable) CreateReader() rpc.ReaderInterface {
	return &responsesCursor{
		nextCursor:       "",
		formID:           t.formID,
		token:            t.token,
		mapIdColumnIndex: t.mapIdColumnIndex,
	}
}

// A destructor to clean up resources
func (t *responsesTable) Close() error {
	return nil
}
