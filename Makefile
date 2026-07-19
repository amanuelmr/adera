# Development commands. `make up seed` gives a full running environment.

.PHONY: up down seed migrate run test test-race vet vuln lint docker-build check

up: ## start postgres + minio + api via docker compose
	docker compose up -d --build

down: ## stop the local environment (volumes preserved)
	docker compose down

seed: ## apply migrations and development seed data (in docker)
	docker compose run --rm api seed

migrate: ## apply migrations only (in docker)
	docker compose run --rm api migrate

run: ## run the API locally against docker-compose services
	go run ./cmd/api serve

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

vuln:
	govulncheck ./...

lint: ## requires golangci-lint (https://golangci-lint.run)
	golangci-lint run

docker-build:
	docker build -t adera-api:local .

check: vet test test-race vuln ## everything CI runs
