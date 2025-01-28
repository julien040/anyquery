package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func (m *discordMod) messagesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	// Get a session from the token
	session, cache, err := m.getSession(token)
	if err != nil {
		return nil, nil, err
	}

	return &messageTable{
			session: session,
			cache:   cache,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: false,
			HandlesDelete: true,
			HandleOffset:  false,
			PrimaryKey:    1,
			BufferInsert:  0,
			BufferDelete:  100,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "channel_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The ID of the channel. In https://discord.com/channels/12345678/98765432, the channel ID is 98765432",
				},
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the message. It is the concatenation of the channel ID and the message ID",
				},
				{
					Name:        "message_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the message",
				},
				{
					Name:        "content",
					Type:        rpc.ColumnTypeString,
					Description: "The content of the message",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeString,
					Description: "The creation date of the message",
				},
				{
					Name:        "edited_at",
					Type:        rpc.ColumnTypeString,
					Description: "The edition date of the message",
				},
				{
					Name:        "user_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the user who sent the message",
				},
				{
					Name:        "username",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the user who sent the message",
				},
				{
					Name:        "pinned",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the message is pinned",
				},
				{
					Name:        "attachments",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON array of attachments. Fields for each attachment are: content_type (MIME type), filename, height, id, proxy_url, size, url, width",
				},
				{
					Name:        "mentions",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON array of mentions. Fields for each mention are: id, username, discriminator, avatar, bot",
				},
				{
					Name:        "reactions",
					Type:        rpc.ColumnTypeString,
					Description: "A JSON array of reactions. Fields for each reaction are: count, emoji {name, id, animated}, me",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type messageTable struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from discordTable, an offset, a cursor, etc.)
type messageCursor struct {
	session *discordgo.Session
	cache   *helper.Cache
	cursor  string
}

// Create a new cursor that will be used to read rows
func (t *messageTable) CreateReader() rpc.ReaderInterface {
	return &messageCursor{
		session: t.session,
		cache:   t.cache,
		cursor:  firstSnowflake,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *messageCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	channelID := constraints.GetColumnConstraint(0).GetStringValue()
	if channelID == "" {
		return nil, true, fmt.Errorf("channel_id must be set. Pass it as a parameter or use the 'channel_id = ?' constraint")
	}

	var rows [][]interface{}
	var metadata map[string]interface{}
	var err error

	// Try to get the messages from the cache
	cacheKey := fmt.Sprintf("messages_%s_%s", channelID, t.cursor)
	rows, metadata, err = t.cache.Get(cacheKey)
	if err == nil {
		t.cursor = metadata["cursor"].(string)
		return rows, t.cursor == "" || len(rows) < 100, nil
	}

	messages, err := t.session.ChannelMessages(channelID, 100, "", t.cursor, "")
	if err != nil {
		return nil, true, err
	}

	rows = make([][]interface{}, 0, len(messages))
	for _, message := range messages {
		editedTimestamp := interface{}(nil)
		if message.EditedTimestamp != nil {
			editedTimestamp = message.EditedTimestamp.Format(time.RFC3339)
		}

		authorID := interface{}(nil)
		username := interface{}(nil)

		if message.Author != nil {
			authorID = message.Author.ID
			username = message.Author.Username
		}

		// To ensure a SQL NULL value is returned when the slice is empty
		attachments := interface{}(nil)
		mentions := interface{}(nil)
		reactions := interface{}(nil)

		if message.Attachments != nil && len(message.Attachments) > 0 {
			attachments = helper.Serialize(message.Attachments)
		}

		if message.Mentions != nil && len(message.Mentions) > 0 {
			mentions = helper.Serialize(message.Mentions)
		}

		if message.Reactions != nil && len(message.Reactions) > 0 {
			reactions = helper.Serialize(message.Reactions)
		}

		rows = append(rows, []interface{}{
			fmt.Sprintf("%s_%s", channelID, message.ID),
			message.ID,
			message.Content,
			message.Timestamp.Format(time.RFC3339),
			editedTimestamp,
			authorID,
			username,
			message.Pinned,
			attachments,
			mentions,
			reactions,
		})
	}

	// Get the highest ID to use as the cursor
	highestID := rows[0][1].(string)
	for i := 1; i < len(rows); i++ {
		highestID = max(highestID, rows[i][1].(string))
	}

	t.cursor = highestID

	// Save the messages in the cache only if the page is full
	if len(messages) == 100 {
		err = t.cache.Set(cacheKey, rows, map[string]interface{}{
			"cursor": t.cursor,
		}, 1*time.Hour)

		if err != nil {
			log.Printf("Error saving messages in cache: %v", err)
		}
	}

	return rows, len(messages) < 100, nil
}

// A slice of rows to insert
func (t *messageTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		channelID, ok := row[0].(string)
		if !ok {
			return fmt.Errorf("channel_id must be set")
		}

		content, ok := row[3].(string)
		if !ok || content == "" {
			return fmt.Errorf("content must be set")
		}

		data := &discordgo.MessageSend{
			AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{
				discordgo.AllowedMentionTypeUsers,
				discordgo.AllowedMentionTypeEveryone,
				discordgo.AllowedMentionTypeRoles},
			},
			Content: content,
		}

		_, err := t.session.ChannelMessageSendComplex(channelID, data, auditLogReasonCreated)
		if err != nil {
			return fmt.Errorf("error sending message %s: %v", content, err)
		}

		// Clear the cache for the channel
		err = t.cache.ClearWithPrefix(fmt.Sprintf("messages_%s_", channelID))
		if err != nil {
			log.Printf("Error clearing cache for channel %s: %v", channelID, err)
		}

	}

	return nil
}

// A slice of primary keys to delete
func (t *messageTable) Delete(primaryKeys []interface{}) error {
	// Bulk delete is only supported with at least 2 primary keys
	// and the API supports delete per channel
	//
	// Therefore, we group the primary keys by channel
	// and then check if we should run a bulk delete or not
	grouped := make(map[string][]string)
	for _, pk := range primaryKeys {
		splitted := strings.Split(pk.(string), "_")
		if len(splitted) != 2 {
			return fmt.Errorf("invalid primary key %s", pk)
		}

		channelID := splitted[0]
		messageID := splitted[1]
		grouped[channelID] = append(grouped[channelID], messageID)
	}

	for channelID, messageIDs := range grouped {
		if len(messageIDs) == 1 {
			err := t.session.ChannelMessageDelete(channelID, messageIDs[0], auditLogReasonDeleted)
			if err != nil {
				return fmt.Errorf("error deleting message %s: %v", messageIDs[0], err)
			}
		} else if len(messageIDs) > 1 {
			err := t.session.ChannelMessagesBulkDelete(channelID, messageIDs, auditLogReasonDeleted)
			if err != nil {
				return fmt.Errorf("error deleting messages %v: %v", messageIDs, err)
			}
		} else {
			return fmt.Errorf("invalid primary key for channel %s", channelID)
		}

		// Clear the cache for the channel
		err := t.cache.ClearWithPrefix(fmt.Sprintf("messages_%s_", channelID))
		if err != nil {
			log.Printf("Error clearing cache for channel %s: %v", channelID, err)
		}
	}

	return nil
}

// A destructor to clean up resources
func (t *messageTable) Close() error {
	return nil
}
