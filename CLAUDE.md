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

## Service Guidelines

- **`Update` uses full-replace semantics for all optional fields** — `nil` in `UpdateItemInput` means "clear the field", consistent for `Brand`, `Color`, `CategoryID`, `LocationID`, `PurchasePrice`, `PurchaseCurrency`, `PurchaseDate`, and `SellerURL`. Do not introduce `coalesce` helpers that silently preserve existing values; that creates an inconsistency between fields and makes clearing impossible. If a future field needs preserve-vs-clear distinction, use a three-state wrapper type, not a nil-means-preserve shortcut.
- **Future dates in tests must be computed dynamically** — never hardcode a year (e.g. `"2027-01-01"`); use `time.Now().AddDate(1, 0, 0).Format("2006-01-02")` so the test never becomes stale.

## Handler Guidelines

Every HTTP handler constructor must pre-scope the logger with a `"handler"` key set to the handler name (e.g. `logger.With("handler", "auth")`).

Every HTTP handler method must create a per-call scoped logger at the top with a `"call"` key set to the method name (e.g. `log := h.log.With("call", "Register")`), then use that logger for all log calls within the method.

Every HTTP handler must emit two `slog` log lines via `InfoContext`:
1. **On entry** — `"started"` (e.g. `log.InfoContext(ctx, "started")`)
2. **On success** — `"succeeded"` with relevant context fields (e.g. `log.InfoContext(ctx, "succeeded", "user_id", id)`)

Error log calls must use `"error"` as the key for the error value (e.g. `log.ErrorContext(ctx, "...", "error", err)`).

Every HTTP handler method must check `ctx.Err()` immediately after the "started" log line, before any service call. On cancellation, return 503 with `{"error": "request cancelled"}`.

Error messages in JSON responses must always be hardcoded string literals — never use `err.Error()`. If the domain error message is intentionally user-facing, copy the string explicitly. Using `err.Error()` couples the API contract to domain error wording, which can change silently.

Response DTO types belong in the same file as the handler for their primary entity (e.g. `locationResponse` lives in `location.go`, not in `share.go` even though `share.go` embeds it). Define a response type once in its primary handler file; other handlers import it from there.

## Ports Guidelines

- **Comments must be implementation-agnostic** — port interface comments must not mention storage-specific concepts such as "row", "record", "upsert", "insert", "N+1 queries", column names, or SQL keywords. Use domain-neutral language: "creates or updates", "entry", "association", "WornOn descending", "batched call".

## SQL Adapter Guidelines

- **Never use `INSERT OR REPLACE`** for tables with `ON DELETE CASCADE` children — SQLite implements it as DELETE + INSERT, which fires cascades and silently destroys child rows. Use `INSERT INTO ... ON CONFLICT(id) DO UPDATE SET ...` instead.
- **In-memory test DBs do not enforce FK constraints** — `openMigratedDB` uses `sql.Open` directly without `PRAGMA foreign_keys=ON`. Add `PRAGMA foreign_keys = ON` explicitly in any test that must verify cascade-safe behaviour.
- **Compile-time interface guard** — every new repository type must include `var _ ports.XxxRepository = (*XxxRepository)(nil)` immediately before the struct declaration.

## Task Guidelines

**Branch naming:** `username/tasknr-short-name`
Example: `alice/42-add-auth`

**Commit message format:** `tasknr: one sentence message`
Example: `42: add JWT-based authentication`

**PR title format:** `tasknr: short description`
Example: `42: add JWT-based authentication`
