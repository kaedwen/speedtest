FROM golang:1-alpine AS build

WORKDIR /build

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o speedtest ./cmd/main.go

FROM alpine

COPY --from=build /build/speedtest /speedtest

ENTRYPOINT ["/speedtest"]
