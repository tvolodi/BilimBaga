.PHONY: dev build migrate test security-check

dev:
	docker compose up --build

build:
	cd backend && go build \
	  -ldflags "-X main.Version=$$(git rev-parse --short HEAD)" \
	  -o bin/api ./cmd/api
	cd frontend && npm ci && npm run build

migrate:
	docker compose run --rm api ./bin/api migrate

test:
	cd backend && go test ./...
	cd frontend && npm test -- --watchAll=false

# AC-7 (FR-BB64): dependency integrity and vulnerability checks required in CI.
security-check:
	cd backend && go mod verify
	cd frontend && npm audit --audit-level=high
