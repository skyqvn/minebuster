if not exist output\wasm mkdir output\wasm

del /f resource.syso
set GOARCH=wasm
set GOOS=js
set CGO_ENABLED=0

go build -ldflags="-w -s" -o output\wasm\main.wasm
