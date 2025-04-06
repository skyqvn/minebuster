set GOOS=windows
set CGO_ENABLED=0

if not exist output mkdir output
go build -ldflags="-H windowsgui -w -s" -o output\minebuster.exe
