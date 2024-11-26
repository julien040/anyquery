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
func (m *discordMod) guildsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	// Get a session from the token
	session, cache, err := m.getSession(token)
	if err != nil {
		return nil, nil, err
	}

	return &guildsTable{
			session: session,
			cache:   cache,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "guild_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "icon",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "member_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "presence_count",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type guildsTable struct {
	session *discordgo.Session
	cache   *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from guildsTable, an offset, a cursor, etc.)
type guildsCursor struct {
	session *discordgo.Session
	cache   *helper.Cache
	after   string
}

// Create a new cursor that will be used to read rows
func (t *guildsTable) CreateReader() rpc.ReaderInterface {
	return &guildsCursor{
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
func (t *guildsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	// Try to get the guilds from the cache
	cacheKey := fmt.Sprintf("guilds:%s", t.after)
	rows, metadata, err := t.cache.Get(cacheKey)
	if err == nil {
		t.after = metadata["after"].(string)
		return rows, len(rows) < 200, nil
	}

	// Get the guilds from the session
	guilds, err := t.session.UserGuilds(100, "", t.after, true)
	if err != nil {
		return nil, true, fmt.Errorf("error getting guilds: %w", err)
	}

	// Prepare the rows
	rows = make([][]interface{}, 0, len(guilds))
	for _, guild := range guilds {
		rows = append(rows, []interface{}{
			guild.ID,
			guild.Name,
			guild.Icon,
			guild.ApproximateMemberCount,
			guild.ApproximatePresenceCount,
		})
	}

	// Get the next cursor
	t.after = guilds[len(guilds)-1].ID
	for _, guild := range guilds {
		if guild.ID > t.after {
			t.after = guild.ID
		}
	}

	// Cache the rows
	err = t.cache.Set(cacheKey, rows, map[string]interface{}{
		"after": t.after,
	}, time.Hour)

	if err != nil {
		log.Printf("error while caching guilds: %v", err)
	}

	return rows, len(rows) < 200, nil

}

// A slice of rows to insert
func (t *guildsTable) Insert(rows [][]interface{}) error {
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
func (t *guildsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *guildsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *guildsTable) Close() error {
	return nil
}
