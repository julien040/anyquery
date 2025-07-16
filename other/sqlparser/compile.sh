#!/bin/bash

# This script compiles the Yacc to sql.go
set -e
cd "$(dirname "$0")"
go run goyacc/goyacc.go -fo sql.go sql.y