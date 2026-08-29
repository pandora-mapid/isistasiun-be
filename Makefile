.PHONY: up down logs be-run be-test be-migrate-up be-migrate-down be-operator \
        pipeline-shell fmt deploy-pull deploy-logs pipeline-job

# ---- local dev ----
up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f --tail=100

# Run Go API locally with hot reload (requires air: go install github.com/air-verse/air@latest)
be-run:
	cd backend && air

be-test:
	cd backend && go test ./... -count=1

# Buat/rotasi akun operator. Password lewat env, bukan flag.
#   OPERATOR_PASSWORD=... make be-operator EMAIL=ops@kai.id
be-operator:
	cd backend && go run ./cmd/createoperator -email $(EMAIL) -role $(or $(ROLE),operator)

be-migrate-up:
	docker compose --profile tools run --rm migrate 'migrate -path /migrations -database "$$DATABASE_URL" up'

be-migrate-down:
	docker compose --profile tools run --rm migrate 'migrate -path /migrations -database "$$DATABASE_URL" down 1'

pipeline-shell:
	docker compose exec pipeline bash

fmt:
	cd backend && gofmt -w .

# ---- server (run on the VPS, in /opt/isistasiun) ----
DEPLOY_COMPOSE = docker compose -f docker-compose.deploy.yml --env-file .env

deploy-pull:
	$(DEPLOY_COMPOSE) pull
	$(DEPLOY_COMPOSE) up -d --remove-orphans

deploy-logs:
	$(DEPLOY_COMPOSE) logs -f --tail=100

# One-off batch job on the server:
#   make pipeline-job ARGS="extract-struk --station-id <uuid> --job-id j1 --images-dir /app/data/struk"
pipeline-job:
	$(DEPLOY_COMPOSE) --profile batch run --rm pipeline python main.py $(ARGS)
