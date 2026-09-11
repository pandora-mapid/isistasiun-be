# CLAUDE.md

Guidance for Claude Code working in this repository.

Backend for Isi Stasiun (MAPID WebGIS Competition #2 2026, Tim List Pandora):
Go API + Python batch pipeline. Frontend is a separate repo (`../isistasiun-fe/`).

## Product context

Cross-team truth lives in `../Context/` — start at
[`../Context/INDEX.md`](../Context/INDEX.md). Most relevant here:
`../Context/02-BACKEND-SPEC.md` (endpoints, ownership, boundaries) and
`../Context/04-VALUE-PROP-AND-MONETIZATION.md` (`/premium/deep-analysis`, tiering).
Do not auto-load `../Context/source-docs/` (~500k tokens).

Ownership: **Priyapta** station/analytics/confidence/copilot-routing ·
**Arzaka** survey ingestion + pipeline callback · **Firaz** transparency/auth/premium
+ copilot-logic.

## Ops, deploy, quick start

See [`README.md`](README.md) — it covers Docker Compose, migrations, the pipeline
job commands, and the VPS/CI deploy flow. Not repeated here.

```bash
make up             # local stack
make be-run         # Go API, hot reload (air)
make be-test        # go test ./...
make be-migrate-up  # apply migrations
docker compose exec pipeline python -m pytest -q   # pipeline tests
```

CI (`.github/workflows/ci.yml`): `go vet` + `go test` + `pytest` + image build, every PR.

## Architecture invariants

- **Router is Fiber; base path `/api/v1`.** Standard response envelope:
  `{ "success": bool, "data": {...}, "error": null|{...} }` (`internal/response`).
- **Heavy geometry / vector tiles never pass through the Go API.** Nginx serves
  `location /tiles/`; the API returns attribute data only (popup, filter,
  transparency). Breaking this is a hard regression (`../Context/02-BACKEND-SPEC §5`).
- **Two auth schemes, do not mix them:**
  - `/pipeline/*` and `/survey/*` → `X-Service-Key` header (`middleware.RequireServiceKey`),
    service-to-service, **not** JWT.
  - `/auth/*`, `/premium/*` → JWT, role check public vs operator.
  - `/analytics/*`, `/stations/*`, `/confidence-layer`, `/transparency/*` → **open**,
    no token. Base analytics layers must stay free (`../Context/04 §3`).
- **Mount middleware with `app.Use("<prefix>", …)`, never `Group("", …)`.**
  `Group("")` == `Use("/api/v1", …)` — it applies to every route registered after
  it. That once made transparency/auth/premium all demand `X-Service-Key` and 401.
  `internal/router/router_test.go` locks the tier of each route.
- **`/copilot/query` is split by owner:** routing + rate limit in
  `internal/copilot/handler.go` (Priyapta); intent parsing + AI logic in
  `internal/copilot/intent.go` + `service.go` (Firaz). Each query is classified
  locally first (intent, category, time slot), then enriched via `Client.Query` only
  if `AI_SERVICE_URL` answers — the endpoint returns correct filters either way.
- **Idempotency on all `/pipeline/*` callbacks** — they can be retried.
- Pipeline runs as one-shot jobs (cron/manual), not a long-running server.
- Operator accounts are created via `cmd/createoperator` with the password read from
  env, not a flag — no seed migration (would commit a password hash).

## Layout

`backend/` Go API — `internal/{station,analytics,confidence,survey,pipeline,transparency,auth,premium,copilot,middleware,router}`,
`cmd/api`, `migrations/*.sql` (golang-migrate). ·
`pipeline/` Python — `extraction/` (Gemini OCR + classify), `analysis/`
(isochrone, spatial_join, monte_carlo), `shared/`. ·
OpenAPI: `backend/internal/docs/openapi.yaml`; Swagger UI at `/docs` (dev only).

## Conventions for agents

- Smallest change that satisfies the request; match surrounding style.
- Run `make be-test` (and pipeline pytest if it was touched) before claiming done.
- When code and a `../Context/` doc disagree, surface it — don't silently pick one.
