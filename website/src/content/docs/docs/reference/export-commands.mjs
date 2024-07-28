#! /usr/bin/env bun

import { $ } from "bun";
import { readdir } from "node:fs/promises";

// Export the commands
try {
	await $`anyquery tool generate-doc Commands`;
} catch (error) {
	// In case it's a build machine, anyquery might not be installed
	await $`../../../../../../main.out tool generate-doc Commands`;
}

// For each file, take the lop level h2 header, and add the frontmatter
// with the title equaling to the header text

const files = await readdir("Commands");

for (const file of files) {
	const fileContent = Bun.file(`Commands/${file}`);
	let text = await fileContent.text();

	const title = text.match(/## (.*)/)[1];
	// Strip the title from the markdown
	text = text.replace(`## ${title}\n\n`, "");
	const frontmatter = `---
title: ${title}
description: Learn how to use the ${title} command in AnyQuery.
---

`;
	console.log(`Writing ${title} to ${file}`);
	await Bun.write(`Commands/${file}`, frontmatter + text);
}
