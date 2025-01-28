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
func (m *discordMod) rolesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	// Get a session from the token
	session, cache, err := m.getSession(token)
	if err != nil {
		return nil, nil, err
	}

	return &rolesTable{
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
					Description: "The ID of the guild. In https://discord.com/channels/12345678/98765432, the guild ID is 12345678",
				},
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the role",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the role",
				},
				{
					Name:        "managed",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the role is managed by an integration",
				},
				{
					Name:        "mentionable",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the role can be mentioned",
				},
				{
					Name:        "hoist",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the role is hoisted",
				},
				{
					Name:        "color",
					Type:        rpc.ColumnTypeString,
					Description: "The HEX color of the role",
				},
				{
					Name:        "position",
					Type:        rpc.ColumnTypeInt,
					Description: "The position of the role. The higher the position, the higher the role is in the hierarchy",
				},
				{
					Name:        "permissions",
					Type:        rpc.ColumnTypeInt,
					Description: "A bitfield of the permissions of the role",
				},
				{
					Name:        "emoji",
					Type:        rpc.ColumnTypeString,
					Description: "The emoji of the role",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type rolesTable struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from rolesTable, an offset, a cursor, etc.)
type rolesCursor struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// Create a new cursor that will be used to read rows
func (t *rolesTable) CreateReader() rpc.ReaderInterface {
	return &rolesCursor{
		session: t.session,
		cache:   t.cache,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *rolesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the guild ID from the constraints
	guildID := constraints.GetColumnConstraint(0).GetStringValue()
	if guildID == "" {
		return nil, true, fmt.Errorf("guild_id must be set")
	}

	// Get the roles from the cache if possible
	cacheKey := fmt.Sprintf("roles:%s", guildID)
	rows, _, err := t.cache.Get(cacheKey)
	if err == nil {
		return rows, true, nil
	}

	// Get the roles from the session
	roles, err := t.session.GuildRoles(guildID)
	if err != nil {
		return nil, true, err
	}

	// Convert the roles to rows
	rows = make([][]interface{}, 0, len(roles))
	for _, role := range roles {
		rows = append(rows, []interface{}{
			role.ID,
			role.Name,
			role.Managed,
			role.Mentionable,
			role.Hoist,
			fmt.Sprintf("#%06x", role.Color),
			role.Position,
			role.Permissions,
			role.UnicodeEmoji,
		})
	}

	// Cache the roles
	err = t.cache.Set(cacheKey, rows, nil, time.Hour)
	if err != nil {
		log.Printf("error while caching roles: %v", err)
	}

	return rows, true, nil
}

// A destructor to clean up resources
func (t *rolesTable) Close() error {
	return nil
}
