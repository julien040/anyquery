package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func (m *discordMod) invitesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	// Get a session from the token
	session, cache, err := m.getSession(token)
	if err != nil {
		return nil, nil, err
	}

	return &invitesTable{
			session: session,
			cache:   cache,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "guild_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The ID of the guild. In https://discord.com/channels/12345678/98765432, the guild ID is 12345678",
				},
				{
					Name:        "to_channel_id",
					Type:        rpc.ColumnTypeString,
					Description: "To which channel the invite is for",
				},
				{
					Name:        "to_channel_name",
					Type:        rpc.ColumnTypeString,
					Description: "To which channel (name) the invite is for",
				},
				{
					Name:        "created_by_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the user who created the invite",
				},
				{
					Name:        "created_by_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the user who created the invite",
				},
				{
					Name:        "invide_code",
					Type:        rpc.ColumnTypeString,
					Description: "The code of the invite",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the invite was created",
				},
				{
					Name:        "expires_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the invite expires (RFC3339 format)",
				},
				{
					Name:        "max_uses",
					Type:        rpc.ColumnTypeInt,
					Description: "The maximum number of uses the invite has",
				},
				{
					Name:        "uses",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of times the invite has been used",
				},
				{
					Name:        "max_age",
					Type:        rpc.ColumnTypeInt,
					Description: "The maximum age of the invite in seconds",
				},
				{
					Name:        "temporary",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the invite is temporary",
				},
				{
					Name:        "revoked",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the invite has been revoked",
				},
				{
					Name:        "unique",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the invite is unique",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type invitesTable struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from invitesTable, an offset, a cursor, etc.)
type invitesCursor struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// Create a new cursor that will be used to read rows
func (t *invitesTable) CreateReader() rpc.ReaderInterface {
	return &invitesCursor{
		session: t.session,
		cache:   t.cache,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *invitesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	guildID := constraints.GetColumnConstraint(0).GetStringValue()
	if guildID == "" {
		return nil, true, fmt.Errorf("guild_id is required")
	}

	// Try to get the invites from the cache
	cacheKey := fmt.Sprintf("invites:%s", guildID)
	rows, _, err := t.cache.Get(cacheKey)
	if err == nil {
		return rows, true, nil
	}

	// Get the invites from the API
	invites, err := t.session.GuildInvites(guildID)
	if err != nil {
		return nil, true, fmt.Errorf("error getting invites: %w", err)
	}

	// Prepare the rows
	rows = make([][]interface{}, 0, len(invites))
	for _, invite := range invites {
		expiresAt := interface{}(nil)
		if invite.ExpiresAt != nil {
			expiresAt = invite.ExpiresAt.Format(time.RFC3339)
		}
		var channelID, channelName string
		var createdByID, createdByName string
		if invite.Channel != nil {
			channelID = invite.Channel.ID
			channelName = invite.Channel.Name
		}
		if invite.Inviter != nil {
			createdByID = invite.Inviter.ID
			createdByName = invite.Inviter.Username
		}
		rows = append(rows, []interface{}{
			channelID,
			channelName,
			createdByID,
			createdByName,
			invite.Code,
			invite.CreatedAt.Format(time.RFC3339),
			expiresAt,
			invite.MaxUses,
			invite.Uses,
			invite.MaxAge,
			invite.Temporary,
			invite.Revoked,
			invite.Unique,
		})
	}

	// Cache the rows
	err = t.cache.Set(cacheKey, rows, nil, time.Hour)
	if err != nil {
		log.Printf("error while caching invites: %v", err)
	}

	return rows, true, nil
}

// A destructor to clean up resources
func (t *invitesTable) Close() error {
	return nil
}
