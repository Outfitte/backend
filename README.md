[![Go](https://github.com/Outfitte/Outfitte/actions/workflows/go.yml/badge.svg)](https://github.com/Outfitte/Outfitte/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/Outfitte/Outfitte/graph/badge.svg?token=CCAGD8KF43)](https://codecov.io/gh/Outfitte/Outfitte)
[![Dependabot Updates](https://github.com/Outfitte/Outfitte/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/Outfitte/Outfitte/actions/workflows/dependabot/dependabot-updates)

# Outfitte

Self-hosted wardrobe management application built in Go.

> **Status:** Early development — M1 (Users, Items & Locations) complete. Core REST API is functional.

## Overview

Outfitte lets you catalogue your clothing, organise items into locations, log wear events, and build outfit journals — all from your own infrastructure.

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | — | Health check |
| `POST` | `/auth/register` | — | Register a new user |
| `POST` | `/auth/login` | — | Obtain access + refresh tokens |
| `POST` | `/auth/refresh` | — | Rotate refresh token |
| `POST` | `/auth/logout` | — | Revoke refresh token |
| `GET` | `/items` | JWT | List items |
| `POST` | `/items` | JWT | Create item |
| `GET` | `/items/{id}` | JWT | Get item |
| `PATCH` | `/items/{id}` | JWT | Update item |
| `DELETE` | `/items/{id}` | JWT | Delete item |
| `POST` | `/items/{id}/photos` | JWT | Upload photo |
| `DELETE` | `/items/{id}/photos/{key...}` | JWT | Delete photo |
| `PATCH` | `/items/{id}/location` | JWT | Assign item to location |
| `GET` | `/locations` | JWT | List location tree |
| `POST` | `/locations` | JWT | Create location |
| `GET` | `/locations/{id}` | JWT | Get location |
| `PATCH` | `/locations/{id}` | JWT | Update location |
| `DELETE` | `/locations/{id}` | JWT | Delete location |
| `PATCH` | `/locations/{id}/move` | JWT | Move location |
| `GET` | `/categories` | JWT | List categories |
| `GET` | `/media/{key...}` | JWT | Download media file |
| `GET` | `/admin/settings` | JWT + Admin | Get app settings |
| `PATCH` | `/admin/settings` | JWT + Admin | Update app settings |

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
- **Postgres** (`DB_DRIVER=postgres`): standard DSN, e.g. `postgres://user:pass@host:5432/outfitte?sslmode=disable` — accepted once the Postgres adapter is fully wired
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
| M2 | Wear & Archive Lifecycle | Planned |
| M3 | Outfits & Calendar | Planned |
| M4 | Seller URL & Price Tracking | Planned |
| M5 | Family Sharing | Planned |
| M6 | Polish & Public V1 Launch | Planned |
