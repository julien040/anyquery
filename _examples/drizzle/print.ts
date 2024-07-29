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
