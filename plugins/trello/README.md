# Trello

This plugin allows you to run SQL queries on your Trello boards and cards.

## Usage

The plugin supports all the basic SQL operations. Here are some examples:

```sql
-- List all your Trello boards (invited, created, or public)
SELECT * FROM trello_boards;

-- List all the cards of a board
-- You can find the board ID using the `trello_boards` table
SELECT * FROM trello_cards('board_id');

-- Group the cards by list
SELECT list_id, count(id) as number_of_cards FROM trello_cards('board_id') GROUP BY list_id;

-- List all lists of a board
SELECT * FROM trello_lists('board_id');

-- Insert a card in a list
-- You can find the list ID using the `trello_lists` table
INSERT INTO trello_cards (name, description, due_at, list_id) VALUES ('My card', 'My description', '2021-12-31', 'list_id');

-- Update a card
UPDATE trello_cards SET name = 'My new card name' WHERE name = 'My card' and board_id = 'board_id';

-- Delete a card
DELETE FROM trello_cards WHERE name = 'My card' and board_id = 'board_id';

-- Move a card to another list
-- You can find the list ID using the `trello_lists` table
UPDATE trello_cards SET list_id = 'list_id' WHERE name = 'My card' and board_id = 'board_id';
```

## Installation

You need [Anyquery](https://anyquery.dev/docs/#installation) installed on your machine to run this plugin.

Then, install the plugin with the following command:

```bash
anyquery install trello
```

At some point, you will be asked to provide your Trello API key and token. You can find them by creating an application.

### Find your Trello API key and token

1. Go to [Trello's Power-Ups page](https://trello.com/power-ups/admin/new)
2. Fill in the form with the following information:
   1. New Power-Up or Integration: Whatever you want
   2. Workspace: The workspace you want the plugin to have access to
   3. Iframe Connector URL: Leave it empty
   4. Email: Whatever you want
   5. Support Contact: Whatever you want
   6. Author Name: Whatever you want
3. Click on the `Create` button
4. Click on the `Generate a new API key` button and again on the `Generate a new API key` button in the popup
5. Copy the `API key` and paste it when asked by the plugin
6. On the right side of the page, Token is written in blue. Click on it, click on `Allow` and copy the token. Paste it when asked by the plugin
7. The plugin will ask you to provide the userID. Go to your profile page and copy the userID from the URL. Paste it when asked by the plugin

## Table reference

### trello_boards

List all your Trello boards (invited, created, or public). Use their ID to query the cards and lists.

| Column index | Column name      | type    |
| ------------ | ---------------- | ------- |
| 0            | id               | TEXT    |
| 1            | name             | TEXT    |
| 2            | description      | TEXT    |
| 3            | url              | TEXT    |
| 4            | pinned           | INTEGER |
| 5            | starred          | INTEGER |
| 6            | subscribed       | INTEGER |
| 7            | closed_at        | TEXT    |
| 8            | last_viewed_at   | TEXT    |
| 9            | last_activity_at | TEXT    |

### trello_cards

List all the cards of a board. To pass the board ID, pass the board ID as an argument to the table or write `WHERE board_id = 'board_id'`. In case of an insert, update, or delete, you need to provide the board ID too (e.g., `INSERT INTO ... (..., board_id) VALUES (..., 'board_id')`, `WHERE board_id = 'board_id'`).

| Column index | Column name         | type    |
| ------------ | ------------------- | ------- |
| 0            | list_id             | TEXT    |
| 1            | id                  | TEXT    |
| 2            | name                | TEXT    |
| 3            | description         | TEXT    |
| 4            | position            | INTEGER |
| 5            | url                 | TEXT    |
| 6            | start_at            | TEXT    |
| 7            | due_at              | TEXT    |
| 8            | due_completed       | TEXT    |
| 9            | due_reminder        | TEXT    |
| 10           | comments_count      | INTEGER |
| 11           | votes_count         | INTEGER |
| 12           | checklists_count    | INTEGER |
| 13           | checked_items_count | INTEGER |
| 14           | attachments_count   | INTEGER |
| 15           | labels              | TEXT    |
| 16           | subscribed          | INTEGER |
| 17           | location            | TEXT    |
| 18           | member_ids          | TEXT    |
| 19           | label_ids           | TEXT    |

### trello_lists

List all lists of a board. To pass the board ID, pass the board ID as an argument to the table or write `WHERE board_id = 'board_id'`.

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | TEXT    |
| 1            | name        | TEXT    |
| 2            | color       | TEXT    |
| 3            | subscribed  | INTEGER |
| 4            | position    | REAL    |
