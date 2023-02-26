# GoCert

A simple cli/tui tool to inspect TLS certificates details.

# Build

```
go build -o gocert main.go
```

On Linux for windows
```
GOOS=windows GOARCH=amd64 go build -o gocert.exe main.go
```

# Run it

Do a simple check

```
gocert --endpoint https://mywebsite.org
```

You can specify multiple urls
```
gocert --endpoint https://mywebsite.org --endpoint tcp://localhost:3306 --endpoint https://www.youtube.com
```

If you need to query multiple endpoints multiple times, it is possible to create a list of endpoints and create a TUI by adding the _-tui_ parameter:
```
gocert --endpoint https://mywebsite.org --endpoint tcp://localhost:3306 --endpoint https://www.youtube.com -tui
```
For a full list of options use:
```
gocert main.go --help
```
