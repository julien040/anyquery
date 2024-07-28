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
			favicon: "/favicon.png",
			customCss: ["./src/docs.css"],
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
					link: "/docs",
					label: "Getting started",
				},
				{
					label: "Usage",
					items: [
						{
							label: "Running queries",
							link: "/docs/usage/running-queries",
						},
						{
							label: "Managing plugins",
							link: "/docs/usage/plugins",
						},
						{
							label: "Managing profiles",
							link: "/docs/usage/managing-profiles",
						},
						{
							label: "Querying files",
							link: "/docs/usage/querying-files",
						},
						{
							label: "Alternative languages (PRQL, PQL)",
							link: "/docs/usage/alternative-languages",
						},
						{
							label: "Exporting results",
							link: "/docs/usage/exporting-results",
						},
						{
							label: "As a library",
							link: "/docs/usage/as-a-library",
						},
					],
				},
				{
					autogenerate: { directory: "docs/reference" },
					label: "Reference",
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
			theme: "dracula",
			wrap: true,
		},
	},
});
