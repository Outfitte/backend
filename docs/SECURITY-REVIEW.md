# Backend security review (V1)

This document records the V1 security audit of the Outfitte backend. It is a
point-in-time review of the application's security posture for the 0.1.x
release series. No code changes were required as a result of this audit; every
item below was confirmed against the current implementation.

For the vulnerability reporting process, see [`SECURITY.md`](../SECURITY.md).

## Token handling

- **Access tokens** are stateless JWTs. They are signed and verified with the
  server secret and are not persisted; verification happens in
  `internal/api/middleware/auth.go`.
- **Refresh tokens** are never stored in plaintext. The raw token is returned
  to the client once; the server stores only an HMAC-SHA256 hash of it as a
  revocable `Session` entity (`internal/service/auth.go`, `hashToken`). Lookups
  are performed by hash via `SessionRepository.FindByTokenHash`.
- **Rotation:** `/auth/refresh` rotates the refresh token — the presented
  session is consumed and a new session (new raw token) is issued
  (`AuthService.refreshSession`). `/auth/logout` deletes the session, revoking
  the refresh token server-side. A per-user session cap
  (`maxSessionsPerUser = 10`) bounds outstanding sessions.
- **No secret logging:** `internal/api/handler/auth.go` logs no token or
  password values. `Register` logs only `user_id` on success; `Login`,
  `Refresh`, and `Logout` log a bare `"succeeded"` with no credential material.
  Request bodies (which carry passwords and refresh tokens) are never logged.

## Secrets

- `JWT_SECRET` is required and validated at startup in
  `internal/config/config.go`: load fails fast if it is unset, and
  `Validate` rejects any value shorter than 32 characters
  (`JWT_SECRET must be at least 32 characters`).
- The secret is never logged. Startup logging in `cmd/server/run.go` emits only
  the listen port; the `Config` struct is never logged in full.

## CORS

The backend is **not exposed directly** to browsers. All traffic is
same-origin, reaching the backend through the frontend's nginx reverse proxy.
Accordingly, the application ships **no CORS middleware** and sets no
`Access-Control-*` headers — confirmed by the absence of any CORS handling in
`internal/api`. This is intentional: there is no cross-origin browser access to
permit, so the most restrictive posture (no CORS) is also the correct one. If a
future deployment exposes the API to a different origin, a deliberate,
allowlist-based CORS policy must be added at that point.

## Transport

The backend serves plain **HTTP** and performs no TLS termination. TLS is the
responsibility of the operator's reverse proxy, which also owns HSTS and any
other transport-security headers — these belong at the proxy, not in the
application. See `SELF_HOSTING.md` in the
[`outfitte/deploy`](https://github.com/Outfitte/deploy) repository (referenced
from the README [Self-hosting](../README.md#self-hosting) section) for the
recommended TLS and HSTS configuration.

## Known limitations (deferred to V2)

- **Rate limiting / brute-force protection** on `/auth/login` (and the other
  `/auth/*` endpoints) is **not implemented** and is explicitly deferred to V2.
  There is currently no application-level throttling of repeated failed login
  attempts. Operators who need interim protection can apply request-rate limits
  at the reverse proxy. This is a known limitation of the V1 release.
