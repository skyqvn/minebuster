if not exist output\wasm mkdir output\wasm

set GOARCH=wasm
set GOOS=js
set CGO_ENABLED=0

go build -ldflags="-w -s" -o output\wasm\main.wasm
