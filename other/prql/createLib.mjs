#!/usr/bin/env bun

/* 
To build these files on macOS, you need to have the following installed:
- rustup
- zig
- cargo
- bun
- mingw-w64 (for x86_64-pc-windows-gnu)

I couldn't find a way to build the windows library for aarch64-pc-windows-gnullvm on macOS
so i'm dropping support for that target for now.
*/

import { $ } from "bun";
import process from "node:process";

const targets = [
	"aarch64-unknown-linux-musl",
	"x86_64-unknown-linux-musl",
	"x86_64-pc-windows-gnu",
	// "aarch64-pc-windows-gnullvm",
	"aarch64-apple-darwin",
	"x86_64-apple-darwin",
];

const equivalentZigTargets = [
	"aarch64-linux-musl",
	"x86_64-linux-musl",
	"x86_64-windows-gnu",
	// "aarch64-windows-gnu",
	"aarch64-macos",
	"x86_64-macos",
];

// Change the directory to the root of the project
$.cwd(__dirname);

for (let i = 0; i < targets.length; i++) {
	console.log(`Building for ${targets[i]}`);
	try {
		await $`rustup target add ${targets[i]}`;
		/*
    	We have to set CFLAGS to --target=${equivalentZigTargets[i]} because some libraries will overwrite the -target flag
   		and set it to the default target of the system.
    	*/
		await $`CC="zig cc -target ${equivalentZigTargets[i]}" CXX="zig c++ -target ${equivalentZigTargets[i]}" CFLAGS="--target=${equivalentZigTargets[i]}" cargo build --release --target ${targets[i]}`;
		await $`cp target/${targets[i]}/release/libprqlc_c.a libprqlc-${targets[i]}.a`;
		await Bun.sleep(3000);
	} catch (e) {
		process.exit(1);
	}
}
