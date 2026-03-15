# Outfitte

Self-hosted wardrobe management application built in Go.

## Architecture Principles

These rules apply across all milestones and must never be violated:

- **Hybrid layered architecture** — standard Handler → Service → Repository layering, with hexagonal Ports & Adapters applied *only* at the storage and media boundaries
- **Context flows everywhere** — `context.Context` is the first argument in every service, repository, and media provider method. Never omitted, never retrofitted
- **Domain errors are defined when needed** — errors are added to `internal/domain/errors.go` only at the point they are first required; no pre-emptive error catalogue
- **Domain stays pure** — `internal/domain` and `internal/ports` never import from `internal/adapter`, `internal/api`, or any infrastructure package
- **Adapters translate errors** — infrastructure errors are translated into domain errors at the adapter boundary, never leaked upward

## Directory Layout

```
internal/
├── domain/       # Pure structs and domain errors (added as needed)
├── ports/        # Go interfaces: StorageProvider, MediaProvider
├── service/      # Business logic — depends only on domain + ports
├── adapter/
│   ├── store/
│   │   ├── json/     # M1 implementation
│   │   └── sqlite/   # Future
│   └── media/
│       ├── local/    # M1 implementation
│       └── s3/       # Future
└── api/
    └── handler/  # HTTP handlers — depends on service layer
```

## Current Milestone: M0 — Foundation

Goal: get the app running and ready for feature development. No user-facing features.

In scope:
- Project scaffold (directory structure, Go module, linting, build tooling)
- Configuration system (env-var driven, fail fast on startup)
- Health check endpoint (no auth)
- `StorageProvider` and `MediaProvider` interfaces in `internal/ports` (no implementations yet)
- Docker Compose scaffold

Decisions:
- Router: stdlib `net/http` only
- Config: env vars only (no config file)
- No domain errors defined yet

## Handler Guidelines

Every HTTP handler must emit two `slog` log lines via `InfoContext`:
1. **On entry** — `"<action> called"` (e.g. `"register called"`)
2. **On success** — `"<action> succeeded"` with relevant context fields (e.g. `"user_id"`)

Error log calls must use `"error"` as the key for the error value (e.g. `h.log.ErrorContext(ctx, "...", "error", err)`).

## Task Guidelines

**Branch naming:** `username/tasknr-short-name`
Example: `alice/42-add-auth`

**Commit message format:** `tasknr: one sentence message`
Example: `42: add JWT-based authentication`

**PR title format:** `tasknr: short description`
Example: `42: add JWT-based authentication`
