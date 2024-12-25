---
title: Working with JSON arrays and objects
description: Learn how to query columns containing multiple values (arrays) and nested objects (JSON) in Anyquery.
---

Anyquery is not strictly normalized. It can handle JSON arrays and objects in columns. Those are stored as text and can be queried using JSON functions. Most plugins return JSON data, so it's essential to know how to work with them.

## Arrays

Let's say we have a product table with a column containing an array of tags:

```sql
CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  name TEXT,
  tags TEXT
);
```

| id  | name   | tags                               |
| --- | ------ | ---------------------------------- |
| 1   | Apple  | ["fruit", "red", "sweet"]          |
| 2   | Carrot | ["vegetable", "orange", "healthy"] |

### Calculating the number of tags

To calculate the number of tags for each product, you can use the `json_array_length` function:

```sql "json_array_length(tags)"
SELECT name, json_array_length(tags) AS number_of_tags
FROM products;
```

| name   | number_of_tags |
| ------ | -------------- |
| Apple  | 3              |
| Carrot | 3              |

### Exploding the tags

To explode the tags into separate rows, you can use the `json_each` virtual table:

```sql "json_each(tags)"
SELECT name, value AS tag
FROM products, json_each(tags);
```

| name   | tag       |
| ------ | --------- |
| Apple  | fruit     |
| Apple  | red       |
| Apple  | sweet     |
| Carrot | vegetable |
| Carrot | orange    |
| Carrot | healthy   |

From that, you have a normalized view of the tags.

### Filtering by tag

To filter products by a specific tag, you can use the `json_has` function (from 0.3.2):

```sql "json_has(tags, 'fruit')"
SELECT name
FROM products
WHERE json_has(tags, 'fruit');
```

| name  |
| ----- |
| Apple |

### Inserting a value into an array

```sql "json_insert(tags, '$[#]', 'juicy')"
-- Insert 'juicy' into the tags of all products at the end
SELECT name, json_insert(tags, '$[#]', 'juicy') AS tags
FROM products;
```

| name   | tags                                     |
| ------ | ---------------------------------------- |
| Apple  | ["fruit","red","sweet","juicy"]          |
| Carrot | ["vegetable","orange","healthy","juicy"] |

### Updating a value in an array

```sql "json_replace(tags, '$[1]', 'green')"
-- Replace 'red' or 'orange' with 'green' in the tags of all products
SELECT name, json_replace(tags, '$[1]', 'green') AS tags
FROM products;
```

| name   | tags                            |
| ------ | ------------------------------- |
| Apple  | ["fruit","green","sweet"]       |
| Carrot | ["vegetable","green","healthy"] |

### Removing a value from an array

```sql "json_remove(tags, '$[#-1]')"
-- Remove the last tag from the tags of all products
SELECT name, json_remove(tags, '$[#-1]') AS tags
FROM products;
```

| name   | tags                   |
| ------ | ---------------------- |
| Apple  | ["fruit","red"]        |
| Carrot | ["vegetable","orange"] |

### Creating an array

```sql "json_array('fruit', 'green')"
-- Create an array with 'fruit' and 'green' for all products
SELECT name, json_array('fruit', 'green') AS tags
FROM products;
```

| name   | tags              |
| ------ | ----------------- |
| Apple  | ["fruit","green"] |
| Carrot | ["fruit","green"] |

## Objects

Let's say we have a product table with a column containing a JSON object:

```sql
CREATE TABLE products2 (
  id INTEGER PRIMARY KEY,
  name TEXT,
  properties TEXT
);
```

| id  | name   | properties                                                                        |
| --- | ------ | --------------------------------------------------------------------------------- |
| 1   | Apple  | {"color": "red", "taste": "sweet", "tags": ["fruit", "red", "sweet"]}             |
| 2   | Carrot | {"color": "orange", "taste": "sweet", "tags": ["vegetable", "orange", "healthy"]} |

### Extracting a field

To extract a field from the JSON object, you can use the `json_extract` function or its shorthand `->>`. You need to provide a JSON path. The object is referenced by `$`. To access a field, use `$.<field>`. And for an array, use `$[<index>]`. To get all elements of an array, use `$[*]`.

```sql "json_extract(properties, '$.color')"
SELECT name, properties->>'$.color' AS color
FROM products2;

-- Equivalent to:
SELECT name, json_extract(properties, '$.color') AS color
FROM products2;
```

| name   | color  |
| ------ | ------ |
| Apple  | red    |
| Carrot | orange |

### Extracting an array

Similarly, you can extract an array from the JSON object:

```sql "json_each(properties->'$.tags')"
SELECT name, value AS tag
FROM products2, json_each(properties->'$.tags');
```

| name   | tag       |
| ------ | --------- |
| Apple  | fruit     |
| Apple  | red       |
| Apple  | sweet     |
| Carrot | vegetable |
| Carrot | orange    |
| Carrot | healthy   |

### Ensuring a field exists

To ensure a field exists in the JSON object, you can use the `json_has` function:

```sql "json_has(properties, 'color')"
SELECT name, json_has(properties, 'color') AS has_color_field
FROM products2;
```

| name   | has_color_field |
| ------ | --------------- |
| Apple  | 1               |
| Carrot | 1               |

### Inserting or replace a field

```sql "json_set(properties, '$.price', 1.5)"
-- Insert or replace a 'price' field with 1.5 for all products
SELECT name, json_set(properties, '$.price', 1.5) AS properties
FROM products2;
```

| name   | properties                                                                             |
| ------ | -------------------------------------------------------------------------------------- |
| Apple  | {"color":"red","taste":"sweet","tags":["fruit","red","sweet"],"price":1.5}             |
| Carrot | {"color":"orange","taste":"sweet","tags":["vegetable","orange","healthy"],"price":1.5} |

### Create a new object

```sql "json_object('color', 'brown', 'price', 2.5)"
-- Create a new object with 'color' and 'price' for all products
SELECT name, json_object('color', 'brown', 'price', 2.5) AS properties
FROM products2;
```

| name   | properties                    |
| ------ | ----------------------------- |
| Apple  | {"color":"brown","price":2.5} |
| Carrot | {"color":"brown","price":2.5} |

### Filtering by a field

```sql "properties->>'$.color'"
-- Filter products by color
SELECT name
FROM products2
WHERE properties->>'$.color' = 'red';
```

| name  |
| ----- |
| Apple |

### Removing a field

```sql "json_remove(properties, '$.color')"
-- Remove the 'color' field from all products
SELECT name, json_remove(properties, '$.color') AS properties
FROM products2;
```

| name   | properties                                                |
| ------ | --------------------------------------------------------- |
| Apple  | {"taste":"sweet","tags":["fruit","red","sweet"]}          |
| Carrot | {"taste":"sweet","tags":["vegetable","orange","healthy"]} |

## Conclusion

Congratulations! You now know how to work with JSON arrays and objects in Anyquery (and most SQL databases).
