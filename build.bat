if not exist output mkdir output

set GOOS=windows
set CGO_ENABLED=0

go generate
go build -ldflags="-H windowsgui -w -s" -o output\minebuster.exe
