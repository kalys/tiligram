# CORS Plan

## Why
- API: `https://tiligram.osmonov.com`
- Web:  `https://osmonov.com/tiligram`

Different origins (`osmonov.com` vs `tiligram.osmonov.com` are siblings, not same-origin), so browsers will block the search XHR/fetch without CORS headers on the API.

## Scope
Only the API in `start_api_command.go` / `internal/api` needs changes. The bot is unaffected.

## Approach

Add a small CORS middleware in `internal/api` and wrap the mux in `start_api_command.go`. No third-party dependency — the requirements are simple enough to implement directly with `net/http`.

### 1. Allowed origin(s)

- Production: `https://osmonov.com`
- Optionally: a `--cors-origin` CLI flag (repeatable / comma-separated) so local dev (`http://localhost:*`) can be enabled without code changes. Default to `https://osmonov.com`.

Do **not** use `*` — keeps the door open to add credentialed endpoints later and is stricter by default.

### 2. Middleware behavior

For every request:
- Read `Origin` header.
- If it matches an allowed origin, set:
  - `Access-Control-Allow-Origin: <origin>` (echo, not `*`)
  - `Vary: Origin` (so caches don't cross-pollute responses)
- For `OPTIONS` preflight requests, additionally set:
  - `Access-Control-Allow-Methods: GET, OPTIONS`
  - `Access-Control-Allow-Headers: Content-Type`
  - `Access-Control-Max-Age: 86400`
  - Respond `204 No Content` and return (don't fall through to the handler).
- Otherwise: pass through to the next handler.

Note: current API only accepts `GET /search` with a `q` query param and no custom headers, so the browser likely won't even send a preflight. Still, handle `OPTIONS` correctly for future-proofing.

### 3. Files to change

- `internal/api/cors.go` — new file: `CORS(allowed []string, next http.Handler) http.Handler`.
- `internal/api/cors_test.go` — new file: cover allowed origin, disallowed origin, OPTIONS preflight, `Vary` header presence.
- `start_api_command.go`:
  - Add `--cors-origin` `cli.StringSliceFlag` (default `["https://osmonov.com"]`).
  - Wrap `mux` with `api.CORS(c.StringSlice("cors-origin"), mux)` before passing to `http.Server`.

### 4. Out of scope

- Credentials (`Access-Control-Allow-Credentials`) — not needed; API is public read-only.
- Auth / rate limiting — separate concern.
- Reverse-proxy-level CORS (nginx/Caddy) — handling it in the app keeps the deployment simpler and the behavior testable.

## Verification

- Unit tests for the middleware.
- Manual: `curl -H 'Origin: https://osmonov.com' -i https://tiligram.osmonov.com/search?q=test` → response includes `Access-Control-Allow-Origin: https://osmonov.com` and `Vary: Origin`.
- Manual: same with a bogus origin → no `Access-Control-Allow-Origin` header.
