# Outfitte

Self-hosted wardrobe management application built in Go.

> **Status:** Early development — M0 (Foundation) in progress. No user-facing features yet.

## Overview

Outfitte lets you catalogue your clothing, organise items into locations, log wear events, and build outfit journals — all from your own infrastructure.

## Running with Docker Compose

```bash
cp .env.example .env   # edit STORAGE_DATA_PATH and MEDIA_STORAGE_PATH
docker compose up
```

See `.env.example` for all available variables.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | HTTP listen port |
| `APP_ENV` | `dev` | Runtime environment (`dev`/`prod`) |
| `STORAGE_DATA_PATH` | *(required)* | Directory for JSON storage data |
| `MEDIA_STORAGE_PATH` | *(required)* | Directory for media files |
| `LOG_LEVEL` | `info` | Log verbosity |
| `JWT_SECRET` | *(required)* | Secret key for signing JWTs; min 32 chars (`openssl rand -hex 32`) |

## Linting

Install [golangci-lint](https://golangci-lint.run/usage/install/) then run:

```bash
golangci-lint run ./...
```

## Roadmap

| Milestone | Description |
|-----------|-------------|
| M0 | Foundation — scaffold, config, health check |
| M1 | Users, Items & Locations |
| M2 | Wear & Archive Lifecycle |
| M3 | Outfits & Calendar |
| M4 | Seller URL & Price Tracking |
| M5 | Family Sharing |
| M6 | Polish & Public V1 Launch |
