.PHONY: dev build migrate test security-check visual-test visual-update

dev:
	docker compose rm -sf frontend
	docker compose up --build

build:
	cd backend && go build \
	  -ldflags "-X main.Version=$$(git rev-parse --short HEAD)" \
	  -o bin/api ./cmd/api
	cd frontend && npm ci && npm run build

# Applies pending migrations and exits (no server). The image ENTRYPOINT is /app/api, so "migrate" is its argument.
migrate:
	docker compose run --rm api migrate

test:
	cd backend && go test ./...
	cd frontend && npm test -- --watchAll=false

# AC-7 (FR-BB64): dependency integrity and vulnerability checks required in CI.
security-check:
	cd backend && go mod verify
	# Production dependencies gate the build; the full tree (incl. dev toolchain) is audited blocking in security.yml.
	cd frontend && npm audit --omit=dev --audit-level=high

# FR-BB320 AC-11: visual regression in the pinned Playwright container. Baselines are made only there (scripts/visual/run.sh).
visual-test:
	bash scripts/visual/run.sh test

visual-update:
	bash scripts/visual/run.sh update
