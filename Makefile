.PHONY: dev build migrate test

dev:
	docker compose up --build

build:
	cd backend && go build -o bin/api ./cmd/api
	cd frontend && npm ci && npm run build

migrate:
	docker compose run --rm api ./bin/api migrate

test:
	cd backend && go test ./...
	cd frontend && npm test -- --watchAll=false
