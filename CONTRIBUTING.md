# Contributing to Outfitte

Thank you for your interest in contributing. Please read this document before opening a pull request.

## Contributor License Agreement

All contributors must sign the [Contributor License Agreement](https://github.com/Outfitte/outfitte/issues/I0-002) before a pull request can be merged. By submitting a PR you confirm that you have read and agreed to its terms.

## Development setup

**Requirements:** Go 1.26+, golangci-lint, goimports.

```bash
git clone https://github.com/Outfitte/backend
cd backend

# run the server locally
export DB_DRIVER=sqlite DB_DSN=/tmp/dev.db \
       JWT_SECRET=$(openssl rand -hex 32) \
       MEDIA_STORAGE_PATH=/tmp/outfitte-media
go run ./cmd/server
```

## TDD workflow

Tests are written before or alongside implementation — this is non-negotiable. Every new behaviour must have a test. Pull requests without adequate test coverage will not be merged.

Run tests:

```bash
go test ./...
```

The CI gate requires ≥ 90% line coverage across the repository. Check your contribution locally:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Lint and format

All code must pass `golangci-lint` and be formatted with `gofmt`/`goimports` before pushing:

```bash
golangci-lint run ./...
gofmt -w .
goimports -local github.com/outfitte/backend -w .
```

CI will fail if either check reports issues.

## Database migrations

- Migrations live in `internal/adapter/store/sqlstore/migrations/`.
- Each migration is a pair of numbered files: `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`.
- **Always provide the down migration.** A PR that adds an up migration without a corresponding down migration will not be merged.
- Never modify an existing migration file. If a schema change is needed, add a new numbered migration.

## Commit and PR conventions

Branch naming: `username/issuenr-short-description`

Commit format: `issuenr: one sentence in the imperative mood`

PR title format: `issuenr: short description`

Keep each PR focused on a single task. Large or mixed-concern PRs will be asked to split.
