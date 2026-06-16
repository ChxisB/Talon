FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o /cache-server ./cmd/server/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /cache-server /usr/local/bin/cache-server
EXPOSE 8083
ENTRYPOINT ["/usr/local/bin/cache-server"]
