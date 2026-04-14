FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine AS dirs
RUN mkdir -p /data/media

FROM cgr.dev/chainguard/curl:latest
COPY --from=dirs --chown=65532:65532 /data /data
COPY --from=builder /server /server
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
