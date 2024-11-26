# Discord plugin

Query data from Discord servers using SQL.

```sql
-- Get all messages from a channel
SELECT * FROM discord_messages('channel_id');

-- Check which invites are the most popular
SELECT * FROM discord_invites('guild_id') ORDER BY uses DESC;

-- Get all members of a guild
SELECT * FROM discord_members('guild_id');

-- Write a message to a channel
INSERT INTO discord_messages('channel_id', 'content') VALUES ('channel_id', 'Hello, world!');

-- Delete recent messages from a channel that contain the word 'hello'
DELETE FROM discord_messages('channel_id') WHERE content LIKE '%hello%' AND created_at > datetime('now', '-14 days');
```

## Setup

1. Create a new Discord application at [https://discord.com/developers/applications](https://discord.com/developers/applications).
2. Go to the 'Bot' tab and click 'Add Bot'.
3. Reset the bot token and copy it. You will need it later.
4. Scroll down to configure the bot's permissions.
   1. Under 'Privileged Gateway Intents`, enable 'Server Members Intent' and 'Message Content Intent'.
   2. Under 'Bot permissions', enable 'Administrator'. If you know what you're doing, you can enable only the permissions you need.
5. Go to the 'OAuth2' tab and select 'bot' in the 'OAuth2 URL Generator` section. Select the `Administrator` scope.
6. Copy the generated link and open it in your browser. Add the bot to your server.
7. Congratulations! You can now use the plugin.

Install the discord plugin:

```bash
anyquery install discord
```

and paste the bot token when prompted.

### Guide on IDs

- `guild_id` is the ID of the server.
- `channel_id` is the ID of the channel.
- `message_id` is the ID of the message.
- `user_id` is the ID of the user.

You can find the `guild_id` by looking at the URL of the server. It is the first number after `discord.com/channels/`. For example, in `https://discord.com/channels/123456789012345678/987654321098765432`, the `guild_id` is `123456789012345678`.
The `channel_id` is the second number after `discord.com/channels/`. In the same example, the `channel_id` is `987654321098765432`.

Other IDs can be found by running queries like `SELECT * FROM discord_messages('channel_id')`.

Most tables require the `guild_id` or `channel_id` to be specified. You can either pass it as an argument to the table (e.g. `discord_messages('channel_id')`) or set it in the `WHERE` clause (e.g. `SELECT * FROM discord_messages WHERE channel_id = 'channel_id'`).

## Schema

### discord_messages

The `discord_messages` table contains messages from a channel. It supports `SELECT`, `INSERT`, and `DELETE` queries. Inserted messages will be sent under the bot's name. Also, the plugin is unable to delete messages older than 14 days.

```sql
-- Get all messages from a channel
SELECT * FROM discord_messages('channel_id');
SELECT * FROM discord_messages WHERE channel_id = 'channel_id';

-- Write a message to a channel
INSERT INTO discord_messages('channel_id', 'content') VALUES ('channel_id', 'Hello, world!');

-- Delete recent messages from a channel that contain the word 'hello'
DELETE FROM discord_messages('channel_id') WHERE content LIKE '%hello%' AND created_at > datetime('now', '-14 days');

-- Count the number of messages in a channel that contain the word 'hello'
SELECT COUNT(*) FROM discord_messages('channel_id') WHERE content LIKE '%hello%';

-- Get the number of messages per user in a channel
SELECT username, COUNT(*) FROM discord_messages('channel_id') GROUP BY username, user_id ORDER BY COUNT(*) DESC;
```

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | TEXT    |
| 1            | message_id  | TEXT    |
| 2            | content     | TEXT    |
| 3            | created_at  | TEXT    |
| 4            | edited_at   | TEXT    |
| 5            | user_id     | TEXT    |
| 6            | username    | TEXT    |
| 7            | pinned      | INTEGER |
| 8            | attachments | TEXT    |
| 9            | mentions    | TEXT    |
| 10           | reactions   | TEXT    |

### discord_channels

The `discord_channels` table contains channels from a server. It supports `SELECT`, `INSERT`, `UPDATE`, and `DELETE` queries.

```sql
-- Get all channels from a server from top to bottom
SELECT * FROM discord_channels('guild_id') ORDER BY position;

-- Create a new channel
INSERT INTO discord_channels(guild_id, name, type) VALUES ('guild_id', 'my-channel', 'GUILD_TEXT');

-- Update a channel's name
UPDATE discord_channels('guild_id') SET name = 'new-name' WHERE name = 'old-name';

-- Delete a channel
DELETE FROM discord_channels('guild_id') WHERE name = 'channel-name';
```

Valid values for the channel type are `GUILD_TEXT`, `DM`, `GUILD_VOICE`, `GROUP_DM`, `GUILD_CATEGORY`, `GUILD_NEWS`, `GUILD_STORE`, `GUILD_NEWS_THREAD`, `GUILD_PUBLIC_THREAD`, `GUILD_PRIVATE_THREAD`, `GUILD_STAGE_VOICE`, `GUILD_DIRECTORY`, `GUILD_FORUM`, and `GUILD_MEDIA`.

| Column index | Column name         | type    |
| ------------ | ------------------- | ------- |
| 0            | id                  | TEXT    |
| 1            | channel_id          | TEXT    |
| 2            | name                | TEXT    |
| 3            | topic               | TEXT    |
| 4            | type                | TEXT    |
| 5            | nsfw                | INTEGER |
| 6            | position            | INTEGER |
| 7            | bitrate             | INTEGER |
| 8            | user_limit          | INTEGER |
| 9            | rate_limit_per_user | TEXT    |

### discord_members

The `discord_members` list members of a server. It supports `SELECT`  and `DELETE` queries.

```sql
-- Get all members of a server
SELECT * FROM discord_members('guild_id');

-- Count the number of members in a server
SELECT COUNT(*) FROM discord_members('guild_id');

-- Kick a member from a server
DELETE FROM discord_members('guild_id') WHERE username = 'username';

```

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | id                 | TEXT    |
| 1            | user_id            | TEXT    |
| 2            | username           | TEXT    |
| 3            | discriminator      | TEXT    |
| 4            | email_verified     | INTEGER |
| 5            | bot                | INTEGER |
| 6            | joined_at          | TEXT    |
| 7            | nickname           | TEXT    |
| 8            | roles              | TEXT    |
| 9            | pending_membership | INTEGER |
| 10           | premium_since      | TEXT    |
| 11           | deaf               | INTEGER |
| 12           | muted              | INTEGER |

### discord_bans

The `discord_bans` table contains banned users from a server. It supports `SELECT`, `INSERT`, and `DELETE` queries.

```sql
-- Get all banned users from a server
SELECT * FROM discord_bans('guild_id');

-- Ban a user from a server
INSERT INTO discord_bans('guild_id', 'user_id') VALUES ('guild_id', 'user_id');

-- Unban a user from a server
DELETE FROM discord_bans('guild_id') WHERE user_id = 'user_id';
```

| Column index | Column name    | type    |
| ------------ | -------------- | ------- |
| 0            | id             | TEXT    |
| 1            | user_id        | TEXT    |
| 2            | username       | TEXT    |
| 3            | discriminator  | TEXT    |
| 4            | email_verified | INTEGER |
| 5            | bot            | INTEGER |
| 6            | ban_reason     | TEXT    |

### discord_roles

List roles of a server. It supports `SELECT` queries.

```sql
-- Get all roles of a server
SELECT * FROM discord_roles('guild_id');
```

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | TEXT    |
| 1            | name        | TEXT    |
| 2            | managed     | INTEGER |
| 3            | mentionable | INTEGER |
| 4            | hoist       | INTEGER |
| 5            | color       | TEXT    |
| 6            | position    | INTEGER |
| 7            | permissions | INTEGER |
| 8            | emoji       | TEXT    |

### discord_guilds

List all the guilds the bot is in. It supports `SELECT` queries. You can use to get the `guild_id` of a server.

```sql
-- Get all guilds the bot is in
SELECT * FROM discord_guilds;

-- Get the guild ID of a server
SELECT guild_id FROM discord_guilds WHERE name = 'server-name';
```

| Column index | Column name    | type    |
| ------------ | -------------- | ------- |
| 0            | guild_id       | TEXT    |
| 1            | name           | TEXT    |
| 2            | icon           | TEXT    |
| 3            | member_count   | INTEGER |
| 4            | presence_count | INTEGER |

### discord_invites

List all the invites of a server. It supports `SELECT` queries.

```sql
-- Get all invites of a server
SELECT * FROM discord_invites('guild_id');

-- Get the number of uses per invite
SELECT code, uses FROM discord_invites('guild_id') ORDER BY uses DESC;
```

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | to_channel_id   | TEXT    |
| 1            | to_channel_name | TEXT    |
| 2            | created_by_id   | TEXT    |
| 3            | created_by_name | TEXT    |
| 4            | invide_code     | TEXT    |
| 5            | created_at      | TEXT    |
| 6            | expires_at      | TEXT    |
| 7            | max_uses        | INTEGER |
| 8            | uses            | INTEGER |
| 9            | max_age         | INTEGER |
| 10           | temporary       | INTEGER |
| 11           | revoked         | INTEGER |
| 12           | unique          | INTEGER |
