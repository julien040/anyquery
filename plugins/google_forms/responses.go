package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/forms/v1"
	"google.golang.org/api/option"
)

var retryableClient = retryablehttp.NewClient()

type colType int

const (
	colTypeString colType = iota
	colTypeArray
	colTypeNumber
)

type col struct {
	colIndex int
	colType  colType
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func responsesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get from the user config the refresh token, the form id, the client id and the client secret
	var formID, token, clientID, clientSecret string
	if rawInter, ok := args.UserConfig["form_id"]; ok {
		if formID, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("form_id should be a string")
		}
		if formID == "" {
			return nil, nil, fmt.Errorf("form_id should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("form_id is required")
	}

	if rawInter, ok := args.UserConfig["token"]; ok {
		if token, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("token should be a string")
		}
		if token == "" {
			return nil, nil, fmt.Errorf("token should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("token is required")
	}

	if rawInter, ok := args.UserConfig["client_id"]; ok {
		if clientID, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("client_id should be a string")
		}
		if clientID == "" {
			return nil, nil, fmt.Errorf("client_id should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("client_id is required")
	}

	if rawInter, ok := args.UserConfig["client_secret"]; ok {
		if clientSecret, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("client_secret should be a string")
		}
		if clientSecret == "" {
			return nil, nil, fmt.Errorf("client_secret should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("client_secret is required")
	}

	// Request the schema from google forms using the client id and the client secret and the refresh token
	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{forms.FormsBodyReadonlyScope, forms.FormsResponsesReadonlyScope},
	}

	oauthClient := config.Client(context.Background(), &oauth2.Token{
		RefreshToken: token,
	})

	retryableClient = retryablehttp.NewClient()
	retryableClient.HTTPClient = oauthClient

	client, err := forms.NewService(context.Background(), option.WithHTTPClient(retryableClient.StandardClient()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create google forms client: %w", err)
	}
	formGetCall := client.Forms.Get(formID)
	form, err := formGetCall.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get form schema: %w", err)
	}

	tableDescription := strings.Builder{}
	if form.Info != nil {
		tableDescription.WriteString(form.Info.Title)
		tableDescription.WriteString("-")
		tableDescription.WriteString(form.Info.Description)
	}

	colIndex := 2 // 0 is reserved for the id, 1 for the created_at
	mapIdColInfo := make(map[string]col)
	columnNameDedup := make(map[string]bool)
	schema := []rpc.DatabaseSchemaColumn{
		{
			Name:        "id",
			Type:        rpc.ColumnTypeString,
			Description: "The ID of the response",
		},
		{
			Name:        "created_at",
			Type:        rpc.ColumnTypeDateTime,
			Description: "The creation date of the response (RFC3339 format)",
		},
	}

	for _, item := range form.Items {
		// Handle normal fields
		if item.QuestionItem != nil && item.QuestionItem.Question != nil {
			switch {
			case item.QuestionItem.Question.ChoiceQuestion != nil:
				title := item.Title
				_, alreadyExist := columnNameDedup[title]
				for alreadyExist {
					title += "_"
					_, alreadyExist = columnNameDedup[title]
				}
				description := strings.Builder{}
				description.WriteString("A choice question with the following choices: ")
				for i, choice := range item.QuestionItem.Question.ChoiceQuestion.Options {
					if choice != nil {
						if i > 0 {
							description.WriteString(", ")
						}
						description.WriteString(choice.Value)
					}
				}
				columnNameDedup[title] = true
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        title,
					Type:        rpc.ColumnTypeString,
					Description: description.String(),
				})
				mapIdColInfo[item.QuestionItem.Question.QuestionId] = col{colIndex, colTypeArray}
				colIndex++
			case item.QuestionItem.Question.TextQuestion != nil:
				title := item.Title
				_, alreadyExist := columnNameDedup[title]
				for alreadyExist {
					title += "_"
					_, alreadyExist = columnNameDedup[title]
				}
				columnNameDedup[title] = true
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        title,
					Type:        rpc.ColumnTypeString,
					Description: "A text question in the form",
				})
				mapIdColInfo[item.QuestionItem.Question.QuestionId] = col{colIndex, colTypeString}
				colIndex++
			case item.QuestionItem.Question.DateQuestion != nil:
				title := item.Title
				_, alreadyExist := columnNameDedup[title]
				for alreadyExist {
					title += "_"
					_, alreadyExist = columnNameDedup[title]
				}
				columnNameDedup[title] = true
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        title,
					Type:        rpc.ColumnTypeDateTime,
					Description: "A date question in the form",
				})
				mapIdColInfo[item.QuestionItem.Question.QuestionId] = col{colIndex, colTypeString}
				colIndex++
			case item.QuestionItem.Question.TimeQuestion != nil:
				title := item.Title
				_, alreadyExist := columnNameDedup[title]
				for alreadyExist {
					title += "_"
					_, alreadyExist = columnNameDedup[title]
				}
				columnNameDedup[title] = true
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        title,
					Type:        rpc.ColumnTypeTime,
					Description: "A time question in the form",
				})
				mapIdColInfo[item.QuestionItem.Question.QuestionId] = col{colIndex, colTypeString}
				colIndex++
			case item.QuestionItem.Question.ScaleQuestion != nil:
				title := item.Title
				_, alreadyExist := columnNameDedup[title]
				for alreadyExist {
					title += "_"
					_, alreadyExist = columnNameDedup[title]
				}
				columnNameDedup[title] = true
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name: title,
					Type: rpc.ColumnTypeFloat,
					Description: fmt.Sprintf("A scale question in the form with a minimum of %d (%s) and a maximum of %d (%s)", item.QuestionItem.Question.ScaleQuestion.Low, item.QuestionItem.Question.ScaleQuestion.LowLabel,
						item.QuestionItem.Question.ScaleQuestion.High, item.QuestionItem.Question.ScaleQuestion.HighLabel),
				})
				mapIdColInfo[item.QuestionItem.Question.QuestionId] = col{colIndex, colTypeNumber}
				colIndex++
			}
		} else if item.QuestionGroupItem != nil {
			fieldTitle := item.Title
			_, alreadyExist := columnNameDedup[fieldTitle]
			for alreadyExist {
				fieldTitle += "_"
				_, alreadyExist = columnNameDedup[fieldTitle]
			}
			columnNameDedup[fieldTitle] = true

			// Find the subfields
			for _, subfield := range item.QuestionGroupItem.Questions {
				// Handle only grid fields
				if subfield.RowQuestion != nil {
					title := fieldTitle + "_" + subfield.RowQuestion.Title
					_, alreadyExist := columnNameDedup[title]
					for alreadyExist {
						title += "_"
						_, alreadyExist = columnNameDedup[title]
					}
					columnNameDedup[title] = true

					schema = append(schema, rpc.DatabaseSchemaColumn{
						Name:        title,
						Type:        rpc.ColumnTypeString,
						Description: "A grid question in the form",
					})
					mapIdColInfo[subfield.QuestionId] = col{colIndex, colTypeArray}
					colIndex++
				}
			}

		}
	}

	return &google_formsTable{
			colInfo: mapIdColInfo,
			client:  client,
			formID:  formID,
		}, &rpc.DatabaseSchema{
			Columns:     schema,
			Description: tableDescription.String(),
		}, nil
}

type google_formsTable struct {
	colInfo map[string]col
	client  *forms.Service
	formID  string
}

type google_formsCursor struct {
	client     *forms.Service
	colInfo    map[string]col
	formID     string
	nextCursor string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *google_formsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the next page of responses
	req := t.client.Forms.Responses.List(t.formID)
	if t.nextCursor != "" {
		req.PageToken(t.nextCursor)
	}
	req.PageSize(5000)
	resp, err := req.Do()
	if err != nil {
		return nil, true, fmt.Errorf("failed to get responses: %w", err)
	}

	// Prepare the rows
	rows := make([][]interface{}, 0, len(resp.Responses))
	for _, response := range resp.Responses {
		row := make([]interface{}, len(t.colInfo)+2)
		row[0] = response.ResponseId
		row[1] = response.CreateTime
		for _, item := range response.Answers {
			if col, ok := t.colInfo[item.QuestionId]; ok {
				switch col.colType {
				case colTypeArray:
					if item.TextAnswers != nil {
						row[col.colIndex] = serializeArray(item.TextAnswers.Answers)
					}
				case colTypeString:
					if item.TextAnswers != nil {
						if len(item.TextAnswers.Answers) > 0 && item.TextAnswers.Answers[0] != nil {
							row[col.colIndex] = item.TextAnswers.Answers[0].Value
						}
					}
				case colTypeNumber:
					if item.TextAnswers != nil {
						if len(item.TextAnswers.Answers) > 0 && item.TextAnswers.Answers[0] != nil {
							// Parse the number
							res, err := strconv.ParseFloat(item.TextAnswers.Answers[0].Value, 64)
							if err != nil {
								log.Printf("Failed to parse number %v", item.TextAnswers.Answers[0].Value)
							} else {
								row[col.colIndex] = res
							}
						}
					}
				default:
					log.Printf("Unknown column type %v at %v for id %v", col.colType, col.colIndex, item.QuestionId)

				}
			}
		}
		rows = append(rows, row)
	}

	// Update the cursor
	if len(resp.Responses) > 0 {
		t.nextCursor = resp.NextPageToken
	}

	return rows, t.nextCursor == "" || len(resp.Responses) == 0, nil
}

func serializeArray(answer []*forms.TextAnswer) string {
	vals := []string{}
	for _, a := range answer {
		if a == nil {
			continue
		}
		vals = append(vals, a.Value)
	}
	serialized, err := json.Marshal(vals)
	if err != nil {
		log.Printf("Failed to serialize array %v", vals)
		return ""
	}
	return string(serialized)
}

// Create a new cursor that will be used to read rows
func (t *google_formsTable) CreateReader() rpc.ReaderInterface {
	return &google_formsCursor{
		client:  t.client,
		colInfo: t.colInfo,
		formID:  t.formID,
	}
}

// A destructor to clean up resources
func (t *google_formsTable) Close() error {
	return nil
}
