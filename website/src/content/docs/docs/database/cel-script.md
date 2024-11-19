---
title: Import filtering (CEL scripts)
description: To avoid importing all tables of a database, you can filter them using CEL scripts. Learn how to use CEL scripts to filter tables and views.
---

The Common Expression Language (CEL) allows us to provide a simple language, similar to C or JavaScript, to filter tables and views when importing them from a database. This way, you can avoid importing all tables and views and only import the ones you need.

The script is asked when you create a connection to a database. Note that SQLite import does not support filtering.

## Syntax

```javascript
// Filter tables that start with 'my_'
table.name.startsWith("my_") && table.type == "TABLE"

// Filter views that match a regular expression
table.name.matches("my_table.*") && table.type == "VIEW"
```

As you can see, the CEL script is a simple expression that evaluates to `true` or `false`. If the expression evaluates to `true`, the table or view is imported. Otherwise, it is skipped. The syntax is similar to most programming languages.

The script is run for each table and view in the database. The following constants are available in the script:

```go
{
    table: {
        schema: string, // The schema of the table (e.g., 'public', 'information_schema')
        name: string, // The name of the table (e.g., 'my_table')
        type: string, // The type of the table (e.g., 'BASE TABLE', 'VIEW')
        owner: string, // The owner of the table (e.g., 'postgres')
    },
    allTables: []table, // All tables and views in the database
}
```

You can access these constants using the dot notation. For example, to get the table name owner of the ith table, you can use `allTables[i].owner`.

### Types

While Anyquery only uses the `string` type, CEL supports the following types:

- `bool`: Boolean values (`true` or `false`)
- `int`: Integer values (e.g., `1`, `2`, `3`)
- `double`: Floating-point values (e.g., `1.0`, `2.0`, `3.0`)
- `string`: String values (e.g., `"hello"`, `"world"`)
- `bytes`: Byte values (e.g., `b"hello"`, `b"world"`)
- `list(T)`: List of values of type `T` (e.g., `[1, 2, 3]`, `["hello", "world"]`)
- `map(K, V)`: Map of keys of type `K` and values of type `V` (e.g., `{"key": "value"}`, `{1: 2}`)

Most of the times, you can cast a value to the desired type using the function `type(value)`. For example, to cast a string to an integer, you can use `int("1")`.

### Functions and operators

CEL supports a few functions to manipulate values. The following functions are available:

- `.startsWith(string)`: Check if a string starts with another string
- `.endsWith(string)`: Check if a string ends with another string
- `.contains(string)`: Check if a string contains another string
- `.matches(string)`: Check if a string matches a regular expression
- `size(array)`: Get the size of an array
- `type(value)`: Get the type of a value

To check if a value is in an array, you can use the `in` operator. For example, to filter by a predefined list of tables, you can use the following script:

```javascript
table.name in ["my_table", "another_table"]
```

CEL also supports the following operators:

- `==`: Check if two values are equal
- `!=`: Check if two values are different
- `&&`: Logical AND
- `||`: Logical OR
- `!`: Logical NOT
- `+`: Addition
- `-`: Subtraction
- `*`: Multiplication
- `/`: Division
- `%`: Remainder
- `?:`: Ternary operator (e.g., `condition ? return_if_true : return_if_false`)
- `in`: Check if a value is in an array

### Examples

Here a few examples of CEL scripts:

#### Filter tables that start with 'my_'

```javascript
table.name.startsWith("my_") && table.type == "TABLE"
```

#### Filter views that match a regular expression

```javascript
table.name.matches("my_table.*") && table.type == "VIEW"
```

#### Filter tables that are owned by a specific user

```javascript
table.owner == "postgres"
```

#### Filter tables that are in a specific schema for PostgreSQL

```javascript
// For PostgreSQL, public references all the tables in the database (without the introspection tables)
table.schema == "public"
// For MySQL, the schema is the database name
table.schema == "my_database"
```

#### Filter tables that are not in a specific schema

```javascript
table.schema != "information_schema"
```

#### Filter tables that are listed in a predefined list

```javascript
table.name in ["my_table", "another_table"]
```

#### Exclude tables that are listed in a predefined list

```javascript
!(table.name in ["my_table", "another_table"])
```

#### Filter tables that are not views

```javascript
table.type != "VIEW"
```
