# Sendr - Complete Project Documentation

## Table of Contents

1. [Project Overview](#project-overview)
2. [Technology Stack](#technology-stack)
3. [Architecture](#architecture)
4. [Database Schema](#database-schema)
5. [API Endpoints](#api-endpoints)
6. [Authentication & Authorization](#authentication--authorization)
7. [Email Pipeline](#email-pipeline)
8. [Email Queue System](#email-queue-system)
9. [Rate Limiting](#rate-limiting)
10. [Plans & Limits](#plans--limits)
11. [Payments (Razorpay)](#payments-razorpay)
12. [Frontend Architecture](#frontend-architecture)
13. [Configuration](#configuration)
14. [Capacity & Scalability](#capacity--scalability)
15. [Security Considerations](#security-considerations)
16. [Deployment](#deployment)

---

## Project Overview

**Sendr** is a developer-facing transactional email API built in Go. It provides a queued email delivery system with:

- Landing page for product introduction and user onboarding
- Google OAuth authentication
- API key-based access
- Tiered subscription plans with Razorpay payment integration
- Rate limiting per plan (daily email limits)
- Pluggable email providers (SendGrid, Mailgun, SMTP, Mock)
- Reusable, server-rendered email templates
- Automatic retries with exponential backoff
- Dead Letter Queue (DLQ) for failed emails
- Job status polling and delivery analytics (opens/clicks/bounces)
- One-click unsubscribe with HMAC-signed tokens
- Dashboard with email history, stats, API key management, and profile management

---

## Technology Stack

| Component | Technology |
|-----------|------------|
| Backend Language | Go 1.26 |
| HTTP Router | gin v1.9 |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |
| Email Providers | SendGrid (default), Mailgun, SMTP, Mock |
| Payments | Razorpay |
| JWT | RS256 (RSA 2048-bit) |
| Frontend | React 19 + Vite 8 |
| CSS | Tailwind CSS v4 |
| Containerization | Docker + Docker Compose |
| CI/CD | GitHub Actions |

---

## Architecture

The project follows **Clean Architecture** with clear separation:

```
backend/
├── cmd/
│   └── server/          # Entry point
├── internal/
│   ├── core/
│   │   ├── domain/      # Business models + domain events (User, Plan, APIKey, Job, Payment, Template, EmailEvent, Unsubscription)
│   │   ├── ports/       # Interface definitions (repositories + services)
│   │   └── services/    # Business logic (auth, apikey, email, payment, template)
│   ├── adapters/        # Infrastructure implementations
│   │   ├── userrepo/           # PostgreSQL user repository
│   │   ├── apikeyrepo/         # PostgreSQL API key repository
│   │   ├── jobrepo/            # PostgreSQL job repository (claim/retry/DLQ)
│   │   ├── planrepo/           # PostgreSQL plan repository
│   │   ├── paymentrepo/        # PostgreSQL payment repository
│   │   ├── emaileventrepo/     # PostgreSQL delivery-event repository
│   │   ├── templaterepo/       # PostgreSQL template repository
│   │   ├── unsubrepo/          # PostgreSQL unsubscribe repository
│   │   ├── ratelimit/          # Redis rate limiter (Lua script)
│   │   ├── tokenstore/         # Redis token store (refresh + blacklist)
│   │   ├── eventbus/           # In-process pub/sub for email events
│   │   └── emailsender/        # SendGrid / Mailgun / SMTP / Mock senders
│   ├── handlers/        # HTTP handlers
│   │   ├── authhandler/
│   │   ├── apikeyhandler/
│   │   ├── emailhandler/        # send, status, history, stats
│   │   ├── mehandler/
│   │   ├── paymenthandler/      # Razorpay order/verify + plan list
│   │   ├── templatehandler/     # template CRUD
│   │   ├── unsubhandler/        # one-click unsubscribe
│   │   └── webhookhandler/      # Razorpay + SendGrid webhooks
│   ├── middleware/      # HTTP middleware (JWT, API key, CORS, throttle, request ID, logger)
│   ├── worker/          # Background job processor
│   ├── router/          # Route definitions
│   ├── app/             # Bootstrap + graceful shutdown
│   └── health/          # Health check endpoints
├── pkg/
│   ├── config/          # Configuration loading + validation
│   ├── db/              # PostgreSQL (pgxpool) + Redis connections
│   ├── constants/       # Durations + error definitions
│   ├── helpers/         # API key generation + SHA-256 hashing
│   └── response/        # HTTP response helpers
├── migrations/          # Database migrations (10 pairs)
├── Dockerfile           # Multi-stage production build
└── Dockerfile.migrate   # Migration runner image

frontend/
├── src/
│   ├── pages/
│   │   ├── Landing.jsx         # Public landing page
│   │   ├── Login.jsx           # Google OAuth sign-in
│   │   ├── Callback.jsx        # OAuth callback handler
│   │   ├── Dashboard.jsx       # Main dashboard
│   │   ├── APIKeys.jsx         # API key management
│   │   ├── SendEmail.jsx       # Email composer
│   │   ├── MailHistory.jsx     # Email job history
│   │   ├── Pricing.jsx         # Plan upgrade (Razorpay)
│   │   └── Profile.jsx         # User profile
│   ├── components/
│   │   └── Layout.jsx          # Sidebar layout (authenticated)
│   ├── lib/
│   │   ├── api.js              # API client
│   │   ├── auth.jsx            # Auth provider
│   │   ├── auth-context.js     # Auth context
│   │   └── config.js           # API base URL config + host allowlist
│   ├── index.css               # Global styles + design tokens
│   ├── landing.css             # Landing page styles
│   ├── main.jsx                # React entry point
│   └── App.jsx                 # Router + route definitions
├── index.html                  # HTML entry with SEO meta tags
└── vercel.json                 # SPA rewrites
```

### Data Flow

```
User Request → API Key Validation → Rate Limit Check → Job Enqueue → 202 Accepted
                                                                         ↓
                                                              Background Worker
                                                                         ↓
                                                       Poll Jobs → Email Provider
                                                                         ↓
                                                 Update Status (sent/failed) + Event Bus
                                                                         ↓
                                      SendGrid Webhook → email_events → /emails/stats
```

---

## Database Schema

### Tables (9)

#### `plans`
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| name | TEXT | Plan name (free/pro/max) |
| daily_limit | INT | Maximum emails per day (-1 = unlimited) |
| max_api_keys | INT | Maximum API keys per user (-1 = unlimited) |
| rate_wait_secs | INT | Minimum seconds between email sends (0 = no wait) |
| price_paise | INT | Monthly price in paise (0 = free) |
| created_at | TIMESTAMPTZ | Creation timestamp |

> Seeded in migration 000004 then updated with the final limits in migration 000006.

#### `users`
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| email | TEXT | User email (unique) |
| name | TEXT | Display name |
| google_id | TEXT | Google OAuth ID (unique) |
| plan_id | UUID | Foreign key to plans |
| created_at | TIMESTAMPTZ | Creation timestamp |

#### `api_keys`
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| user_id | UUID | Foreign key to users (ON DELETE CASCADE) |
| name | TEXT | Key name for identification |
| prefix | TEXT | Public key prefix (unique, 12 hex chars) |
| hashed_key | TEXT | SHA-256 hash of secret |
| revoked | BOOLEAN | Revocation status |
| created_at | TIMESTAMPTZ | Creation timestamp |

#### `jobs`
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| user_id | UUID | Foreign key to users |
| api_key_id | UUID | Foreign key to api_keys |
| payload | JSONB | Email data (to, subject, bodies, headers, tags) |
| status | TEXT | pending/processing/sent/failed |
| retries | INT | Current retry count |
| max_retries | INT | Maximum retry attempts (3) |
| run_at | TIMESTAMPTZ | When job should run |
| locked_until | TIMESTAMPTZ | Lock expiration for workers |
| provider_message_id | TEXT | Provider-side ID (e.g. SendGrid `sg_message_id`) |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Last update timestamp |

Indexes: `idx_jobs_queue(status, run_at, locked_until)`, `idx_jobs_user_created_at`, `idx_jobs_user_status_created_at`, `idx_jobs_provider_message_id`.

#### `dlq`
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| job_id | UUID | Reference to failed job |
| payload | JSONB | Original email payload |
| error_message | TEXT | Reason for failure |
| failed_at | TIMESTAMPTZ | Failure timestamp |

#### `payments`
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| user_id | UUID | Foreign key to users (ON DELETE CASCADE) |
| razorpay_order_id | TEXT | Razorpay order ID (unique) |
| razorpay_payment_id | TEXT | Razorpay payment ID (set after payment) |
| razorpay_signature | TEXT | Razorpay signature (for verification) |
| plan_name | TEXT | Target plan name |
| amount_paise | INT | Amount charged in paise |
| currency | TEXT | Currency code (default: INR) |
| status | TEXT | created/paid/failed |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Last update timestamp |

Indexes: `idx_payments_user_id`, `idx_payments_order_id`.

#### `email_events`
| Column | Type | Description |
|--------|------|-------------|
| id | TEXT | Primary key |
| email | TEXT | Recipient email |
| event_type | TEXT | open/click/bounce/delivered/... |
| sg_event_id | TEXT | SendGrid event ID |
| sg_message_id | TEXT | SendGrid message ID (links to a job) |
| job_id | UUID | Foreign key to jobs |
| timestamp | TIMESTAMPTZ | Event time |
| metadata | JSONB | Raw event payload |
| created_at | TIMESTAMPTZ | Creation timestamp |

#### `unsubscriptions`
| Column | Type | Description |
|--------|------|-------------|
| email | TEXT | Primary key |
| reason | TEXT | Reason (default: manual) |
| created_at | TIMESTAMPTZ | Creation timestamp |

#### `templates`
| Column | Type | Description |
|--------|------|-------------|
| id | TEXT | Primary key |
| user_id | UUID | Foreign key to users |
| name | TEXT | Template name |
| subject_template | TEXT | Go template for subject |
| html_template | TEXT | Go template for HTML body |
| text_template | TEXT | Go template for text body |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Last update timestamp |

---

## API Endpoints

27 routes total.

### Public

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/health` | None | Liveness check (DB + Redis) |
| `GET` | `/plans` | None | List all available plans |
| `POST` | `/webhooks/razorpay` | HMAC signature | Razorpay payment webhook |
| `POST` | `/webhooks/sendgrid` | None | SendGrid delivery event webhook |
| `GET` | `/unsubscribe` | Token (query) | Unsubscribe confirmation page |
| `POST` | `/unsubscribe` | Token (query) | Process an unsubscribe |

### Authentication (throttled to 10 concurrent requests)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/auth/google` | None | Start OAuth flow |
| `GET` | `/auth/google/callback` | None | OAuth callback, sets auth cookies |
| `GET` | `/auth/token` | One-time cookie | Exchange auth cookie for access JWT |
| `POST` | `/auth/refresh` | Refresh cookie | Rotate access + refresh tokens |
| `POST` | `/auth/logout` | JWT | Revoke tokens |

### User Management (JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/me` | Get current user profile + plan + usage |
| `PATCH` | `/me` | Update user profile (name) |

### API Keys (JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/apikeys` | Create new API key (enforces plan limit) |
| `GET` | `/apikeys` | List user's API keys |
| `DELETE` | `/apikeys/:id` | Revoke API key |

### Emails

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/emails/send` | API Key | Queue an email (enforces rate limit) |
| `GET` | `/emails/:id` | API Key | Get job status |
| `GET` | `/emails` | JWT | List email history (`?status=&limit=&offset=`) |
| `GET` | `/emails/stats` | JWT | Aggregate delivery stats |

### Templates (JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/templates` | Create a template |
| `GET` | `/templates` | List templates |
| `GET` | `/templates/:id` | Get a template |
| `PATCH` | `/templates/:id` | Update a template |
| `DELETE` | `/templates/:id` | Delete a template |

### Payments (Razorpay, JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/payments/orders` | Create Razorpay order for plan upgrade |
| `POST` | `/payments/verify` | Verify Razorpay payment signature |

---

## Authentication & Authorization

### OAuth Flow (Google)

1. User visits `/auth/google` → server sets an HMAC-signed `oauth_state` cookie → redirected to Google
2. After Google login → callback at `/auth/google/callback`
3. Server validates state (constant-time + User-Agent bound), exchanges the code, upserts the user
4. Sets short-lived `auth_token` cookie (60s) + `refresh_token` cookie (7 days, path `/auth`)
5. Redirects to the frontend `/callback?token=...` (or a CLI local server via `redirect_uri`)

### JWT Authentication

- **Algorithm**: RS256 (RSA 2048-bit)
- **Access Token TTL**: 15 minutes
- **Refresh Token TTL**: 7 days (rotated on every refresh)
- **Issuer / Audience**: `sendr` / `sendr-api`
- **Token Storage**: refresh token in Redis (deleted on logout); access token `jti` blacklisted on logout

### API Key Authentication

Key format: `mk_live_<prefix>.<secret>`

- **Prefix**: 12 hex characters (6 random bytes), stored in DB, used for lookup
- **Secret**: 64 hex characters (32 random bytes), stored as SHA-256 hash
- Compared with `crypto/subtle.ConstantTimeCompare` (anti-timing-attack)
- Use in Authorization header: `Bearer mk_live_xxx.secret`

---

## Email Pipeline

### Providers

The sender is selected at startup by `EMAIL_PROVIDER`:

| Provider | Value | Required configuration |
|----------|-------|------------------------|
| SendGrid (default) | `sendgrid` | `SENDGRID_KEY` |
| Mailgun | `mailgun` | `MAILGUN_DOMAIN`, `MAILGUN_KEY` (+ optional `MAILGUN_BASE_URL`) |
| SMTP | `smtp` | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS` |
| Mock | `mock` | — (no actual send; useful for tests/load) |

Errors are classified as retryable (network, 5xx) or non-retryable (4xx) by `emailsender.IsRetryable`.

### Templates

- Stored per user; created/updated via `/templates`.
- Rendered server-side with Go `html/template` when a send request includes `template_id` + `template_data`.
- Rendered subject/HTML/text fill in any fields the caller leaves empty.

### Unsubscribe

- The worker attaches a signed `List-Unsubscribe` header to every outbound email.
- Token = `HMAC-SHA256(email, UNSUBSCRIBE_SECRET)`; `/unsubscribe` verifies it with `hmac.Equal`.
- Opt-outs are recorded in the `unsubscriptions` table.

### Delivery Events & Analytics

- SendGrid posts events to `/webhooks/sendgrid`; each event is stored in `email_events` and linked to a job via `sg_message_id` → `jobs.provider_message_id`.
- `GET /emails/stats` returns `{sent, delivered, opens, clicks, bounces, spam}` for the user.

---

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

- **Claiming**: `UPDATE … WHERE id IN (SELECT … FOR UPDATE SKIP LOCKED)` — safe across multiple workers, no double-send
- **Poll Interval**: every 2 seconds
- **Batch Size**: 10 jobs per poll
- **Concurrent Jobs**: 10 (semaphore-constrained)
- **Lock Duration**: 30 seconds per job
- **Zombie Recovery**: every 60 seconds (jobs stuck in `processing` with an expired lock are returned to `pending`)
- **Graceful shutdown**: on SIGINT/SIGTERM the worker drains all in-flight jobs before exit

### Retry Strategy

| Attempt | Backoff |
|---------|---------|
| 1 | 10 seconds |
| 2 | 60 seconds |
| 3 | 300 seconds (5 minutes) |

### Retryable vs Non-Retryable Errors

- **Retryable**: Network errors, 5xx from the provider
- **Non-Retryable**: 4xx errors → straight to DLQ

---

## Rate Limiting

- **Mechanism**: Redis fixed-window counter, incremented atomically via a Lua script
- **Key Format**: `rate_limit:<userID>:<YYYY-MM-DD>`, TTL 25 hours
- **Window**: UTC midnight to midnight
- **Unlimited plans** (`daily_limit = -1`) skip the limiter entirely
- **On 429**: `X-RateLimit-Reset` (unix seconds) and `Retry-After` (seconds) headers
- **Current usage**: `GET /me` returns `usage_today` and `remaining`

---

## Plans & Limits

| Plan | Daily Limit | Max API Keys | Wait Between Sends | Price (INR) |
|------|-------------|-------------|-------------------|-------------|
| free | 5 emails/day | 1 | 30 seconds | ₹0 (free forever) |
| pro | 10 emails/day | 3 | 5 seconds | ₹299/month |
| max | Unlimited | Unlimited | None | ₹999/month |

> **Note**: `daily_limit = -1` means unlimited. `max_api_keys = -1` means unlimited. `rate_wait_secs = 0` means no wait.

> **Enforcement note**: `rate_wait_secs` exists in the plan model and DB but is **not currently enforced** — only the daily limit is enforced. See `PROJECT_BRIEF.md`.

**To upgrade**: Users upgrade through the Pricing page in the dashboard. Razorpay handles payment processing, and the plan is upgraded automatically after successful payment verification (via frontend verification or the Razorpay webhook).

---

## Payments (Razorpay)

### Payment Flow

```
User clicks "Upgrade" → POST /payments/orders → Razorpay Order created
        ↓
Razorpay Checkout opens → User completes payment
        ↓
POST /payments/verify → Signature verified → Plan upgraded
        ↓ (backup)
POST /webhooks/razorpay → Webhook verifies + upgrades plan
```

### Payment Lifecycle

| Status | Description |
|--------|-------------|
| `created` | Order created, awaiting payment |
| `paid` | Payment verified, plan upgraded |
| `failed` | Payment failed or rejected |

### Signature Verification

- **Frontend verification**: `HMAC-SHA256(order_id + "|" + payment_id, RAZORPAY_KEY_SECRET)`, modeled as hex, compared with `hmac.Equal`.
- **Webhook**: `HMAC-SHA256(raw_body, RAZORPAY_WEBHOOK_SECRET)` compared against the `X-Razorpay-Signature` header.

### Idempotency

Payment verification is idempotent — if a payment is already `paid`, the plan upgrade is retried and success is returned. This prevents issues with duplicate webhook deliveries or races between frontend verification and the webhook. A payment must belong to the requesting user before an upgrade is applied.

---

## Frontend Architecture

### User Flow

```
Landing Page (/) → Login (/login) → Google OAuth → Callback (/callback) → Dashboard (/dashboard)
```

### Route Map

| Route | Component | Auth | Description |
|-------|-----------|------|-------------|
| `/` | Landing | Public | Product landing page |
| `/login` | Login | Public | Google OAuth sign-in |
| `/callback` | Callback | Public | OAuth callback handler |
| `/dashboard` | Dashboard | Protected | Main dashboard with stats |
| `/keys` | APIKeys | Protected | API key management |
| `/send` | SendEmail | Protected | Email composer |
| `/history` | MailHistory | Protected | Email job history |
| `/pricing` | Pricing | Protected | Plan comparison + Razorpay upgrade |
| `/profile` | Profile | Protected | User profile management |

### API Client

`src/lib/config.js` resolves the API base URL (`VITE_API_URL`, else `https://sendr-api.onrender.com` in production or `http://localhost:8080` in dev) and rejects any host not on the allowlist (`localhost:8080`, `127.0.0.1:8080`, `sendr.app`, `sendr-api.onrender.com`).

### Landing Page

The landing page (`/`) is the public-facing entry point with the following sections:

1. **Sticky Navbar** — Glassmorphism blur on scroll, navigation links, CTA
2. **Hero Section** — Gradient headline, subtitle, live code snippet (curl example), stats bar
3. **Trust Bar** — Tech stack badges (Go, PostgreSQL, Redis, SendGrid, Docker)
4. **Features Grid** (×6) — Queued Delivery, Automatic Retries, Secure API Keys, Real-time Status, Rate Limiting, Dead Letter Queue
5. **How it Works** (×3) — Create Account → Generate API Key → Send Emails
6. **Pricing Cards** (×3) — Free, Pro, Max plans with accurate limits
7. **CTA Section** — Final call-to-action with glowing background
8. **Footer** — Brand, links, copyright

### Design System

| Token | Value | Usage |
|-------|-------|-------|
| `--bg` | `#0a0a0a` | Page background |
| `--surface` | `#111111` | Card/panel backgrounds |
| `--border` | `#1f1f1f` | Default borders |
| `--border-hover` | `#2e2e2e` | Hover-state borders |
| `--text` | `#e8e8e8` | Primary text |
| `--muted` | `#666666` | Secondary/muted text |
| `--accent` | `#00d084` | Brand green (CTAs, highlights) |
| `--accent-dim` | `rgba(0,208,132,0.08)` | Accent background tint |
| `--accent-border` | `rgba(0,208,132,0.2)` | Accent borders |
| `--danger` | `#ff4d4d` | Error/destructive actions |

**Typography**: DM Sans (UI) + DM Mono (code)

---

## Configuration

### Environment Variables

| Variable | Required | Description | Default / Example |
|----------|----------|-------------|-------------------|
| `APP_ENV` | No | Environment (development/production) | development |
| `LOG_LEVEL` | No | debug/info/warn/error | info |
| `DB_URL` | Yes | PostgreSQL connection string | postgres://user:pass@localhost:5433/sendr |
| `REDIS_URL` | Yes | Redis connection string | redis://:pass@localhost:6379/0 |
| `JWT_PRIVATE_KEY_PEM` / `JWT_PRIVATE_KEY_PATH` | Yes* | RSA private key (inline PEM or file path) | ./private.pem |
| `JWT_PUBLIC_KEY_PEM` / `JWT_PUBLIC_KEY_PATH` | Yes* | RSA public key (inline PEM or file path) | ./public.pem |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth client ID | xxx.apps.googleusercontent.com |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth secret | GOCSPX-xxx |
| `OAUTH_STATE_SECRET` | Yes | OAuth state HMAC secret | random-string |
| `RAZORPAY_KEY_ID` | Yes | Razorpay key ID | rzp_test_xxx |
| `RAZORPAY_KEY_SECRET` | Yes | Razorpay key secret | xxx |
| `RAZORPAY_WEBHOOK_SECRET` | Yes | Razorpay webhook secret | xxx |
| `EMAIL_PROVIDER` | No | sendgrid/mailgun/smtp/mock | sendgrid |
| `SENDGRID_KEY` | Cond. | SendGrid API key (when provider = sendgrid) | SG.xxx |
| `MAILGUN_DOMAIN` / `MAILGUN_KEY` / `MAILGUN_BASE_URL` | Cond. | Mailgun config (when provider = mailgun) | — |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` | Cond. | SMTP config (when provider = smtp) | — |
| `UNSUBSCRIBE_SECRET` | No | HMAC key for unsubscribe links | change-me |
| `FROM_EMAIL` | No | Sender email address | noreply@example.com |
| `FROM_NAME` | No | Sender display name | Sendr |
| `PORT` | No | Server port | 8080 |
| `BACKEND_URL` | No | Public backend URL (OAuth redirect) | http://localhost:8080 |
| `FRONTEND_URL` | No | Frontend origin (OAuth + CORS) | http://localhost:5173 |
| `ALLOWED_ORIGINS` | No | CORS origins (comma-separated) | FRONTEND_URL |
| `VITE_API_URL` | No | Frontend: backend API URL | http://localhost:8080 |

\* Supply either the `_PEM` or `_PATH` variant. CORS origins and URLs are validated at startup; `*` is rejected.

### Generating Keys

```bash
# RSA key pair for JWT
openssl genrsa -traditional -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem
```

---

## Capacity & Scalability

### Current Limits

| Component | Current Value | Location |
|-----------|---------------|----------|
| Concurrent email jobs | 10 | `internal/worker/worker.go:66` |
| Job poll batch size | 10 | `internal/worker/worker.go:96` |
| Poll interval | 2 seconds | `internal/worker/worker.go:63` |
| Zombie reclaim | every 60 seconds | `internal/worker/worker.go:64` |
| Job lock duration | 30 seconds | `internal/adapters/jobrepo/postgres.go:54` |
| Auth throttle | 10 concurrent requests | `internal/router/router.go:94` |
| HTTP server timeouts | read 15s / write 15s / idle 60s | `internal/app/app.go:101` |
| Request body limit | 1 MB | `internal/router/router.go:48` |

### Scaling Strategies

1. **Increase Worker Concurrency**
   ```go
   // backend/internal/worker/worker.go:66
   sem := make(chan struct{}, 10) // change to 50, 100, etc.
   ```

2. **Increase Poll Batch Size**
   ```go
   // backend/internal/worker/worker.go:96
   jobs, err := w.repo.ClaimBatch(ctx, 10) // change to 20, 50, etc.
   ```

3. **Horizontal Scaling**
   - Run multiple server instances
   - `SELECT … FOR UPDATE SKIP LOCKED` already makes job claiming safe across instances
   - Load balancer in front

4. **Database Optimization**
   - Add connection pool settings
   - Add indexes on frequently queried columns
   - Consider read replicas

5. **Rate Limit Increase**
   - Create/update a custom plan row in the database
   - Update the user's `plan_id`

---

## Security Considerations

### Implemented

- **API Keys**: stored as SHA-256, compared in constant time (anti-timing-attack)
- **Tokens**: refresh tokens in Redis; access token `jti` blacklisted on logout
- **OAuth State**: HMAC-signed, User-Agent bound, compared in constant time (CSRF protection)
- **Payment Signatures**: Razorpay payment + webhook HMAC-SHA256 verified with `hmac.Equal`
- **Unsubscribe Tokens**: HMAC-SHA256 verified with `hmac.Equal`
- **Input Validation**: recipient parsing, 998-char subject cap, 50 KB body cap, 1 MB body limit
- **CORS**: validated origin allowlist (`*` rejected at startup)
- **Abuse Controls**: auth-route throttle (10 concurrent), per-plan daily rate limit, request ID + structured logging
- **Cookies**: `Secure`, `HttpOnly`, `SameSite` set appropriately; refresh cookie scoped to `/auth`

### Recommendations for Production

1. Enable HTTPS/TLS (already on Render)
2. Rotate JWT keys periodically
3. Add signature verification to the SendGrid webhook (currently unauthenticated — consider the signed-webhook / Ed25519 verification)
4. Implement IP allowlisting
5. Add request logging/audit trail and monitoring/alerting
6. Use a secrets manager (Vault, AWS Secrets Manager) instead of plain env vars

---

## Deployment

### Prerequisites

- Go 1.26+
- Node 20+
- Docker & Docker Compose
- PostgreSQL 16
- Redis 7
- OpenSSL (for JWT keys)
- migrate CLI (for database migrations)

### Quick Start

```bash
# 1. Clone repository
git clone https://github.com/muhammedfazall/Sendr
cd Sendr/backend

# 2. Start infrastructure
docker compose up -d

# 3. Generate JWT keys
openssl genrsa -traditional -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# 4. Configure environment
cp .env.example .env
# Edit .env with your values

# 5. Run migrations
migrate -path migrations -database "postgres://sendr:secret@localhost:5433/sendr?sslmode=disable" up

# 6. Start backend
go run ./cmd/server

# 7. Start frontend
cd ../frontend
npm install
npm run dev
```

### Production Deployment

- **Backend**: multi-stage `Dockerfile` builds a static, non-root Alpine image; deployed to **Render** (`https://sendr-api.onrender.com`).
- **Database**: **Neon** PostgreSQL; CI applies migrations with `migrate/migrate:v4.18.3`.
- **Frontend**: Vite build deployed to **Vercel** with SPA rewrites (`frontend/vercel.json`).
- **CI/CD** (`.github/workflows/deploy.yml`, on push to `main`):
  1. `build-and-test` — setup Go 1.26 → `go test -v ./...`
  2. `migrate-and-deploy` — apply migrations to Neon → trigger the Render deploy hook.
- **Prod Compose** (`docker-compose.prod.yml`): runs the `migrate` service to completion, then starts `api`.

### Production Considerations

- Use environment-specific config
- Set up a reverse proxy (handled by Render)
- Configure structured logging (already JSON via `log/slog`)
- Set up health checks (`GET /health`)
- Configure resource limits (Docker)
- Use managed PostgreSQL/Redis (Neon/Render)

---

## License

MIT License - See LICENSE file
