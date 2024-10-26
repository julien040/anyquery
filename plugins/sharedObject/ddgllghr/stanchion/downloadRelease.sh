#!/usr/bin/env bash
# Written in [Amber](https://amber-lang.com/)
# version: 0.3.4-alpha
# date: 2024-08-02 10:38:59
__release_files=(
    "https://github.com/dgllghr/stanchion/releases/download/v0.1.0-alpha.4/libstanchion-0.1.0-alpha.4-x86_64-linux.so"
    "https://github.com/dgllghr/stanchion/releases/download/v0.1.0-alpha.4/libstanchion-0.1.0-alpha.4-aarch64-macos.dylib"
    "https://github.com/dgllghr/stanchion/releases/download/v0.1.0-alpha.4/libstanchion-0.1.0-alpha.4-x86_64-macos.dylib"
    "https://github.com/dgllghr/stanchion/releases/download/v0.1.0-alpha.4/stanchion-0.1.0-alpha.4-x86_64-windows.dll"
)
__0_urls=("${__release_files[@]}")
__repo_name="dgllghr/stanchion"
__main_branch="main"
__readme_file="README.md"
__directories=("linux-amd64" "darwin-arm64" "darwin-amd64" "windows-amd64")
__extension=("so" "dylib" "dylib" "dll")

# Script
__4_directory_name=("${__directories[@]}")
index=0
for url in "${__0_urls[@]}"; do
    echo "Downloading ${url}..."
    mkdir -p "${__4_directory_name[${index}]}"
    __AS=$?
    if [ $__AS != 0 ]; then
        echo "Failed to create directory ${__4_directory_name[${index}]}"
    fi
    curl -L "${url}" >"${__4_directory_name[${index}]}/libstanchion.${__extension[${index}]}"
    __AS=$?
    if [ $__AS != 0 ]; then
        echo "Failed to download ${url}. Status code: $__AS"
    fi
    ((index++)) || true
done
if [ $(
    [ "_${__readme_file}" == "_" ]
    echo $?
) != 0 ]; then
    echo "Downloading README from ${__repo_name}..."
    curl -L https://raw.githubusercontent.com/${__repo_name}/${__main_branch}/${__readme_file} -o README.md
    __AS=$?
    if [ $__AS != 0 ]; then
        echo "Failed to download README from ${__repo_name}. Status code: $__AS"
    fi
fi
