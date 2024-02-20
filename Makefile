build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o speedtest main.go

build-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -trimpath -o speedtest.arm main.go