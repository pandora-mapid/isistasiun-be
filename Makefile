.PHONY: help up down logs be-run be-test be-migrate-up be-migrate-down be-seed be-seed-down \
        be-operator pipeline-shell fmt deploy-pull deploy-logs pipeline-job

.DEFAULT_GOAL := help

help: ## Daftar target
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ---- local dev ----
up: ## Nyalakan stack lokal (build)
	docker compose up -d --build

down: ## Matikan stack lokal
	docker compose down

logs: ## Ikuti log semua service lokal
	docker compose logs -f --tail=100

be-run: ## Go API lokal, hot reload (butuh air)
	cd backend && air

be-test: ## go test ./...
	cd backend && go test ./... -count=1

# Buat/rotasi akun operator. Password lewat env, bukan flag.
#   OPERATOR_PASSWORD=... make be-operator EMAIL=ops@kai.id
be-operator: ## Buat/rotasi akun operator (EMAIL=, ROLE=)
	cd backend && go run ./cmd/createoperator -email $(EMAIL) -role $(or $(ROLE),operator)

be-migrate-up: ## Jalankan semua migration (lokal)
	docker compose --profile tools run --rm migrate 'migrate -path /migrations -database "$$DATABASE_URL" up'

be-migrate-down: ## Rollback 1 migration (lokal)
	docker compose --profile tools run --rm migrate 'migrate -path /migrations -database "$$DATABASE_URL" down 1'

be-seed: ## Demo data untuk endpoint station-summary (dev/demo saja)
	docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < backend/seed/demo_station_summary.sql

be-seed-down: ## Hapus demo data station-summary
	docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < backend/seed/demo_station_summary_down.sql

pipeline-shell: ## Shell ke container pipeline (lokal)
	docker compose exec pipeline bash

fmt: ## gofmt -w seluruh backend
	cd backend && gofmt -w .

# ---- server (run on the VPS, in /opt/isistasiun) ----
DEPLOY_COMPOSE = docker compose -f docker-compose.deploy.yml --env-file .env

deploy-pull: ## (server) pull image terbaru + up -d
	$(DEPLOY_COMPOSE) pull
	$(DEPLOY_COMPOSE) up -d --remove-orphans

deploy-logs: ## (server) ikuti log stack deploy
	$(DEPLOY_COMPOSE) logs -f --tail=100

# One-off batch job on the server:
#   make pipeline-job ARGS="extract-struk --station-id <uuid> --job-id j1 --images-dir /app/data/struk"
pipeline-job: ## (server) jalankan job pipeline sekali (ARGS="...")
	$(DEPLOY_COMPOSE) --profile batch run --rm pipeline python main.py $(ARGS)
