[![Go](https://github.com/Outfitte/Outfitte/actions/workflows/go.yml/badge.svg)](https://github.com/Outfitte/Outfitte/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/Outfitte/backend/graph/badge.svg?token=CCAGD8KF43)](https://codecov.io/gh/Outfitte/backend) 
[![Dependabot Updates](https://github.com/Outfitte/Outfitte/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/Outfitte/Outfitte/actions/workflows/dependabot/dependabot-updates)


# Outfitte

Self-hosted wardrobe management application built in Go.

> **Status:** Early development — M5 (Granular Sharing) in progress. Core REST API is functional.

## Overview

Outfitte lets you catalogue your clothing, organise items into locations, log wear events, and build outfit journals — all from your own infrastructure.

## API Reference

The full API specification is in [`docs/openapi.yaml`](docs/openapi.yaml) (OpenAPI 3.1).

To render it locally:

```bash
# Redocly CLI
npx @redocly/cli preview-docs docs/openapi.yaml

# Or paste the file contents into https://editor.swagger.io
```

## Running with Docker Compose

```bash
cp .env.example .env   # set JWT_SECRET and review DB_DSN / MEDIA_STORAGE_PATH
docker compose up
```

See `.env.example` for all available variables.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | HTTP listen port |
| `APP_ENV` | `dev` | Runtime environment (`dev`/`prod`) |
| `DB_DRIVER` | `sqlite` | Storage driver: `sqlite`, `json`, or `postgres` |
| `DB_DSN` | *(required)* | Data source name for the selected driver (see below) |
| `MEDIA_STORAGE_PATH` | *(required)* | Directory for media files |
| `LOG_LEVEL` | `info` | Log verbosity |
| `JWT_SECRET` | *(required)* | Secret key for signing JWTs; min 32 chars (`openssl rand -hex 32`) |

### DB_DSN format

- **SQLite** (`DB_DRIVER=sqlite`): path to the database file, e.g. `/data/outfitte.db`
- **Postgres** (`DB_DRIVER=postgres`): standard DSN, e.g. `postgres://user:pass@host:5432/outfitte?sslmode=disable` — not yet implemented; the app will exit with an unsupported driver error on startup
- **JSON** (`DB_DRIVER=json`): directory path for JSON storage files, e.g. `/data/storage` — the JSON file store is no longer the default but remains available for local development by swapping the adapter in `run.go`

## Linting

Install [golangci-lint](https://golangci-lint.run/usage/install/) then run:

```bash
golangci-lint run ./...
```

## Roadmap

| Milestone | Description | Status |
|-----------|-------------|--------|
| M0 | Foundation — scaffold, config, health check | ✓ Done |
| M1 | Users, Items & Locations | ✓ Done |
| M2 | Wear & Archive Lifecycle | ✓ Done |
| M3 | Outfits & Calendar | ✓ Done |
| M4 | Seller URL & Price Tracking | ✓ Done |
| M5 | Granular Sharing | In progress |
| M6 | Polish & Public V1 Launch | Planned |
