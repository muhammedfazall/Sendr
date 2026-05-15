.PHONY: infra down backend frontend migrate dev

DB_URL=postgres://sendr:secret@localhost:5433/sendr?sslmode=disable

# Starts: postgres & redis
infra:
	cd backend && docker compose up -d

# DB schema changes
down:
	cd backend && docker compose down

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

# Apply all pending database migrations.
migrate:
	cd backend && migrate -path migrations -database "$(DB_URL)" up

# rollback ONE migration
migrate-down:
	cd backend && migrate -path migrations -database "$(DB_URL)" down 1

dev:
	make infra

test:
	cd backend && go test ./...