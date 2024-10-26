#!/usr/bin/env bash
# Written in [Amber](https://amber-lang.com/)
# version: 0.3.4-alpha
# date: 2024-08-02 10:38:59
__release_files=(
    "https://github.com/asg017/sqlite-vec/releases/download/v0.1.3/sqlite-vec-0.1.3-loadable-linux-x86_64.tar.gz"
    "https://github.com/asg017/sqlite-vec/releases/download/v0.1.3/sqlite-vec-0.1.3-loadable-macos-aarch64.tar.gz"
    "https://github.com/asg017/sqlite-vec/releases/download/v0.1.3/sqlite-vec-0.1.3-loadable-macos-x86_64.tar.gz"
    "https://github.com/asg017/sqlite-vec/releases/download/v0.1.3/sqlite-vec-0.1.3-loadable-windows-x86_64.tar.gz"
)
__0_urls=("${__release_files[@]}")
__repo_name="asg017/sqlite-vec"
__main_branch="main"
__readme_file="README.md"
__directories=("linux-amd64" "darwin-arm64" "darwin-amd64" "windows-amd64")

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
    curl -L "${url}" | tar -xz -C "${__4_directory_name[${index}]}" -f -
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
