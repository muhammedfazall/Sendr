# ─── Sendr — Root Makefile ────────────────────────────────────────────
# Orchestrates infra, backend, frontend, and migrations from one place.
# Usage:
#   make dev        — spin up everything (infra → migrate → backend + frontend)
#   make stop       — tear down Docker containers
#   make clean      — stop containers and remove build artifacts

# ── Paths ─────────────────────────────────────────────────────────────
BACKEND_DIR  = backend
FRONTEND_DIR = frontend

# ── Database ──────────────────────────────────────────────────────────
DB_URL = postgres://sendr:secret@localhost:5433/sendr?sslmode=disable

# ──────────────────────────────────────────────────────────────────────
# Targets
# ──────────────────────────────────────────────────────────────────────

.PHONY: dev stop clean infra infra-wait migrate-up migrate-down migrate-drop backend frontend install lint build

## ── Composite ────────────────────────────────────────────────────────

# Start everything: infra → migrations → backend & frontend in parallel
dev: infra infra-wait migrate-up
	@echo ""
	@echo ============================================
	@echo   Starting backend and frontend ...
	@echo ============================================
	$(MAKE) -j2 backend frontend

## ── Infrastructure ───────────────────────────────────────────────────

# Spin up Postgres + Redis via Docker Compose
infra:
	docker compose -f $(BACKEND_DIR)/docker-compose.yml up -d

# Wait for Postgres and Redis to be ready before running migrations
infra-wait:
	@echo Waiting for Postgres ...
	@powershell -Command "for ($$i=0; $$i -lt 30; $$i++) { try { $$null = New-Object System.Net.Sockets.TcpClient('localhost', 5433); Write-Host 'Postgres ready'; break } catch { Start-Sleep 1 } }"
	@echo Waiting for Redis ...
	@powershell -Command "for ($$i=0; $$i -lt 30; $$i++) { try { $$null = New-Object System.Net.Sockets.TcpClient('localhost', 6379); Write-Host 'Redis ready'; break } catch { Start-Sleep 1 } }"

# Tear down Docker containers
stop:
	docker compose -f $(BACKEND_DIR)/docker-compose.yml down

## ── Migrations ───────────────────────────────────────────────────────

migrate-up:
	cd $(BACKEND_DIR) && migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	cd $(BACKEND_DIR) && migrate -path migrations -database "$(DB_URL)" down

migrate-drop:
	cd $(BACKEND_DIR) && migrate -path migrations -database "$(DB_URL)" drop -f

## ── Backend ──────────────────────────────────────────────────────────

backend:
	cd $(BACKEND_DIR) && go run ./cmd/server

## ── Frontend ─────────────────────────────────────────────────────────

# Install npm dependencies
install:
	cd $(FRONTEND_DIR) && npm install

frontend:
	cd $(FRONTEND_DIR) && npm run dev

## ── Quality ──────────────────────────────────────────────────────────

lint:
	cd $(BACKEND_DIR) && go vet ./...
	cd $(FRONTEND_DIR) && npm run lint

build:
	cd $(BACKEND_DIR) && go build -o bin/sendr ./cmd/server
	cd $(FRONTEND_DIR) && npm run build

## ── Cleanup ──────────────────────────────────────────────────────────

clean: stop
	cd $(BACKEND_DIR) && if exist bin rmdir /s /q bin
	cd $(FRONTEND_DIR) && if exist dist rmdir /s /q dist
