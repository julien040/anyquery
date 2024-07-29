---
title: Drizzle
description: Learn how to connect Drizzle to anyquery.
---

![Drizzle](/icons/drizzle.svg)

Let's connect Drizzle to anyquery so that you can bridge the JavaScript world with anyquery. To do so, we'll use the MySQL driver for Drizzle.

## Prerequisites

Before you begin, ensure that you have the following:

- A working installation of anyquery
- A JavaScript environment (npm, Node.js, etc.)

## Step 1: Set up the project

First, create a new project and install `drizzle`:

```bash
npm init -y
npm i -D drizzle-orm drizzle-kit mysql2
```

Once done, let's create the configuration file for Drizzle:

```bash
touch drizzle.config.js
```

and add the following content:

```javascript
import { defineConfig } from "drizzle-kit";
export default defineConfig({
    schema: "./schema.ts",
    out: "./drizzle",
    dialect: "mysql",
    dbCredentials: {
        url: "mysql://root:password@localhost:8070/main",
    },
});
```

## Step 2: Pull the schema

Once done, let's launch the anyquery server on a second terminal:

```bash
anyquery server
```

Now, let's pull the schema:

```bash
npm exec drizzle-kit introspect
```

Congratulations! We have generated a schema of all the tables in the database.

:::caution
If you have any columns that contain spaces or hyphens, you will need to modify or delete them manually from the `schema.ts` file. It is a bug from `drizzle-kit`.
:::

## Step 3: Query the database

Let's do a simple query. We'll fetch all the commits from the [drizzle-team/drizzle-orm](https://github.com/drizzle-team/drizzle-orm) repository:

Make sure to install the [`git` plugin](/integrations/git) first:

```bash
anyquery install git
```

Once done, launch the server in the first terminal:

```bash
anyquery server
```

Now, let's create a new file called `print.ts` and add the following content:

```typescript
// print.ts
import { drizzle } from "drizzle-orm/mysql2";
import mysql from "mysql2/promise";
import * as schema from "./drizzle/schema.ts";
import { sql } from "drizzle-orm";

const connection = await mysql.createConnection({
    host: "127.0.0.1",
    port: 8070,
    database: "main",
});

const db = drizzle(connection, { schema: schema, mode: "default" });

const result = await db
    .select()
    .from(schema.git_commits)
    .where(sql`repository = 'https://github.com/drizzle-team/drizzle-orm.git'`);

for (const row of result) {
    console.log(
        `On ${row.author_date}, ${row.author_name} committed ${row.message}`
    );
}

console.log("Found", result.length, "commits");
```

Now, let's run the script.
Personally, to avoid transpiling the TypeScript code, I use `bun`:

```bash
bun print.ts
```

You should see the commits from the repository.

## Conclusion

You've successfully connected Drizzle to anyquery. You can now query your database using Drizzle and anyquery. Don't hesitate to share your use cases with us in the GitHub discussions. I'm trying to prioritize development based on the community's needs.

You can see the resulting code at [https://github.com/julien040/anyquery/tree/main/_examples/drizzle](https://github.com/julien040/anyquery/tree/main/_examples/drizzle)
