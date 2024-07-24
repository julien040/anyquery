/** @type {import('tailwindcss').Config} */
export default {
	content: ["./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}"],
	theme: {
		extend: {
			boxShadow: {
				"outline-primary": "0px 0px 100px 0px rgba(135, 124, 255, 0.22)",
				"outline-secondary": "0px 0px 68.5px 0px rgba(255, 255, 255, 0.04)",
			},
			borderColor: {
				primary: "#202020",
			},
		},
	},
	plugins: [],
};
