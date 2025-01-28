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
func (m *discordMod) membersCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	// Get a session from the token
	session, cache, err := m.getSession(token)
	if err != nil {
		return nil, nil, err
	}
	return &membersTable{
			session: session,
			cache:   cache,
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
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
					Description: "The ID of the member. It is the concatenation of the guild ID and the user ID",
				},
				{
					Name:        "user_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the user in the guild",
				},
				{
					Name:        "username",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the user in the guild",
				},
				{
					Name:        "discriminator",
					Type:        rpc.ColumnTypeString,
					Description: "The discriminator of the user in the guild. In the username#discriminator, the discriminator is the 4 digits after the #",
				},
				{
					Name:        "email_verified",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the user has verified their email",
				},
				{
					Name:        "bot",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the user is a bot",
				},
				{
					Name:        "joined_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the user joined the guild",
				},
				{
					Name:        "nickname",
					Type:        rpc.ColumnTypeString,
					Description: "The nickname of the user in the guild",
				},
				{
					Name:        "roles",
					Type:        rpc.ColumnTypeString,
					Description: "The roles of the user in the guild",
				},
				{
					Name:        "pending_membership",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the user has a pending membership",
				},
				{
					Name:        "premium_since",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the user started premium subscription",
				},
				{
					Name:        "deaf",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the user is deafened",
				},
				{
					Name:        "muted",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the user is muted",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type membersTable struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from membersTable, an offset, a cursor, etc.)
type membersCursor struct {
	session *discordgo.Session
	cache   *helper.Cache
	after   string
}

// Create a new cursor that will be used to read rows
func (t *membersTable) CreateReader() rpc.ReaderInterface {
	return &membersCursor{
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
func (t *membersCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the guild ID from the constraints
	guildID := constraints.GetColumnConstraint(0).GetStringValue()
	if guildID == "" {
		return nil, true, fmt.Errorf("guild_id must be set")
	}

	// Try to get the members from the cache
	cacheKey := fmt.Sprintf("members_%s_%s", guildID, t.after)
	rows, metadata, err := t.cache.Get(cacheKey)
	if err == nil {
		t.after = metadata["after"].(string)
		return rows, len(rows) < 1000, nil
	}

	// Get the members from the API
	members, err := t.session.GuildMembers(guildID, t.after, 1000)
	if err != nil {
		return nil, true, fmt.Errorf("error while fetching members: %w", err)
	}

	log.Printf("fetched %d members for cursor %s", len(members), t.after)

	// Prepare the rows
	rows = make([][]interface{}, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}
		var userID, username, discriminator, joinedAt string
		var emailVerified, bot bool
		var premiumSince interface{}

		if member.User != nil {
			userID = member.User.ID
			username = member.User.Username
			discriminator = member.User.Discriminator
			emailVerified = member.User.Verified
			bot = member.User.Bot
		}

		joinedAt = member.JoinedAt.Format(time.RFC3339)

		if member.PremiumSince != nil {
			premiumSince = member.PremiumSince.Format(time.RFC3339)
		}

		rows = append(rows, []interface{}{
			fmt.Sprintf("%s_%s", guildID, userID),
			userID,
			username,
			discriminator,
			emailVerified,
			bot,
			joinedAt,
			member.Nick,
			helper.Serialize(member.Roles),
			member.Pending,
			premiumSince,
			member.Deaf,
			member.Mute,
		})
	}

	// Get the next cursor
	t.after = members[len(members)-1].User.ID
	for _, member := range members {
		if member.User.ID > t.after {
			t.after = member.User.ID
		}
	}

	// Save the rows in the cache
	err = t.cache.Set(cacheKey, rows, map[string]interface{}{
		"after": t.after,
	}, 1*time.Hour)

	if err != nil {
		log.Printf("error while saving members in the cache: %v", err)
	}

	return rows, len(members) < 1000, nil
}

// A slice of primary keys to delete
func (t *membersTable) Delete(primaryKeys []interface{}) error {
	// Kick the members from the guild
	for _, primaryKey := range primaryKeys {
		splited := strings.Split(primaryKey.(string), "_")
		if len(splited) != 2 {
			return fmt.Errorf("invalid primary key")
		}

		err := t.session.GuildMemberDelete(splited[0], splited[1])
		if err != nil {
			return fmt.Errorf("error while kicking member %s: %w", primaryKey, err)
		}
	}

	return nil
}

// A destructor to clean up resources
func (t *membersTable) Close() error {
	return nil
}
