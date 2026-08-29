## Ringkasan

<!-- Apa yang berubah & kenapa. Tautkan issue kalau ada. -->

## Domain

<!-- Priyapta: public/analytics/copilot-routing · Arzaka: survey/pipeline · Firaz: transparency/auth/premium/copilot-logic -->

- [ ] Priyapta
- [ ] Arzaka
- [ ] Firaz
- [ ] Shared (router / middleware / infra / migrations)

## Checklist

- [ ] `make be-test` hijau
- [ ] `pytest -q` hijau (kalau menyentuh `pipeline/`)
- [ ] Migration baru punya `.up.sql` **dan** `.down.sql`
- [ ] `internal/router/router_test.go` di-update kalau ada route baru
- [ ] `internal/docs/openapi.yaml` di-update kalau kontrak API berubah

## Catatan deploy

<!-- Perlu env var baru? migration? langkah manual di server? Tulis di sini. -->
