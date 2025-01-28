package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
	"github.com/vishalkuo/bimap"
)

// Map an enum type to its string representation for the channel type
var channelType = bimap.NewBiMapFromMap(map[discordgo.ChannelType]string{
	discordgo.ChannelTypeGuildText:          "GUILD_TEXT",
	discordgo.ChannelTypeDM:                 "DM",
	discordgo.ChannelTypeGuildVoice:         "GUILD_VOICE",
	discordgo.ChannelTypeGroupDM:            "GROUP_DM",
	discordgo.ChannelTypeGuildCategory:      "GUILD_CATEGORY",
	discordgo.ChannelTypeGuildNews:          "GUILD_NEWS",
	discordgo.ChannelTypeGuildStore:         "GUILD_STORE",
	discordgo.ChannelTypeGuildNewsThread:    "GUILD_NEWS_THREAD",
	discordgo.ChannelTypeGuildPublicThread:  "GUILD_PUBLIC_THREAD",
	discordgo.ChannelTypeGuildPrivateThread: "GUILD_PRIVATE_THREAD",
	discordgo.ChannelTypeGuildStageVoice:    "GUILD_STAGE_VOICE",
	discordgo.ChannelTypeGuildDirectory:     "GUILD_DIRECTORY",
	discordgo.ChannelTypeGuildForum:         "GUILD_FORUM",
	discordgo.ChannelTypeGuildMedia:         "GUILD_MEDIA",
})

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func (m *discordMod) channelsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	// Get a session from the token
	session, cache, err := m.getSession(token)
	if err != nil {
		return nil, nil, err
	}

	return &channelsTable{
			session: session,
			cache:   cache,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: true,
			HandlesDelete: true,
			HandleOffset:  false,
			PrimaryKey:    1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "guild_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The ID of the guild. In https://discord.com/channels/12345678/98765432, the guild ID is 12345678",
				},
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the channel. It is the concatenation of the guild ID and the channel ID",
				},
				{
					Name:        "channel_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the channel",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the channel",
				},
				{
					Name:        "topic",
					Type:        rpc.ColumnTypeString,
					Description: "The topic of the channel",
				},
				{
					Name:        "type",
					Type:        rpc.ColumnTypeString,
					Description: "The type of the channel. One of GUILD_TEXT, DM, GUILD_VOICE, GROUP_DM, GUILD_CATEGORY, GUILD_NEWS, GUILD_STORE, GUILD_NEWS_THREAD, GUILD_PUBLIC_THREAD, GUILD_PRIVATE_THREAD, GUILD_STAGE_VOICE, GUILD_DIRECTORY, GUILD_FORUM, and GUILD_MEDIA.",
				},
				{
					Name:        "nsfw",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the channel is NSFW",
				},
				{
					Name:        "position",
					Type:        rpc.ColumnTypeInt,
					Description: "The position of the channel in the list of channels",
				},
				{
					Name:        "bitrate",
					Type:        rpc.ColumnTypeInt,
					Description: "The bitrate of the channel for voice channels",
				},
				{
					Name:        "user_limit",
					Type:        rpc.ColumnTypeInt,
					Description: "The max number of users that can be in the channel at the same time",
				},
				{
					Name:        "rate_limit_per_user",
					Type:        rpc.ColumnTypeString,
					Description: "How many seconds a user has to wait before sending another message in the channel",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type channelsTable struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from channelsTable, an offset, a cursor, etc.)
type channelsCursor struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// Create a new cursor that will be used to read rows
func (t *channelsTable) CreateReader() rpc.ReaderInterface {
	return &channelsCursor{
		session: t.session,
		cache:   t.cache,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *channelsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the guildID from the constraints
	guildID := constraints.GetColumnConstraint(0).GetStringValue()
	if guildID == "" {
		return nil, true, fmt.Errorf("guildID must be set")
	}

	// Try to get the channels from the cache
	cacheKey := fmt.Sprintf("channels:%s", guildID)
	rows, _, err := t.cache.Get(cacheKey)
	if err == nil {
		return rows, true, nil
	}

	// Otherwise, get the channels from the API
	channels, err := t.session.GuildChannels(guildID)
	if err != nil {
		return nil, true, fmt.Errorf("error getting channels from API: %w", err)
	}

	// Create the rows
	rows = make([][]interface{}, 0, len(channels))
	for _, channel := range channels {

		cType, _ := channelType.Get(channel.Type)

		rows = append(rows, []interface{}{
			fmt.Sprintf("%s_%s", guildID, channel.ID),
			channel.ID,
			channel.Name,
			channel.Topic,
			cType,
			channel.NSFW,
			channel.Position,
			channel.Bitrate,
			channel.UserLimit,
			channel.RateLimitPerUser,
		})
	}

	// Save the channels in the cache
	err = t.cache.Set(cacheKey, rows, nil, 1*time.Hour)
	if err != nil {
		log.Printf("error saving channels in cache: %v", err)
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *channelsTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		guildID := ""
		reqInsert := discordgo.GuildChannelCreateData{}
		if val, ok := row[0].(string); ok {
			guildID = val
		}
		if val, ok := row[3].(string); ok {
			reqInsert.Name = val
		}
		if val, ok := row[4].(string); ok {
			reqInsert.Topic = val
		}

		cType, ok := channelType.GetInverse(row[5].(string))
		if ok {
			reqInsert.Type = cType
		}

		if val, ok := row[6].(int64); ok {
			reqInsert.NSFW = val == 1
		}

		if val, ok := row[7].(int64); ok {
			reqInsert.Position = int(val)
		}

		if val, ok := row[8].(int64); ok {
			reqInsert.Bitrate = int(val)
		}

		if val, ok := row[9].(int64); ok {
			reqInsert.UserLimit = int(val)
		}

		if val, ok := row[10].(int64); ok {
			reqInsert.RateLimitPerUser = int(val)
		}

		if guildID == "" {
			return fmt.Errorf("guildID must be set")
		}

		// Create the channel
		_, err := t.session.GuildChannelCreateComplex(guildID, reqInsert, auditLogReasonCreated)
		if err != nil {
			return fmt.Errorf("error creating channel: %w", err)
		}

		// Clear the cache for the guild
		err = t.cache.Delete(fmt.Sprintf("channels:%s", guildID))
		if err != nil {
			log.Printf("Error clearing cache for guild %s: %v", guildID, err)
		}

	}

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *channelsTable) Update(rows [][]interface{}) error {
	for _, row := range rows {
		actualRow := row[1:]
		primaryKey := row[0].(string)
		splitted := strings.Split(primaryKey, "_")
		guildID := splitted[0]
		channelID := splitted[1]

		reqUpdate := &discordgo.ChannelEdit{}

		if val, ok := actualRow[3].(string); ok {
			reqUpdate.Name = val
		}

		if val, ok := actualRow[4].(string); ok {
			reqUpdate.Topic = val
		}

		// Can't update the type of a channel

		if val, ok := actualRow[6].(int64); ok {
			res := false
			if val == 1 {
				res = true
			}
			reqUpdate.NSFW = &res
		}

		if val, ok := actualRow[7].(int64); ok {
			pos := int(val)
			reqUpdate.Position = &pos
		}

		if val, ok := actualRow[8].(int64); ok {
			reqUpdate.Bitrate = int(val)
		}

		if val, ok := actualRow[9].(int64); ok {
			reqUpdate.UserLimit = int(val)
		}

		if val, ok := actualRow[10].(int64); ok {
			rateLimit := int(val)
			reqUpdate.RateLimitPerUser = &rateLimit
		}

		_, err := t.session.ChannelEditComplex(channelID, reqUpdate)
		if err != nil {
			return fmt.Errorf("error updating channel: %w", err)
		}

		// Clear the cache for the
		err = t.cache.Delete(fmt.Sprintf("channels:%s", guildID))
		if err != nil {
			log.Printf("Error clearing cache for channel %s: %v", channelID, err)
		}

	}

	return nil
}

// A slice of primary keys to delete
func (t *channelsTable) Delete(primaryKeys []interface{}) error {
	for _, pk := range primaryKeys {
		splitted := strings.Split(pk.(string), "_")
		if len(splitted) != 2 {
			return fmt.Errorf("invalid primary key %s", pk)
		}

		guildId := splitted[0]
		channelID := splitted[1]
		_, err := t.session.ChannelDelete(channelID, auditLogReasonDeleted)
		if err != nil {
			return fmt.Errorf("error deleting channel %s: %w", channelID, err)
		}

		// Clear the cache for the channel
		err = t.cache.Delete(fmt.Sprintf("channels:%s", guildId))
		if err != nil {
			log.Printf("Error clearing cache for channel %s: %v", channelID, err)
		}
	}

	return nil
}

// A destructor to clean up resources
func (t *channelsTable) Close() error {
	return nil
}
