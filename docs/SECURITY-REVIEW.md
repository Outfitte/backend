# Backend Security Review — V1 Audit

**Date:** 2026-06-03
**Scope:** Outfitte backend, milestone M7

---

## Token handling

Access tokens are stateless JWTs signed with HMAC-SHA256 using `JWT_SECRET`. They carry user identity and role claims and are validated by the auth middleware on every protected route.

Refresh tokens are random values issued at login. The raw token is returned to the client once and never persisted; only an HMAC-SHA256 keyed digest (`hashToken(secret, rawToken)`) is stored in the `sessions` table. This makes sessions revocable: `DELETE /sessions` invalidates a token without knowing its plaintext value.

On `POST /auth/refresh` the old session is deleted and a new one is created — tokens are rotated on every use and cannot be replayed.

`internal/api/handler/auth.go` was audited for sensitive value leakage:

- `Register` logs `user_id` on success only. Passwords and tokens are not logged.
- `Login`, `Refresh`, and `Logout` log no sensitive values on success.
- Error paths log the opaque `error` value, which does not contain credentials.

---

## Secrets

`JWT_SECRET` is read from the environment and validated at startup (`internal/config/config.go`):

- The server refuses to start if `JWT_SECRET` is absent or shorter than 32 characters.
- The startup log emits only `port`; the secret value is never written to any log output.

Generate a suitable value with:

```sh
openssl rand -hex 32
```

---

## CORS

There is no CORS middleware in the application. `internal/api/server/server.go` registers routes directly on a plain `net/http.ServeMux` with no cross-origin response headers.

This is intentional: the backend is not exposed directly to browsers. All browser traffic reaches it as same-origin requests proxied through the frontend nginx reverse proxy. Adding permissive CORS headers would be a misconfiguration in this deployment model.

---

## Transport security

The backend serves plain HTTP. TLS termination is the responsibility of the operator's reverse proxy (nginx). HSTS and related transport-layer headers must be configured there, not in the application. See the self-hosting guide (`docs/SELF_HOSTING.md`) for the recommended reverse-proxy configuration.

---

## Known limitations / deferred

| Item | Status |
|------|--------|
| Rate limiting and brute-force protection on `POST /auth/login` | **Deferred to V2** — no per-IP or per-user request throttling is in place. Operators running in production should add rate limiting at the reverse-proxy layer until this is addressed in the application. |
