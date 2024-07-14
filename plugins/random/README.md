# Random data plugin

A plugin to generate random data.

## Installation

```bash
anyquery install random
```

## Usage

Specify the columns you want to generate in the `SELECT` clause. The plugin will generate random data for each column and return the result. To see the list of available columns, refer to the [Tables schema](#tables-schema) section.

### Example

```bash
# Generate 100 random records and save them to a CSV file
anyquery -q "SELECT first_name, last_name, credit_card_number FROM random_people LIMIT 100" --csv > random.csv
# Generate 100 random records and save them to a JSON file
anyquery -q "SELECT username, email, phone_number, password FROM random_people LIMIT 100" --json > random.json
# Generate 320 passwords seperated by new lines
anyquery -q "SELECT password_lower_upper FROM random_passwords LIMIT 320" > passwords.txt
```

> **⚠️ Limitations**
> 
> - The generator is not trully random so don't use for security purposes.
>
> - Always set a LIMIT clause because the plugin will generate infinite records if you don't.
>
> - Because the plugin returns infinite records, you can't do joins with other tables. As a workaround, save the result to a table with `CREATE TABLE ... AS SELECT...` and then join with that table.

## Tables schema

### random_people

| Column index | Column name                   | type    |
| ------------ | ----------------------------- | ------- |
| 0            | id                            | INTEGER |
| 1            | first_name                    | TEXT    |
| 2            | last_name                     | TEXT    |
| 3            | gender                        | TEXT    |
| 4            | ssn                           | TEXT    |
| 5            | hobby                         | TEXT    |
| 6            | job_company                   | TEXT    |
| 7            | job_title                     | TEXT    |
| 8            | address                       | TEXT    |
| 9            | street                        | TEXT    |
| 10           | city                          | TEXT    |
| 11           | state                         | TEXT    |
| 12           | zip                           | TEXT    |
| 13           | country                       | TEXT    |
| 14           | latitude                      | REAL    |
| 15           | longitude                     | REAL    |
| 16           | phone                         | TEXT    |
| 17           | email                         | TEXT    |
| 18           | credit_card_number            | TEXT    |
| 19           | credit_card_type              | TEXT    |
| 20           | credit_card_expiration        | TEXT    |
| 21           | credit_card_cvv               | INTEGER |
| 22           | username                      | TEXT    |
| 23           | password                      | TEXT    |
| 24           | favorite_beer                 | TEXT    |
| 25           | car_maker                     | TEXT    |
| 26           | car_model                     | TEXT    |
| 27           | car_type                      | TEXT    |
| 28           | car_transmission              | TEXT    |
| 29           | car_fuel                      | TEXT    |
| 30           | favorite_fruit                | TEXT    |
| 31           | favorite_vegetable            | TEXT    |
| 32           | uuid                          | TEXT    |
| 33           | favorite_color                | TEXT    |
| 34           | favorite_color_hex            | TEXT    |
| 35           | pet_name                      | TEXT    |
| 36           | pet_type                      | TEXT    |
| 37           | language_spoken               | TEXT    |
| 38           | ƒavorite_programming_language | TEXT    |
| 39           | favorite_sport_player         | TEXT    |
| 40           | favorite_actor                | TEXT    |
| 41           | favorite_movie                | TEXT    |
| 42           | favorite_book                 | TEXT    |

### random_password

| Column index | Column name                  | type    |
| ------------ | ---------------------------- | ------- |
| 0            | id                           | INTEGER |
| 1            | username                     | TEXT    |
| 2            | password_lower               | TEXT    |
| 3            | password_lower_upper         | TEXT    |
| 4            | password_with_special        | TEXT    |
| 5            | password_with_special_number | TEXT    |

### random_internet

| Column index | Column name      | type    |
| ------------ | ---------------- | ------- |
| 0            | id               | INTEGER |
| 1            | url              | TEXT    |
| 2            | domain_name      | TEXT    |
| 3            | domain_extension | TEXT    |
| 4            | ipv4             | TEXT    |
| 5            | ipv6             | TEXT    |
| 6            | mac_address      | TEXT    |
| 7            | user_agent       | TEXT    |
