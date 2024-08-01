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
							label: "Installing plugins",
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
							label: "SQL join between APIs",
							link: "/docs/usage/sql-joins",
						},
						{
							label: "MySQL server",
							link: "/docs/usage/mysql-server",
						},
						{
							label: "As a library",
							link: "/docs/usage/as-a-library",
						},
					],
				},
				{
					label: "Reference",
					items: [
						{
							label: "Functions",
							link: "/docs/reference/functions",
						},
						{
							label: "CLI",
							autogenerate: { directory: "docs/reference/Commands"},
							collapsed: true,
						},
					]
				},
				{
					autogenerate: { directory: "connection-guide" },
					label: "Connection guide",
				},
				{
					autogenerate: { directory: "docs/developers" },
					label: "Developers",
				},
				{
					autogenerate: { directory: "recipes" },
					label: "Recipes",
					collapsed: false,
				},
			],
			head: [
				/* 
				<!-- 100% privacy-first analytics -->
				<script data-collect-dnt="true" async defer src="https://scripts.simpleanalyticscdn.com/latest.js"></script>
				<noscript><img src="https://queue.simpleanalyticscdn.com/noscript.gif?collect-dnt=true" alt="" referrerpolicy="no-referrer-when-downgrade" /></noscript>
				*/
				{
					tag: "script",
					attributes: {
						"data-collect-dnt": "true",
						async: true,
						defer: true,
						src: "https://sa.anyquery.dev/latest.js",
					},
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
