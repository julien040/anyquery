import { defineCollection, z } from "astro:content";
import { docsSchema } from "@astrojs/starlight/schema";

export const collections = {
    docs: defineCollection({ schema: docsSchema() }),
    integrations: defineCollection({
        schema: z.object({
            title: z.string(),
            description: z.string(),
            icon: z.string(),
        }),
    }),
    recipes: defineCollection({
        schema: z.object({
            title: z.string(),
            description: z.string(),
        }),
    }),
    databases: defineCollection({
        schema: z.object({
            name: z.string().min(1),
            url: z.string().min(1),
            icon: z.string().min(1),
            description: z.string().min(1),
        }),
        type: "data",
    }),
};
