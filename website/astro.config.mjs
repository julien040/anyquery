import { defineConfig } from "astro/config";
import alpinejs from "@astrojs/alpinejs";
import tailwind from "@astrojs/tailwind";

import starlight from "@astrojs/starlight";

// https://astro.build/config
export default defineConfig({
	integrations: [
		alpinejs(),
		tailwind({}),
		starlight({
			title: "Anyquery",
			credits: false,
			favicon: "/favicon.svg",
			logo: {
				src: "./public/images/logo.png",
				alt: "Anyquery logo",
			},
			components: {
				Footer: "./src/components/footer-docs.astro",
			},
			description:
				"Anyquery allows you to run SQL queries on pretty much any data source, including REST APIs, local files, SQL databases, and more.",
			sidebar: [
				{
					autogenerate: { directory: "docs" },
					label: "Introduction",
				},
				{
					autogenerate: { directory: "docs/Features" },
					label: "Features",
				},
				{
					autogenerate: { directory: "connection-guide" },
					label: "Connection guide",
				},
				{
					autogenerate: { directory: "recipes" },
					label: "Recipes",
					collapsed: true,
				},
			],
		}),
	],
	prefetch: {
		prefetchAll: true,
	},
	markdown: {
		shikiConfig: {
			theme: "nord",
			wrap: true,
		},
	},
});
