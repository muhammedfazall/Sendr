# Sendr

A developer-facing transactional email API built in Go. Authenticate with Google, generate API keys, and send emails through a queued pipeline with rate limiting and automatic retries.

## Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.26, chi/v5 |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |
| Email | SendGrid v3 |
| Payments | Razorpay |
| Auth | RS256 JWT + Google OAuth |
| Frontend | React 19 + Vite 8 |
| CSS | Tailwind CSS v4 |
| Containerization | Docker + Docker Compose |

## Architecture

Clean architecture — domain models, port interfaces, and adapters are fully separated. Core business logic has zero infrastructure imports.

```
backend/
├── cmd/server/              # Entry point
├── internal/
│   ├── core/
│   │   ├── domain/          # Models (User, Plan, APIKey, Job, Payment)
│   │   ├── ports/           # Interface definitions
│   │   └── services/        # Business logic (auth, email, api key, payment)
│   ├── adapters/            # Infrastructure implementations
│   │   ├── userrepo/        # PostgreSQL user repository
│   │   ├── apikeyrepo/      # PostgreSQL API key repository
│   │   ├── jobrepo/         # PostgreSQL job repository
│   │   ├── planrepo/        # PostgreSQL plan repository
│   │   ├── paymentrepo/     # PostgreSQL payment repository
│   │   ├── ratelimit/       # Redis fixed-window rate limiter
│   │   ├── tokenstore/      # Redis token store (refresh + blacklist)
│   │   └── emailsender/     # SendGrid adapter
│   ├── handlers/            # HTTP handlers
│   │   ├── authhandler/     # Google OAuth, JWT, refresh, logout
│   │   ├── apikeyhandler/   # API key CRUD
│   │   ├── emailhandler/    # Email send + status polling
│   │   ├── mehandler/       # User profile
│   │   ├── paymenthandler/  # Razorpay order + verify
│   │   └── webhookhandler/  # Razorpay webhook
│   ├── middleware/          # JWT auth, API key validation, CORS
│   ├── worker/              # Background job processor
│   ├── router/              # Route definitions
│   └── health/              # Health check (DB + Redis ping)
├── pkg/
│   ├── config/              # Env loading + validation
│   ├── db/                  # PostgreSQL (pgxpool) + Redis connections
│   ├── constants/           # Durations, error definitions
│   ├── helpers/             # API key gen + SHA-256 hashing
│   └── response/            # JSON response helpers
└── migrations/              # 6 migration pairs

frontend/
├── src/
│   ├── pages/               # Landing, Login, Callback, Dashboard, APIKeys,
│   │                        # SendEmail, MailHistory, Pricing, Profile
│   ├── components/          # Layout (sidebar with SVG icons)
│   ├── lib/                 # API client, auth provider, config
│   ├── index.css            # Design tokens + Tailwind
│   └── landing.css          # Landing page styles
└── index.html               # SEO meta tags
```

## API Endpoints

### Health & Public

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/health` | None | Liveness check (DB + Redis) |
| `GET` | `/plans` | None | List all available plans |

### Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/auth/google` | None | Start OAuth flow |
| `GET` | `/auth/google/callback` | None | OAuth callback, returns JWT |
| `GET` | `/auth/token` | JWT | Get token info |
| `POST` | `/auth/refresh` | JWT | Refresh access token |
| `POST` | `/auth/logout` | JWT | Logout and revoke tokens |

### User Management

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/me` | JWT | Get user profile + plan + usage |
| `PATCH` | `/me` | JWT | Update user profile |

### API Keys

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/apikeys` | JWT | Create API key (enforces plan limit) |
| `GET` | `/apikeys` | JWT | List user's API keys |
| `DELETE` | `/apikeys/{id}` | JWT | Revoke API key |

### Emails

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/emails/send` | API Key | Queue an email (enforces rate + wait) |
| `GET` | `/emails/{id}` | API Key | Get job status |
| `GET` | `/emails` | JWT | List user's email history |

### Payments (Razorpay)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/payments/orders` | JWT | Create Razorpay order for plan upgrade |
| `POST` | `/payments/verify` | JWT | Verify Razorpay payment signature |
| `POST` | `/webhooks/razorpay` | None | Razorpay webhook (HMAC-verified) |

## Authentication

**Google OAuth:** User visits `/auth/google` → redirected to Google → callback at `/auth/google/callback` → returns JWT.

**JWT:**
- Algorithm: RS256 (RSA 2048-bit)
- Access token TTL: 15 minutes
- Refresh token TTL: 7 days (idle timeout: 30 minutes)
- Blacklisted tokens stored in Redis on logout

**API Key:**
- Format: `mk_live_<8-char-prefix>.<32-char-secret>`
- Secret stored as SHA-256 hash (constant-time comparison)
- Use in `Authorization: Bearer mk_live_xxx.secret`

## Email Queue System

### Job Lifecycle

```
PENDING → PROCESSING → SENT
    ↓                    ↑
    └──── (retry) ───────┘
         (max 3 retries)
              ↓
           FAILED → DLQ
```

### Worker Process

- Poll interval: 2 seconds
- Batch size: 10 jobs per poll
- Concurrent jobs: 10 (semaphore-constrained)
- Lock duration: 30 seconds per job
- Zombie recovery: every 60 seconds

### Retry Strategy

| Attempt | Backoff |
|---------|---------|
| 1 | 10 seconds |
| 2 | 60 seconds |
| 3 | 300 seconds (5 minutes) |

**Retryable:** Network errors, 5xx from SendGrid. **Non-retryable:** 4xx errors → DLQ.

## Rate Limiting

- **Mechanism:** Redis fixed-window counter (`rate_limit:<userID>:<YYYY-MM-DD>`)
- **Window:** UTC midnight to midnight
- **Headers returned:** `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After` (on 429)

## Plans & Limits

| Plan | Daily Limit | Max API Keys | Wait Between Sends | Price (INR) |
|------|-------------|-------------|-------------------|-------------|
| free | 5 emails/day | 1 | 30 seconds | ₹0 |
| pro | 10 emails/day | 3 | 5 seconds | ₹299/month |
| max | Unlimited | Unlimited | None | ₹999/month |

Upgrade via the dashboard Pricing page. Razorpay handles payment; plan upgrades automatically after signature verification or webhook.

## Frontend

### User Flow

```
Landing (/) → Login (/login) → Google OAuth → Callback (/callback) → Dashboard (/dashboard)
```

### Route Map

| Route | Component | Auth | Description |
|-------|-----------|------|-------------|
| `/` | Landing | Public | Product landing page |
| `/login` | Login | Public | Google OAuth sign-in |
| `/callback` | Callback | Public | OAuth callback handler |
| `/dashboard` | Dashboard | Protected | Stats overview |
| `/keys` | APIKeys | Protected | API key management |
| `/send` | SendEmail | Protected | Email composer |
| `/history` | MailHistory | Protected | Email history |
| `/pricing` | Pricing | Protected | Plan comparison + upgrade |
| `/profile` | Profile | Protected | User profile |

### Design System

| Token | Value | Usage |
|-------|-------|-------|
| `--bg` | `#0a0a0a` | Page background |
| `--surface` | `#111111` | Card backgrounds |
| `--accent` | `#00d084` | Brand green (CTAs) |
| `--text` | `#e8e8e8` | Primary text |
| `--muted` | `#666666` | Secondary text |
| `--danger` | `#ff4d4d` | Errors |

Typography: DM Sans (UI) + DM Mono (code).

## Environment Variables

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `DB_URL` | Yes | PostgreSQL connection string | — |
| `REDIS_URL` | Yes | Redis connection string | — |
| `JWT_PRIVATE_KEY_PATH` | Yes | Path to RSA private key | — |
| `JWT_PUBLIC_KEY_PATH` | Yes | Path to RSA public key | — |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth client ID | — |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth client secret | — |
| `OAUTH_STATE_SECRET` | Yes | OAuth state encryption key | — |
| `SENDGRID_KEY` | Yes | SendGrid API key | — |
| `FROM_EMAIL` | Yes | Sender email address | — |
| `FROM_NAME` | Yes | Sender display name | Sendr |
| `RAZORPAY_KEY_ID` | Yes | Razorpay key ID | — |
| `RAZORPAY_KEY_SECRET` | Yes | Razorpay key secret | — |
| `RAZORPAY_WEBHOOK_SECRET` | Yes | Razorpay webhook secret | — |
| `BACKEND_URL` | Yes | Public backend URL (for OAuth redirect) | — |
| `PORT` | No | Server port | `8080` |
| `FRONTEND_URL` | No | Frontend origin (CORS) | `http://localhost:5173` |
| `ALLOWED_ORIGINS` | No | Comma-separated CORS origins | `http://localhost:5173` |
| `APP_ENV` | No | Environment | `development` |

## Setup

**Prerequisites:** Go 1.26+, Docker, OpenSSL

```bash
# 1. Clone
git clone https://github.com/muhammedfazall/Sendr
cd Sendr

# 2. Start Postgres + Redis
docker compose -f backend/docker-compose.yml up -d

# 3. Generate RSA key pair
openssl genrsa -traditional -out backend/private.pem 2048
openssl rsa -in backend/private.pem -pubout -out backend/public.pem

# 4. Configure environment
cp backend/.env.example backend/.env
# Fill in: DB_URL, REDIS_URL, GOOGLE_CLIENT_*, SENDGRID_KEY,
# RAZORPAY_*, BACKEND_URL, OAUTH_STATE_SECRET

# 5. Run migrations
migrate -path backend/migrations \
  -database "$(grep DB_URL backend/.env | cut -d= -f2)" up

# 6. Start backend
cd backend && go run ./cmd/server

# 7. Start frontend
cd ../frontend && npm install && npm run dev
```

## Sending an Email

```bash
# 1. Login → get JWT
open http://localhost:5173/login

# 2. Create an API key
curl -X POST http://localhost:8080/apikeys \
  -H "Authorization: Bearer <jwt>" \
  -d '{"name":"my-key"}'

# 3. Send an email
curl -X POST http://localhost:8080/emails/send \
  -H "Authorization: Bearer mk_live_<prefix>.<secret>" \
  -H "Content-Type: application/json" \
  -d '{"to":"user@example.com","subject":"Hello","body":"From Sendr"}'
# → 202 {"job_id":"...","message":"email queued"}

# 4. Poll status
curl http://localhost:8080/emails/<job_id> \
  -H "Authorization: Bearer mk_live_..."
# → {"status":"sent"}
```

## Deployment

**Backend:** Docker multi-stage build (`backend/Dockerfile`), production compose at `backend/docker-compose.prod.yml`.

**Frontend:** Built with Vite, deployable to Vercel (`frontend/vercel.json` with SPA rewrites) or any static host.

**CI/CD:** GitHub Actions workflow (`.github/workflows/deploy.yml`) builds, tests, pushes Docker image, and deploys to EC2 via SSH.

## Security

- API keys hashed with SHA-256 (constant-time comparison)
- JWT tokens blacklisted in Redis on logout
- OAuth state parameter for CSRF protection
- CORS origin whitelist (configurable)
- 1MB request body limit enforced
- Frontend API URL host whitelist (`config.js`)
- HTTPS enforced in production

## License

MIT License. See [LICENSE](LICENSE) file.
