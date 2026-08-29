# Isi Stasiun — Backend

WebGIS estimasi potensi pendapatan non-tiket pada simpul transportasi massal
— MAPID WebGIS Competition 2026, Tim List Pandora. Repo ini berisi
**backend** (Go API + Python batch pipeline) saja — frontend dikelola
terpisah.

Mengukur *spending gap* per stasiun: potensi belanja komuter (`F × E × C × V`
per kategori) vs belanja yang tertangkap gerai in-station, disimulasikan
Monte Carlo (P10-P90). Lihat `Brainstorm_Claude_-_MAPID.md` dan
`ListPandora_1__IsiStasiun_docx.pdf` untuk detail metodologi lengkap.

## Tech Stack

| Layer | Tech |
|---|---|
| API | Go (Fiber) |
| Database | PostgreSQL + PostGIS |
| Batch pipeline | Python (GeoPandas, NumPy, Gemini VLM) |
| Object storage | Cloudflare R2 |
| Infra | Docker Compose, Nginx, Let's Encrypt |
| Basemap/data | MAPID MAPS, GEO MAPID, MAPID Apps (survey) |

## Struktur Repo

```text
isi-stasiun/
├── backend/       # Go API — internal/{station,analytics,confidence,survey,
│                  #   pipeline,transparency,auth,premium,copilot}
├── pipeline/      # Python batch: OCR extraction, spatial analysis, Monte Carlo
├── nginx/         # Reverse proxy config (prod) — /api dan /tiles saja
├── docker-compose.yml
└── docker-compose.prod.yml
```

Pembagian ownership: lihat `BACKEND_TASK_DIVISION_3_PERSON.md`
(Priyapta: station/analytics/confidence/copilot-routing · Arzaka: survey +
pipeline callback · Firaz: transparency/auth/premium/copilot-logic).

## Quick Start

```bash
cp .env.example .env        # isi API key MAPID, Gemini, JWT secret, dll
make up                      # docker compose up -d --build
make be-migrate-up           # jalankan migration (butuh golang-migrate, lihat catatan di bawah)
make be-test                 # go test ./... — tier akses + klasifikasi copilot
```

Buat akun operator untuk tier premium (password dibaca dari env, bukan flag,
supaya tidak masuk shell history):

```bash
OPERATOR_PASSWORD=... go run ./cmd/createoperator -email ops@kai.id -role operator
```

Sengaja tidak ada seed migration untuk ini — migration akan meng-commit hash
password ke repo dan menyamakan kredensial di semua checkout.

- Backend API: http://localhost:8080/api/v1 (health check: `/healthz`)
- Swagger UI (development only): http://localhost:8080/docs
- OpenAPI spec (development only): http://localhost:8080/docs/openapi.yaml
- Postgres: `localhost:5432`

## Development

**Backend** (hot reload via [air](https://github.com/air-verse/air)):
```bash
cd backend && cp .env.example .env
go install github.com/air-verse/air@latest
air
```

**Pipeline** (jalankan job manual, bukan long-running server):
```bash
# OCR / klasifikasi batch (butuh GEMINI_API_KEY)
docker compose exec pipeline python main.py extract-struk \
  --station-id <uuid> --job-id <job-id> --images-dir /app/data/struk
docker compose exec pipeline python main.py extract-properti \
  --station-id <uuid> --job-id <job-id> --images-dir /app/data/properti --manifest /app/data/plots.json
docker compose exec pipeline python main.py extract-gerai \
  --station-id <uuid> --job-id <job-id> --images-dir /app/data/gerai --manifest /app/data/gerai.json

# Simulasi Monte Carlo spending-gap (baca survey + struk_extractions dari Postgres)
docker compose exec pipeline python main.py simulate \
  --station-id <uuid> --job-id <job-id> --time-slot morning

# Test logika pipeline (parsing nominal struk, normalisasi kategori, Monte Carlo)
docker compose exec pipeline python -m pytest -q
```

`--manifest` opsional: JSON `{"G-07.jpg": {"gerai_id": "<uuid>"}}` memetakan
nama file foto ke id UUID dari lembar inventaris survei. Tanpa manifest,
id diambil dari nama file (hanya valid kalau tim survei menamai file dengan UUID).

## Migrations

Skema ada di `backend/migrations/*.sql` (format `golang-migrate`).

- **Lokal:** `make be-migrate-up` / `make be-migrate-down` — pakai image
  `migrate/migrate` lewat compose profile `tools`, nggak perlu install CLI.
- **Server:** binary `migrate` ikut di image prod (`backend/Dockerfile`), dan
  service `migrate` di `docker-compose.deploy.yml` menjalankannya sekali tiap
  `up` sebelum `backend` start (`depends_on: service_completed_successfully`).

## Deploy (VPS + CI/CD)

CI (`.github/workflows/ci.yml`) jalan tiap PR/push: `go vet` + `go test` +
`pytest` + build image. Deploy dev (`deploy-dev.yml`) jalan tiap push ke `dev`:
build & push image ke `ghcr.io/pandora-mapid/isistasiun-be/{api,pipeline}` →
scp stack file ke VPS → SSH → `docker compose pull && up -d` → cek `/healthz`.

TLS + routing di-handle **reverse proxy bersama** yang sudah ada di box
(`trackster-nginx-1` di network `shared-web-net`). Stack ini cuma expose
`backend` di network itu sebagai `${COMPOSE_PROJECT_NAME}-backend-1:8080`.
Postgres pakai container sendiri (butuh PostGIS), internal only.

**Sekali di server** (`/opt/isistasiun`, setelah deploy pertama nge-scp file ke sini):

```bash
cp .env.deploy.example .env      # isi semua CHANGE_ME (DATABASE_URL & POSTGRES_PASSWORD konsisten)
```

Lalu di config `trackster-nginx-1`, tambah server block:
`api.dev.isistasiun.trackster.cloud` → `http://isistasiun-dev-backend-1:8080`,
issue cert lewat certbot yang sama.

Deploy berikutnya cukup `git push` ke `dev`.

- `.env` di server **tidak** disentuh CI — itu satu-satunya sumber kebenaran.
  `IMAGE_TAG` di-overwrite tiap deploy ke tag per-commit.
- Container `pipeline` di-gate profile `batch` (tidak nyala terus). Jalankan job:
  `make pipeline-job ARGS="extract-struk --station-id <uuid> --job-id j1 --images-dir /app/data/struk"`
- GitHub Secrets yang dipakai: `VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY` (deploy key
  khusus, bukan key pribadi), `GHCR_PAT` (PAT classic, scope `write:packages` +
  `read:packages`).

## Catatan Arsitektur

- Vector tile geometry **tidak** lewat Go API sebagai payload berat —
  dilayani via Nginx (`location /tiles/`), API cuma handle attribute data.
- **Middleware dipasang dengan `Use("<prefix>", …)`, bukan `Group("", …)`.**
  `Group("")` di Fiber sama dengan `Use("/api/v1", …)` — berlaku ke setiap
  route yang didaftarkan sesudahnya. Pola itu sempat membuat transparency,
  auth, dan premium ikut menuntut `X-Service-Key` dan semuanya menjawab 401.
  `internal/router/router_test.go` mengunci tier tiap route supaya tidak
  terulang.
- Endpoint `/copilot/query` di-split: routing/rate-limit di Priyapta
  (`internal/copilot/handler.go`), logic AI di Firaz
  (`internal/copilot/intent.go` + `service.go`). Tiap query diklasifikasikan
  lokal lebih dulu — intent, kategori, dan slot waktu — lalu diperkaya model
  lewat `Client.Query` kalau `AI_SERVICE_URL` menjawab. Kalau tidak, endpoint
  tetap mengembalikan filter dan jawaban yang benar, cuma prosanya lebih kaku.
- Semua endpoint pipeline callback (`/pipeline/*`) dan survey ingestion
  (`/survey/*`) diproteksi `X-Service-Key`, bukan JWT — lihat
  `middleware.RequireServiceKey`.
- CORS (`CORS_ALLOWED_ORIGINS` di `.env`) default ke `http://localhost:3000`
  — sesuaikan begitu tahu origin frontend-nya.

## Belum Termasuk

Frontend (Next.js + MapLibre) dikerjakan sebagai repo/folder terpisah, dan
akan konsumsi API ini via `NEXT_PUBLIC_API_BASE_URL` yang mengarah ke
`/api/v1` di atas.
