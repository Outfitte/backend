FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM cgr.dev/chainguard/static:latest
COPY --from=builder /server /server
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
