build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o speedtest main.go