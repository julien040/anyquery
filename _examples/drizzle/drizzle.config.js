import { defineConfig } from "drizzle-kit";
export default defineConfig({
    schema: "./schema.ts",
    out: "./drizzle",
    dialect: "mysql",
    dbCredentials: {
        url: "mysql://root:password@localhost:8070/main",
    },
});
