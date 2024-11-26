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
				},
				{
					Name: "to_channel_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "to_channel_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_by_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_by_name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "invide_code",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "expires_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "max_uses",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "uses",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "max_age",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "temporary",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "revoked",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "unique",
					Type: rpc.ColumnTypeBool,
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

// A slice of rows to insert
func (t *invitesTable) Insert(rows [][]interface{}) error {
	// Example: insert the rows in a database
	// for _, row := range rows {
	// 	err := db.Insert(row[0], row[1], row[2])
	// 	if err != nil {
	// 		return err
	// 	}
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *invitesTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *invitesTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *invitesTable) Close() error {
	return nil
}
