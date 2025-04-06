set GOARCH=wasm
set GOOS=js
set CGO_ENABLED=0

if not exist output\wasm mkdir output\wasm
go build -ldflags="-w -s" -o output\wasm\main.wasm
