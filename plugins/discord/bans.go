package main

import (
	"fmt"
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
func (m *discordMod) bansCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	// Get a session from the token
	session, cache, err := m.getSession(token)
	if err != nil {
		return nil, nil, err
	}

	return &bansTable{
			session: session,
			cache:   cache,
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "guild_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "user_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "username",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "discriminator",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "email_verified",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "bot",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "ban_reason",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type bansTable struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from bansTable, an offset, a cursor, etc.)
type bansCursor struct {
	session *discordgo.Session
	cache   *helper.Cache
	after   string
}

// Create a new cursor that will be used to read rows
func (t *bansTable) CreateReader() rpc.ReaderInterface {
	return &bansCursor{
		session: t.session,
		cache:   t.cache,
		after:   firstSnowflake,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *bansCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the guild ID from the constraints
	guildID := constraints.GetColumnConstraint(0).GetStringValue()
	if guildID == "" {
		return nil, true, fmt.Errorf("guild_id must be set")
	}

	// Try to get the bans from the cache
	cacheKey := fmt.Sprintf("bans_%s_%s", guildID, t.after)
	rows, metadata, err := t.cache.Get(cacheKey)
	if err == nil {
		t.after = metadata["after"].(string)
		return rows, len(rows) < 1000, nil
	}

	// Get the bans from the API
	bans, err := t.session.GuildBans(guildID, 1000, "", t.after)
	if err != nil {
		return nil, true, fmt.Errorf("error while fetching bans: %w", err)
	}

	// Get the next cursor
	t.after = bans[len(bans)-1].User.ID
	for _, ban := range bans {
		if ban.User.ID > t.after {
			t.after = ban.User.ID
		}
	}

	// Create the rows
	rows = make([][]interface{}, 0, len(bans))
	for _, ban := range bans {
		if ban.User == nil {
			continue
		}
		rows = append(rows, []interface{}{
			fmt.Sprintf("%s_%s", guildID, ban.User.ID),
			ban.User.ID,
			ban.User.Username,
			ban.User.Discriminator,
			ban.User.Verified,
			ban.User.Bot,
			ban.Reason,
		})
	}

	// Store the rows in the cache
	err = t.cache.Set(cacheKey, rows, map[string]interface{}{
		"after": t.after,
	}, time.Hour)

	return rows, len(rows) < 1000, err
}

// A slice of rows to insert
func (t *bansTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		data := &discordgo.GuildBanAdd{}
		data.User = &discordgo.User{}
		if val, ok := row[1].(string); ok {
			data.User.ID = val
		}
		var reason, guildID string
		if val, ok := row[6].(string); ok {
			reason = val
		}
		if val, ok := row[0].(string); ok {
			guildID = val
		}

		if guildID == "" {
			return fmt.Errorf("guild_id must be set")
		}

		if data.User.ID == "" {
			return fmt.Errorf("user_id must be set")
		}

		err := t.session.GuildBanCreateWithReason(row[0].(string), data.User.ID, reason, 0, auditLogReasonCreated)
		if err != nil {
			return fmt.Errorf("error while creating ban: %w", err)
		}
	}

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *bansTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *bansTable) Delete(primaryKeys []interface{}) error {
	for _, primaryKey := range primaryKeys {
		splited := strings.Split(primaryKey.(string), "_")
		if len(splited) != 2 {
			return fmt.Errorf("invalid primary key")
		}

		err := t.session.GuildBanDelete(splited[0], splited[1])
		if err != nil {
			return fmt.Errorf("error while deleting ban %s: %w", primaryKey, err)
		}
	}

	return nil
}

// A destructor to clean up resources
func (t *bansTable) Close() error {
	return nil
}
