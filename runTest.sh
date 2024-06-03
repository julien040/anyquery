#!/bin/bash

# Set the current working directory to the directory of the script
cd "$(dirname "$0")" || exit

go test -tags "vtable" -v -coverprofile=coverage.out ./module ./namespace ./rpc ./controller/config/registry ./controller
