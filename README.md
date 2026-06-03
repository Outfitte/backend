[![Go](https://github.com/Outfitte/backend/actions/workflows/go.yml/badge.svg)](https://github.com/Outfitte/backend/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/Outfitte/backend/graph/badge.svg?token=CCAGD8KF43)](https://codecov.io/gh/Outfitte/backend)
[![Dependabot Updates](https://github.com/Outfitte/backend/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/Outfitte/backend/actions/workflows/dependabot/dependabot-updates)

# Outfitte

Self-hosted wardrobe management application built in Go (1.26).

Outfitte lets you catalogue your clothing, organise items into locations, log wear events, and build outfit journals — all from your own infrastructure.

## Running locally

```bash
export DB_DRIVER=sqlite
export DB_DSN=/tmp/outfitte.db
export JWT_SECRET=$(openssl rand -hex 32)
export MEDIA_STORAGE_PATH=/tmp/outfitte-media
export APP_ENV=dev
export LOG_LEVEL=info

go run ./cmd/server
```

## Self-hosting

See [outfitte/deploy](https://github.com/Outfitte/deploy) for Docker Compose and deployment guides.

## Environment variables

| Variable | Default | Required | Description |
|---|---|---|---|
| `DB_DRIVER` | `sqlite` | | Storage driver (`sqlite` or `json`) |
| `DB_DSN` | — | yes | SQLite: path to database file (e.g. `/data/outfitte.db`); JSON: path to storage directory |
| `JWT_SECRET` | — | yes | Secret for signing JWTs — min 32 chars (`openssl rand -hex 32`) |
| `MEDIA_STORAGE_PATH` | — | yes | Directory for uploaded media files |
| `APP_ENV` | `dev` | | Runtime environment (`dev`/`prod`) |
| `LOG_LEVEL` | `info` | | Log verbosity (`debug`/`info`/`warn`/`error`) |
| `SERVER_PORT` | `8080` | | HTTP listen port |

## API reference

The full OpenAPI 3.1 specification lives at [`docs/openapi.yaml`](docs/openapi.yaml). To browse it locally:

```bash
npx @redocly/cli preview-docs docs/openapi.yaml
```

A rendered HTML version is produced by CI and attached to each workflow run as the `api-reference` artifact.

## Tests and coverage

```bash
# run all tests
go test ./...

# with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

The CI gate requires ≥ 90% line coverage. New code is expected to maintain this threshold.

## Lint and format

Install [golangci-lint](https://golangci-lint.run/usage/install/) then run:

```bash
golangci-lint run ./...
```

The linter also enforces `gofmt` and `goimports` formatting. Run before committing:

```bash
gofmt -w .
goimports -local github.com/outfitte/backend -w .
```

## License

Outfitte is released under the [GNU Affero General Public License v3.0 only](LICENSE) (AGPL-3.0-only).

By contributing you agree to the project's [Contributor License Agreement](https://github.com/Outfitte/outfitte/issues/I0-002).
