#!/bin/bash

rm -f go.mod go.sum
go mod init minebuster
go mod tidy
mkdir -p ./output

# 编译Linux版本
export GOARCH=wasm
export GOOS=js
export CGO_ENABLED=0
go build -ldflags="-w -s" -o output\wasm\main.wasm

chmod +x build.sh
