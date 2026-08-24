.PHONY: up down logs be-run be-migrate-up be-migrate-down pipeline-shell fmt

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f --tail=100

# Run Go API locally with hot reload (requires air: go install github.com/air-verse/air@latest)
be-run:
	cd backend && air

be-migrate-up:
	docker compose exec backend migrate -path /app/migrations -database "$${DATABASE_URL}" up

be-migrate-down:
	docker compose exec backend migrate -path /app/migrations -database "$${DATABASE_URL}" down 1

pipeline-shell:
	docker compose exec pipeline bash

fmt:
	cd backend && gofmt -w .
