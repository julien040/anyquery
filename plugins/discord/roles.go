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
				},
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "managed",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "mentionable",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "hoist",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "color",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "position",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "permissions",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "emoji",
					Type: rpc.ColumnTypeString,
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

// A slice of rows to insert
func (t *rolesTable) Insert(rows [][]interface{}) error {
	// Example: insert the rows in a database
	// for _, row := range rows {
	// 	err := db.Insert(row[0], row[1], row[2])
	// 	if err != nil {
	// 		return err
	// 	}
	return fmt.Errorf("insert are not yet supported")
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *rolesTable) Update(rows [][]interface{}) error {
	return fmt.Errorf("update are not yet supported")
}

// A slice of primary keys to delete
func (t *rolesTable) Delete(primaryKeys []interface{}) error {
	return fmt.Errorf("delete are not yet supported")
}

// A destructor to clean up resources
func (t *rolesTable) Close() error {
	return nil
}
