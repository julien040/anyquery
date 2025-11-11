#! /usr/bin/env bun
import { $ } from "bun";
import { Glob } from "bun";
import { chdir } from "process";
import { stringify } from "@iarna/toml";
import readline from "readline";

console.log("Downloading sqlean shared objects");

const currentVersion = "0.28.0";

const platforms = [
    {
        name: "sqlean-linux-arm64.zip",
        folder: "linux-arm64",
        fileExtension: "so",
        goName: "linux/arm64",
    },
    {
        name: "sqlean-linux-x64.zip",
        folder: "linux-x86",
        fileExtension: "so",
        goName: "linux/amd64",
    },
    {
        name: "sqlean-macos-x64.zip",
        folder: "macos-x86",
        fileExtension: "dylib",
        goName: "darwin/amd64",
    },
    {
        name: "sqlean-macos-arm64.zip",
        folder: "macos-arm64",
        fileExtension: "dylib",
        goName: "darwin/arm64",
    },
    {
        name: "sqlean-win-x64.zip",
        folder: "windows-x64",
        fileExtension: "dll",
        goName: "windows/amd64",
    },
];

// Change the directory to the script directory
chdir(import.meta.dir);

let user = "";
let password = "";

// Request the user and password for the uploader
const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
});

rl.question("Enter the plugin-manager username: ", (u) => {
    user = u;
    rl.question("Enter the plugin-manager password: ", (p) => {
        password = p;
        rl.close();
    });
});

// Wait for the user and password
await new Promise((resolve) => rl.on("close", resolve));

// For each platform, we will download the file and extract it to the correct folder
// Then, we will copy each shared object to the correct folder. For example, crypto.dll goes to crypto/windows_amd64/lib.dll
for (const platform of platforms) {
    console.log(`Downloading ${platform.name}`);
    const url = `https://github.com/nalgeon/sqlean/releases/download/${currentVersion}/${platform.name}`;
    // Curl the file
    await $`curl -L ${url} > ${platform.name}`;
    // Unzip the file
    await $`unzip -o ${platform.name} -d ${platform.folder}`;

    // Glob the files in the folder
    const glob = new Glob(`${platform.folder}/*.${platform.fileExtension}`);
    for await (const file of glob.scan(".")) {
        // Folder name is the last part of the file name without the extension
        // @ts-ignore because with the glob, their won't be any undefined
        const libName = file.split("/").pop().split(".").shift();
        const folderName = "lib_" + libName;
        // Create the folder
        await $`mkdir -p ${folderName}/${platform.folder}`;
        const destination = `${folderName}/${platform.folder}/${libName}.${platform.fileExtension}`;

        console.log(`Copying ${file} to ${destination}`);
        await $`cp ${file} ${destination}`;
    }

    // Remove the zip file
    await $`rm ${platform.name}`;

    // Remove the folder
    await $`rm -r ${platform.folder}`;
}

const libs = [
    {
        name: "crypto",
        description: "Hashing, encoding and decoding data",
    },
    {
        name: "define",
        description: "User-defined functions and dynamic sql",
    },
    {
        name: "fileio",
        description: "Read and write files",
    },
    {
        name: "fuzzy",
        description: "Fuzzy string matching and phonetics",
    },
    {
        name: "ipaddr",
        description: "IP address manipulation",
    },
    {
        name: "math",
        description: "Math functions",
    },
    {
        name: "regexp",
        description: "Regular expressions",
    },
    {
        name: "stats",
        description: "Math statistics",
    },
    {
        name: "text",
        description: "String functions and Unicode",
    },
    {
        name: "time",
        description: "High-precision date/time",
    },
    {
        name: "uuid",
        description: "Universally Unique IDentifiers",
    },
    {
        name: "vsv",
        description: "CSV files as virtual tables",
    },
];

// For each library, we will write a manifest.toml, add the README.md and upload the files
for (const lib of libs) {
    const folderName = `lib_${lib.name}`;
    console.log(`Creating ${folderName}/manifest.toml`);
    const libName = `sqlean-${lib.name}`;
    chdir(folderName);
    let manifest = {
        name: libName,
        displayName: "sqlean " + lib.name,
        description: lib.description,
        version: currentVersion,
        author: "nalgeon",
        license: "MIT",
        homepage: `https://github.com/nalgeon/sqlean/blob/main/docs/${lib.name}.md`,
        repository: "https://github.com/nalgeon/sqlean",
        type: "sharedObject",
        minimumAnyqueryVersion: "0.0.1",
        file: [],
    };

    // Add to the manifest each platform that exists
    for (const platform of platforms) {
        const executablePath = `${platform.folder}/${lib.name}.${platform.fileExtension}`;
        const tempFile = Bun.file(executablePath);
        if (await tempFile.exists()) {
            // @ts-ignore
            manifest.file.push({
                platform: platform.goName,
                directory: platform.folder,
                executablePath: `${lib.name}.${platform.fileExtension}`,
            });
        }
    }

    // Write the manifest
    const file = Bun.file("manifest.toml");
    Bun.write(file, stringify(manifest));

    // Write the README
    const readmeURL = `https://raw.githubusercontent.com/nalgeon/sqlean/main/docs/${lib.name}.md`;
    await $`curl -L ${readmeURL} > README.md`;

    // Upload the files
    await $`../../../../store-manager/store-manager.out -u ${user} --config manifest.toml -p ${libName}`.env(
        { ...process.env, ANYQUERY_PASSWORD: password }
    );

    // Go back to the root directory
    chdir(import.meta.dir);
}
