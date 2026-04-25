DB_URL=postgres://sendr:secret@localhost:5433/sendr?sslmode=disable

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

migrate-drop:
	migrate -path migrations -database "$(DB_URL)" drop -f