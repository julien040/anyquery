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
    docs: defineCollection({
        loader: docsLoader(),
        schema: docsSchema({
            extend: z.object({
                banner: z.object({ content: z.string() }).default({
                    content:
                        'Available for a 6-month internship starting in January 2027. <a href="https://cdn.julienc.me/resume.pdf">Read my resume</a> or contact me at <a href="mailto:contact@julienc.me" class="underline">contact@julienc.me</a>.',
                }),
            }),
        }),
    }),
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
