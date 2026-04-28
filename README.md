# Sendr

A developer-facing transactional email API built in Go. Authenticate with Google, generate API keys, and send emails through a queued pipeline with rate limiting and automatic retries.

## Stack

Go · PostgreSQL · Redis · SendGrid · Docker · chi · RS256 JWT

## Architecture

Clean architecture — domain models, port interfaces, and adapters are fully separated. Core business logic has zero infrastructure imports.

```
internal/
├── core/
│   ├── domain/        # models
│   ├── ports/         # interfaces
│   └── services/      # business logic
├── adapters/          # postgres + redis implementations
├── handlers/          # HTTP layer
└── worker/            # background job processor
```

## API

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/health` | — | Liveness check |
| `GET` | `/auth/google` | — | Start OAuth flow |
| `GET` | `/auth/google/callback` | — | OAuth callback, returns JWT |
| `GET` | `/me` | JWT | Profile + plan info |
| `POST` | `/apikeys` | JWT | Create API key |
| `GET` | `/apikeys` | JWT | List API keys |
| `DELETE` | `/apikeys/:id` | JWT | Revoke API key |
| `POST` | `/emails/send` | API key | Queue an email |
| `GET` | `/emails/:id` | API key | Poll job status |

## Setup

**Prerequisites:** Go 1.21+, Docker, OpenSSL, `migrate` CLI

```bash
# 1. Clone and install dependencies
git clone https://github.com/muhammedfazall/Sendr
cd Sendr
go mod download

# 2. Start Postgres + Redis
docker compose up -d

# 3. Generate RSA key pair
openssl genrsa -traditional -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# 4. Configure environment
cp .env.example .env
# fill in values (see below)

# 5. Run migrations
migrate -path migrations -database "postgres://sendr:secret@localhost:5433/sendr?sslmode=disable" up

# 6. Start server
go run ./cmd/server
```

## Environment Variables

```env
DB_URL=postgres://sendr:secret@localhost:5433/sendr?sslmode=disable
REDIS_URL=redis://localhost:6379

JWT_PRIVATE_KEY_PATH=./private.pem
JWT_PUBLIC_KEY_PATH=./public.pem

GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=

SENDGRID_KEY=
FROM_EMAIL=noreply@yourdomain.com
FROM_NAME=Sendr

PORT=8080
```

## Sending an Email

```bash
# 1. Login → get JWT
open http://localhost:8080/auth/google

# 2. Create an API key
curl -X POST http://localhost:8080/apikeys \
  -H "Authorization: Bearer <jwt>" \
  -d '{"name":"my-key"}'

# 3. Send an email
curl -X POST http://localhost:8080/emails/send \
  -H "Authorization: Bearer mk_live_<prefix>.<secret>" \
  -H "Content-Type: application/json" \
  -d '{"to":"user@example.com","subject":"Hello","body":"Hello from Sendr"}'
# → 202 {"job_id":"...","message":"email queued"}

# 4. Poll status
curl http://localhost:8080/emails/<job_id> \
  -H "Authorization: Bearer mk_live_..."
# → {"status":"sent"}
```

## How It Works

- **Queueing** — `POST /emails/send` validates the API key, checks the Redis rate limit, and inserts a job into Postgres. Returns `202` immediately.
- **Worker** — polls every 2s, claims jobs atomically with `SELECT FOR UPDATE SKIP LOCKED`, delivers via SendGrid.
- **Retries** — retryable failures (5xx, network) back off at 10s → 60s → 300s. Non-retryable failures (4xx) go straight to the DLQ.
- **Rate limiting** — fixed daily window per user in Redis, resets at UTC midnight. Returns `X-RateLimit-*` headers on every response.
- **Zombie recovery** — jobs stuck in `processing` with an expired lock are reclaimed every 60s.

## Plans

| Plan | Daily limit |
|------|-------------|
| free | 100 emails |
| pro | 1,000 emails |
| max | 10,000 emails |

## License

MIT