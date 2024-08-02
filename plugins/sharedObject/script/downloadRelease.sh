#!/usr/bin/env bash
# Written in [Amber](https://amber-lang.com/)
# version: 0.3.4-alpha
# date: 2024-08-02 10:38:59
__AMBER_ARRAY_0=("https://github.com/asg017/sqlite-http/releases/download/v0.1.1/sqlite-http-v0.1.1-loadable-linux-x86_64.tar.gz" "https://github.com/asg017/sqlite-http/releases/download/v0.1.1/sqlite-http-v0.1.1-loadable-macos-aarch64.tar.gz" "https://github.com/asg017/sqlite-http/releases/download/v0.1.1/sqlite-http-v0.1.1-loadable-macos-x86_64.tar.gz" "https://github.com/asg017/sqlite-http/releases/download/v0.1.1/sqlite-http-v0.1.1-loadable-windows-x86_64.zip");
__0_urls=("${__AMBER_ARRAY_0[@]}")
__1_repo="asg017/sqlite-http"
__2_main_branch="main"
__3_readme_file="README.md"
__AMBER_ARRAY_1=("linux-amd64" "darwin-arm64" "darwin-amd64" "windows-amd64");
__4_directory_name=("${__AMBER_ARRAY_1[@]}")
index=0;
for url in "${__0_urls[@]}"
do
    echo "Downloading ${url}..."
    mkdir -p ${__4_directory_name[${index}]};
    __AS=$?;
if [ $__AS != 0 ]; then
        echo "Failed to create directory ${__4_directory_name[${index}]}"
fi
    curl -L ${url} | tar -xz -C ${__4_directory_name[${index}]} -f - ;
    __AS=$?;
if [ $__AS != 0 ]; then
        echo "Failed to download ${url}. Status code: $__AS"
fi
    (( index++ )) || true
done
if [ $([ "_${__3_readme_file}" == "_" ]; echo $?) != 0 ]; then
    echo "Downloading README from ${__1_repo}..."
    curl -L https://raw.githubusercontent.com/${__1_repo}/${__2_main_branch}/${__3_readme_file} -o README.md ;
    __AS=$?;
if [ $__AS != 0 ]; then
        echo "Failed to download README from ${__1_repo}. Status code: $__AS"
fi
fi