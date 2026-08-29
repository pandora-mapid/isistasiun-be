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
docker compose exec pipeline python main.py extract-struk \
  --station-id <uuid> --job-id <job-id> --images-dir /app/data/struk
```

## Migrations

Skema ada di `backend/migrations/*.sql` (format `golang-migrate`). Install
[golang-migrate](https://github.com/golang-migrate/migrate) CLI untuk
apply/rollback manual, atau tambahkan ke Dockerfile image kalau mau otomatis
lewat `make be-migrate-up`.

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
