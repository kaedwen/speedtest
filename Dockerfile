FROM golang:alpine AS build

WORKDIR /build

RUN ls -la

COPY . .

RUN ls -la

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -trimpath -ldflags="-s -w" -o speedtest main.go

FROM scratch

COPY --from=build /build/speedtest /speedtest

ENTRYPOINT ["/speedtest"]
