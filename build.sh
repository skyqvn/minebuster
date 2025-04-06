#!/bin/bash

rm -f go.mod go.sum
go mod init minebuster
go mod tidy
mkdir -p ./output

# 编译Linux版本
export GOOS=linux
export CGO_ENABLED=0
go build -ldflags="-w -s" -o ./output/minebuster

chmod +x build.sh
