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
};
