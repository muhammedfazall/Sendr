# Sendr

A transactional email API built in Go for developers. Sign in with Google, generate API keys, and send email through a queued pipeline with daily and monthly quotas, automatic retries, delivery analytics, reusable templates, and one-click unsubscribe. Ships with a React dashboard and a companion [CLI](../sendr-cli/README.md).

## Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.26, gin v1.9.1 |
| Database | PostgreSQL 16 (`pgx/v5` + `pgxpool`) |
| Cache/Queue | Redis 7 (`go-redis/v9`) |
| Email | SendGrid (default), Mailgun, SMTP, or Mock |
| Payments | Razorpay |
| Auth | RS256 JWT + Google OAuth 2.0 |
| Frontend | React 19.2 + Vite 8 |
| Routing | React Router 7 |
| CSS | Tailwind CSS v4 (via `@tailwindcss/vite`) |
| Linting | ESLint 10 |
| Containerization | Docker + Docker Compose |
| CI/CD | GitHub Actions |

## Architecture

Clean architecture — domain models, port interfaces, and adapters are fully separated. Core business logic has zero infrastructure imports.

```
backend/
├── cmd/server/              # Entry point
├── internal/
│   ├── core/
│   │   ├── domain/          # Models (User, Plan, APIKey, Job, Payment, Template, EmailEvent)
│   │   ├── ports/           # Interface definitions
│   │   └── services/        # Business logic (auth, apikey, email, payment, template)
│   ├── adapters/            # Infrastructure implementations
│   │   ├── userrepo/        # PostgreSQL user repository
│   │   ├── apikeyrepo/      # PostgreSQL API key repository
│   │   ├── jobrepo/         # PostgreSQL job repository (claims, retries, DLQ)
│   │   ├── planrepo/        # PostgreSQL plan repository
│   │   ├── paymentrepo/     # PostgreSQL payment repository
│   │   ├── emaileventrepo/  # PostgreSQL delivery-event repository
│   │   ├── templaterepo/    # PostgreSQL template repository
│   │   ├── unsubrepo/       # PostgreSQL unsubscribe repository
│   │   ├── ratelimit/       # Redis daily + monthly counters (Lua script)
│   │   ├── tokenstore/      # Redis token store (refresh + blacklist)
│   │   ├── eventbus/        # In-process pub/sub for email events
│   │   └── emailsender/     # Pluggable senders: SendGrid, Mailgun, SMTP, Mock
│   ├── handlers/            # HTTP handlers
│   │   ├── authhandler/     # Google OAuth, JWT, refresh, logout
│   │   ├── apikeyhandler/   # API key CRUD
│   │   ├── emailhandler/    # Send, status polling, history, stats
│   │   ├── mehandler/       # User profile + daily/monthly usage
│   │   ├── paymenthandler/  # Razorpay order + verify, plan listing
│   │   ├── templatehandler/ # Template CRUD
│   │   ├── unsubhandler/    # One-click unsubscribe
│   │   └── webhookhandler/  # Razorpay + SendGrid webhooks
│   ├── middleware/          # JWT auth, API key validation, CORS, throttle, logging, request ID
│   ├── worker/              # Background job processor
│   ├── router/              # Route definitions
│   ├── app/                 # Bootstrap + graceful shutdown
│   └── health/              # Health check (DB + Redis ping)
├── pkg/
│   ├── config/              # Env loading + validation
│   ├── db/                  # PostgreSQL (pgxpool) + Redis connections
│   ├── constants/           # Durations, error definitions
│   ├── helpers/             # API key gen + SHA-256 hashing
│   └── response/            # JSON response helpers
├── migrations/              # 11 migration pairs
├── Dockerfile               # Multi-stage production image
└── Dockerfile.migrate       # Migration runner image

frontend/
├── public/                  # favicon + icon sprite
├── src/
│   ├── pages/               # Landing, Login, Callback, Dashboard, APIKeys,
│   │                        # SendEmail, MailHistory, Pricing, Profile
│   ├── components/          # Layout (sidebar with SVG icons)
│   ├── lib/                 # API client, auth provider, config
│   ├── index.css            # Design tokens + Tailwind
│   ├── landing.css          # Landing page styles
│   ├── App.jsx              # Router + route definitions
│   └── main.jsx             # React entry point
├── index.html               # SEO meta tags
├── eslint.config.js         # ESLint flat config
├── vite.config.js           # Vite + React + Tailwind plugins
└── vercel.json              # SPA rewrites
```

## API Endpoints

27 routes total.

### Public

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/health` | None | Liveness check (DB + Redis) |
| `GET` | `/plans` | None | List all available plans |
| `POST` | `/webhooks/razorpay` | HMAC signature | Razorpay payment webhook |
| `POST` | `/webhooks/sendgrid` | None | SendGrid delivery event webhook |
| `GET` | `/unsubscribe` | Token in query | One-click unsubscribe page |
| `POST` | `/unsubscribe` | Token in query | Process an unsubscribe request |

### Authentication (throttled to 10 concurrent requests)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/auth/google` | None | Start OAuth flow (accepts `?redirect_uri=` for the CLI) |
| `GET` | `/auth/google/callback` | None | OAuth callback, sets one-time auth + refresh cookies |
| `GET` | `/auth/token` | One-time cookie | Exchange auth cookie for access JWT |
| `POST` | `/auth/refresh` | Refresh cookie | Rotate access + refresh tokens |
| `POST` | `/auth/logout` | JWT | Revoke tokens |

### User Management

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/me` | JWT | Get user profile + plan + daily/monthly usage |
| `PATCH` | `/me` | JWT | Update user profile (name) |

### API Keys

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/apikeys` | JWT | Create API key (enforces plan limit) |
| `GET` | `/apikeys` | JWT | List user's API keys |
| `DELETE` | `/apikeys/:id` | JWT | Revoke API key |

### Emails

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/emails/send` | API Key | Queue an email (enforces daily + monthly limits) |
| `GET` | `/emails/:id` | API Key | Get job status |
| `GET` | `/emails` | JWT | List email history (`?status=&limit=&offset=`, limit ≤ 100) |
| `GET` | `/emails/stats` | JWT | Aggregate delivery stats (sent/opens/clicks/bounces) |

### Templates

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/templates` | JWT | Create a template |
| `GET` | `/templates` | JWT | List templates |
| `GET` | `/templates/:id` | JWT | Get a template |
| `PATCH` | `/templates/:id` | JWT | Update a template |
| `DELETE` | `/templates/:id` | JWT | Delete a template |

### Payments (Razorpay)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/payments/orders` | JWT | Create Razorpay order for plan upgrade |
| `POST` | `/payments/verify` | JWT | Verify Razorpay payment signature |

## Authentication

**Google OAuth:** User visits `/auth/google` → redirected to Google → callback at `/auth/google/callback` → tokens set as HttpOnly cookies → frontend/CLI fetches the access token via `/auth/token` (one-time cookie, 60s TTL). The CLI passes `?redirect_uri=http://127.0.0.1:<port>/callback` and receives the token directly on its local callback.

**JWT:**
- Algorithm: RS256 (RSA 2048-bit)
- Access token TTL: 15 minutes
- Refresh token TTL: 7 days (rotated on every refresh; cookie scoped to `/auth` so it is only sent on auth routes)
- Blacklisted access tokens (`jti`) stored in Redis on logout
- Issuer `sendr`, audience `sendr-api`
- OAuth state is HMAC-signed and bound to the User-Agent (5-minute cookie)

**API Key:**
- Format: `mk_live_<12-char-hex-prefix>.<64-char-hex-secret>`
- Secret stored as SHA-256 hash, compared in constant time
- Use in `Authorization: Bearer mk_live_xxx.secret`
- The full key is shown once at creation; only the prefix is returned afterwards

## Email Pipeline

### Providers

The sender is pluggable via `EMAIL_PROVIDER`:

| Provider | Value | Required vars | Notes |
|----------|-------|---------------|-------|
| SendGrid (default) | `sendgrid` | `SENDGRID_KEY` | Supports CC/BCC, headers, tags, attachments |
| Mailgun | `mailgun` | `MAILGUN_DOMAIN`, `MAILGUN_KEY` | Supports CC/BCC, headers, tags, attachments |
| SMTP | `smtp` | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS` | STARTTLS; To + text/HTML bodies (no CC/BCC/attachments) |
| Mock (no send) | `mock` | — | Logs only; useful for local testing |

`MAILGUN_BASE_URL` optionally overrides the Mailgun API base (defaults to `https://api.mailgun.net/v3`), for EU regions or self-hosted proxies.

### Templates & Unsubscribe

- Templates are stored per user and rendered server-side with Go's templating. Pass `template_id` + `template_data` to `/emails/send`; explicit `subject`, `text_body`, or `html_body` values take precedence over rendered values.
- Every outbound email gets a signed `List-Unsubscribe` header; `/unsubscribe` validates the HMAC token and records the opt-out.

### Delivery Events

SendGrid event webhooks (`open`, `click`, `bounce`, etc.) are stored in `email_events`, linked to jobs via `sg_message_id`, and surfaced through `/emails/stats`.

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

- Jobs claimed atomically with `SELECT … FOR UPDATE SKIP LOCKED` (no double-sends)
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

**Retryable:** Network errors, 5xx from the provider. **Non-retryable:** 4xx errors → DLQ.

## Rate Limiting

Two independent quotas are enforced on `POST /emails/send`:

- **Mechanism:** Redis fixed-window counters using an atomic Lua script
- **Daily key:** `rate_limit:<userID>:<YYYY-MM-DD>`, TTL 25 hours (resets at UTC midnight)
- **Monthly key:** `usage_monthly:<userID>:<YYYY-MM>`, TTL 32 days (resets on the 1st, UTC)
- **Daily exceeded:** `429 rate_limit_exceeded`
- **Monthly exceeded:** `429 monthly_rate_limit_exceeded`
- **On 429:** `X-RateLimit-Reset` (unix ts) and `Retry-After` (seconds) headers
- Current usage (`usage_today`, `usage_this_month`, `remaining`, `remaining_monthly`) is exposed via `GET /me`
- Limits of `-1` mean unlimited and skip the corresponding check entirely

## Plans & Limits

| Plan | Daily Limit | Monthly Limit | Max API Keys | Price (INR) |
|------|-------------|---------------|--------------|-------------|
| free | 100 emails/day | 3,000 emails/month | 1 | ₹0 |
| pro | Unlimited | 50,000 emails/month | 5 | ₹299/month |
| max | Unlimited | 100,000 emails/month | Unlimited | ₹999/month |

`daily_limit = -1`, `monthly_limit = -1`, and `max_api_keys = -1` mean unlimited. Upgrades go through the dashboard Pricing page; Razorpay handles payment and the plan is upgraded after signature verification or via the `payment.captured` webhook.

> **Note:** `rate_wait_secs` still exists in the plan model and database for historical reasons, but all seeded plans now use `0` and no code enforces it — only the daily and monthly quotas are applied.

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
| `/pricing` | Pricing | Protected | Plan comparison + Razorpay upgrade |
| `/profile` | Profile | Protected | User profile |

Unknown routes redirect to `/`.

### Session Handling

- The access token is kept in `sessionStorage`; the refresh token stays in an HttpOnly cookie
- The API client refreshes silently when the JWT is within 60 seconds of expiry, and retries once on a `401`
- A failed refresh clears local state and emits `sendr:session-expired`, which shows a "Session expired" screen
- Requests use `credentials: 'include'` so the refresh cookie is sent to `/auth/*`

### Design System

| Token | Value | Usage |
|-------|-------|-------|
| `--bg` | `#0a0a0a` | Page background |
| `--surface` | `#111111` | Card backgrounds |
| `--border` | `#1f1f1f` | Borders/dividers |
| `--accent` | `#00d084` | Brand green (CTAs) |
| `--text` | `#e8e8e8` | Primary text |
| `--muted` | `#666666` | Secondary text |
| `--danger` | `#ff4d4d` | Errors |

Typography: DM Sans (UI) + DM Mono (code). The API base URL is resolved in `src/lib/config.js` (production: `https://sendr-api.onrender.com`) and validated against a host allowlist.

## Environment Variables

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `DB_URL` | Yes | PostgreSQL connection string | — |
| `REDIS_URL` | Yes | Redis connection string | — |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth client ID | — |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth client secret | — |
| `OAUTH_STATE_SECRET` | Yes | OAuth state HMAC key | — |
| `RAZORPAY_KEY_ID` | Yes | Razorpay key ID | — |
| `RAZORPAY_KEY_SECRET` | Yes | Razorpay key secret | — |
| `RAZORPAY_WEBHOOK_SECRET` | Yes* | Razorpay webhook HMAC secret | — |
| `JWT_PRIVATE_KEY_PEM` / `JWT_PRIVATE_KEY_PATH` | Yes* | RSA private key (inline PEM or path) | — |
| `JWT_PUBLIC_KEY_PEM` / `JWT_PUBLIC_KEY_PATH` | Yes* | RSA public key (inline PEM or path) | — |
| `SENDGRID_KEY` | Yes* | SendGrid API key (when provider = sendgrid) | — |
| `MAILGUN_DOMAIN` | Cond. | Mailgun domain (when provider = mailgun) | — |
| `MAILGUN_KEY` | Cond. | Mailgun API key (when provider = mailgun) | — |
| `MAILGUN_BASE_URL` | No | Mailgun API base URL | `https://api.mailgun.net/v3` |
| `SMTP_HOST` | Cond. | SMTP host (when provider = smtp) | — |
| `SMTP_PORT` | Cond. | SMTP port (when provider = smtp) | — |
| `SMTP_USER` | Cond. | SMTP username | — |
| `SMTP_PASS` | Cond. | SMTP password | — |
| `EMAIL_PROVIDER` | No | `sendgrid` \| `mailgun` \| `smtp` \| `mock` | `sendgrid` |
| `UNSUBSCRIBE_SECRET` | No | HMAC key for unsubscribe links | `change-me` |
| `FROM_EMAIL` | No | Sender email address | `noreply@example.com` |
| `FROM_NAME` | No | Sender display name | `Sendr` |
| `BACKEND_URL` | No | Public backend URL (OAuth redirect) | `http://localhost:8080` |
| `FRONTEND_URL` | No | Frontend origin (OAuth + CORS) | `http://localhost:5173` |
| `ALLOWED_ORIGINS` | No | Comma-separated CORS origins | `FRONTEND_URL` |
| `PORT` | No | Server port | `8080` |
| `APP_ENV` | No | `development` \| `production` | `development` |
| `LOG_LEVEL` | No | `debug` \| `info` \| `warn` \| `error` | `info` |
| `VITE_API_URL` | No | Frontend: backend API URL | prod/dev default |

\* Required at startup unless supplying the alternative (`*_PEM` vs `*_PATH`) or using a different provider. `RAZORPAY_WEBHOOK_SECRET` is not checked at boot, but the Razorpay webhook rejects requests if it is unset or wrong. In `production`, cookie flags become `Secure`.

## Setup

**Prerequisites:** Go 1.26+, Node 20+, Docker, OpenSSL, and the [`migrate`](https://github.com/golang-migrate/migrate) CLI.

```bash
# 1. Clone
git clone https://github.com/muhammedfazall/Sendr
cd Sendr/backend

# 2. Start Postgres + Redis (Postgres is exposed on host port 5433)
docker compose up -d

# 3. Generate RSA key pair
openssl genrsa -traditional -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# 4. Configure environment
cp .env.example .env
# Fill in: DB_URL, REDIS_URL, GOOGLE_CLIENT_*, SENDGRID_KEY,
# RAZORPAY_*, BACKEND_URL, OAUTH_STATE_SECRET, UNSUBSCRIBE_SECRET

# 5. Run migrations
migrate -path migrations -database "postgres://sendr:secret@localhost:5433/sendr?sslmode=disable" up

# 6. Start backend
go run ./cmd/server

# 7. Start frontend
cd ../frontend && npm install && npm run dev
```

From the repository root, the `Makefile` wraps the common steps:

```bash
make infra      # docker compose up -d (Postgres + Redis)
make migrate    # apply pending migrations
make backend    # go run ./cmd/server
make frontend   # npm run dev
make test       # go test ./...
make down       # stop Postgres + Redis
```

Backend tests can be run directly with `cd backend && go test ./...`.

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
  -d '{"to":["user@example.com"],"subject":"Hello","text_body":"From Sendr"}'
# → 202 {"job_id":"...","message":"email queued"}

# 4. Poll status
curl http://localhost:8080/emails/<job_id> \
  -H "Authorization: Bearer mk_live_..."
# → {"id":"...","status":"sent",...}
```

The payload supports `to`/`cc`/`bcc` (arrays), `subject`, `text_body`, `html_body`, `attachments`, `headers`, and `tags`. The legacy single-string `to` and `body`/`html` fields are accepted for backward compatibility. Subject cap: 998 chars. Body cap: 50 KB each. Recipient addresses are validated before queuing.

## CLI

A companion CLI lives in the [`sendr-cli`](../sendr-cli/README.md) repository — `sendr login`, `sendr keys`, `sendr send`, and `sendr config`.

```bash
go install github.com/muhammedfazall/sendr-cli@latest
sendr config set-url http://localhost:8080   # for self-hosted backends
sendr login
sendr keys create local
sendr send --to you@example.com --subject "Hello" --body "From the CLI"
```

## Deployment

- **Backend:** Dockerized Go API deployed to **Render** (`https://sendr-api.onrender.com`), with a static non-root Alpine runtime image (`appuser`, uid 10001).
- **Database:** **Neon** PostgreSQL; CI applies migrations with `migrate/migrate:v4.18.3`.
- **Frontend:** Built with Vite, deployable to **Vercel** (`frontend/vercel.json` SPA rewrites).
- **CI/CD:** GitHub Actions (`.github/workflows/deploy.yml`) on push to `main` — job 1 sets up Go 1.26 and runs `go test -v ./...`; job 2 applies migrations to Neon, then triggers the Render deploy hook via `curl`.

## Security

- API keys hashed with SHA-256 (constant-time comparison)
- JWT access tokens blacklisted in Redis on logout; refresh tokens rotated on every use
- OAuth state parameter (HMAC-signed, User-Agent bound) for CSRF protection
- Razorpay webhook + payment signatures verified with HMAC-SHA256 (`hmac.Equal`)
- CORS origin whitelist (validated at startup, `*` rejected)
- 1 MB request body limit, auth routes throttled to 10 concurrent requests
- Frontend API URL host allowlist (`config.js`)
- Secure/HttpOnly cookies in production; refresh cookie scoped to `/auth`
- Graceful shutdown drains in-flight HTTP requests and worker jobs

## License

MIT License. See [LICENSE](LICENSE) file.
