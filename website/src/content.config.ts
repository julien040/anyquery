import { defineCollection } from "astro:content";
import { z } from "astro/zod";
import { glob } from "astro/loaders";
import { docsLoader } from "@astrojs/starlight/loaders";
import { docsSchema } from "@astrojs/starlight/schema";

const databaseSchema = z.object({
    name: z.string().min(1),
    url: z.string().min(1),
    icon: z.string().min(1),
    description: z.string().min(1),
});

export const collections = {
    docs: defineCollection({ loader: docsLoader(), schema: docsSchema() }),
    integrations: defineCollection({
        loader: glob({
            pattern: "**/*.md",
            base: "./src/content/integrations",
        }),
        schema: z.object({
            title: z.string(),
            description: z.string(),
            icon: z.string(),
        }),
    }),
    recipes: defineCollection({
        loader: glob({ pattern: "**/*.md", base: "./src/content/recipes" }),
        schema: z.object({
            title: z.string(),
            description: z.string(),
        }),
    }),
    databases: defineCollection({
        loader: glob({ pattern: "**/*.yaml", base: "./src/content/databases" }),
        schema: databaseSchema,
    }),
    chats: defineCollection({
        loader: glob({ pattern: "**/*.yaml", base: "./src/content/chats" }),
        schema: databaseSchema,
    }),
};
