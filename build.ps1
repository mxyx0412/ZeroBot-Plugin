# Windows
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags "-s -w" -o zerobot.exe -trimpath

# Linux
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags "-s -w" -o zbp -trimpath
