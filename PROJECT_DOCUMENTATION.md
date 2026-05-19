# Sendr - Complete Project Documentation

## Table of Contents

1. [Project Overview](#project-overview)
2. [Technology Stack](#technology-stack)
3. [Architecture](#architecture)
4. [Database Schema](#database-schema)
5. [API Endpoints](#api-endpoints)
6. [Authentication & Authorization](#authentication--authorization)
7. [Email Queue System](#email-queue-system)
8. [Rate Limiting](#rate-limiting)
9. [Plans & Limits](#plans--limits)
10. [Payments (Razorpay)](#payments-razorpay)
11. [Frontend Architecture](#frontend-architecture)
12. [Configuration](#configuration)
13. [Capacity & Scalability](#capacity--scalability)
14. [Security Considerations](#security-considerations)
15. [Deployment](#deployment)

---

## Project Overview

**Sendr** is a developer-facing transactional email API built in Go. It provides a queued email delivery system with:
- Landing page for product introduction and user onboarding
- Google OAuth authentication
- API key-based access
- Tiered subscription plans with Razorpay payment integration
- Rate limiting per plan (daily email limits + wait-between-sends)
- Automatic retries with exponential backoff
- Dead Letter Queue (DLQ) for failed emails
- Job status polling
- Dashboard with email history, API key management, and profile management

---

## Technology Stack

| Component | Technology |
|-----------|------------|
| Backend Language | Go 1.21+ |
| HTTP Router | chi/v5 |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |
| Email Provider | SendGrid |
| Payments | Razorpay |
| JWT | RS256 |
| Frontend | React 19 + Vite 8 |
| CSS | Tailwind CSS v4 |
| Containerization | Docker + Docker Compose |

---

## Architecture

The project follows **Clean Architecture** with clear separation:

```
backend/
├── cmd/
│   └── server/          # Entry point
├── internal/
│   ├── core/
│   │   ├── domain/      # Business models (User, Plan, APIKey, Job, Payment)
│   │   ├── ports/       # Interface definitions
│   │   └── services/    # Business logic
│   ├── adapters/        # Infrastructure implementations
│   │   ├── userrepo/           # PostgreSQL user repository
│   │   ├── apikeyrepo/         # PostgreSQL API key repository
│   │   ├── jobrepo/            # PostgreSQL job repository
│   │   ├── planrepo/           # PostgreSQL plan repository
│   │   ├── paymentrepo/        # PostgreSQL payment repository
│   │   ├── ratelimit/          # Redis rate limiter
│   │   ├── tokenstore/         # Redis token store
│   │   └── emailsender/        # SendGrid adapter
│   ├── handlers/        # HTTP handlers
│   │   ├── authhandler/
│   │   ├── apikeyhandler/
│   │   ├── emailhandler/
│   │   ├── mehandler/
│   │   ├── paymenthandler/     # Razorpay order/verify
│   │   └── webhookhandler/     # Razorpay webhook receiver
│   ├── middleware/      # HTTP middleware (JWT, CORS, API key, etc.)
│   ├── worker/          # Background job processor
│   ├── router/          # Route definitions
│   └── health/          # Health check endpoints
├── pkg/
│   ├── config/          # Configuration loading
│   ├── db/              # Database connections
│   ├── constants/       # Error definitions
│   └── response/        # HTTP response helpers
└── migrations/          # Database migrations (6 migrations)

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
│   │   └── config.js           # API base URL config
│   ├── index.css               # Global styles + design tokens
│   ├── landing.css             # Landing page styles
│   ├── main.jsx                # React entry point
│   └── App.jsx                 # Router + route definitions
└── index.html                  # HTML entry with SEO meta tags
```

### Data Flow

```
User Request → API Key Validation → Rate Limit Check → Job Enqueue → 202 Accepted
                                                                        ↓
                                                             Background Worker
                                                                        ↓
                                                            Poll Jobs → SendGrid
                                                                        ↓
                                                            Update Status (sent/failed)
```

---

## Database Schema

### Tables

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
| user_id | UUID | Foreign key to users |
| name | TEXT | Key name for identification |
| prefix | TEXT | Public key prefix (mk_live_xxx) |
| hashed | TEXT | SHA-256 hash of secret |
| revoked | BOOLEAN | Revocation status |
| created_at | TIMESTAMPTZ | Creation timestamp |

#### `jobs`
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| user_id | UUID | Foreign key to users |
| api_key_id | UUID | Foreign key to api_keys |
| payload | JSONB | Email data (to, subject, body) |
| status | TEXT | pending/processing/sent/failed |
| retries | INT | Current retry count |
| max_retries | INT | Maximum retry attempts (3) |
| run_at | TIMESTAMPTZ | When job should run |
| locked_until | TIMESTAMPTZ | Lock expiration for workers |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Last update timestamp |

#### `dlq`
| Column | Type | Description |
|--------|------|-------------|
| job_id | UUID | Reference to failed job |
| payload | JSONB | Original email payload |
| error_message | TEXT | Reason for failure |

#### `payments`
| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| user_id | UUID | Foreign key to users |
| razorpay_order_id | TEXT | Razorpay order ID (unique) |
| razorpay_payment_id | TEXT | Razorpay payment ID (set after payment) |
| razorpay_signature | TEXT | Razorpay signature (for verification) |
| plan_name | TEXT | Target plan name |
| amount_paise | INT | Amount charged in paise |
| currency | TEXT | Currency code (default: INR) |
| status | TEXT | created/paid/failed |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Last update timestamp |

---

## API Endpoints

### Health & Public

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/health` | None | Liveness check |
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
| `GET` | `/me` | JWT | Get current user profile + plan |
| `PATCH` | `/me` | JWT | Update user profile |

### API Keys

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/apikeys` | JWT | Create new API key (enforces plan limit) |
| `GET` | `/apikeys` | JWT | List user's API keys |
| `DELETE` | `/apikeys/{id}` | JWT | Revoke API key |

### Emails

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/emails/send` | API Key | Queue an email (enforces rate limit + wait) |
| `GET` | `/emails/{id}` | API Key | Get job status |
| `GET` | `/emails` | JWT | List user's email history |

### Payments (Razorpay)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/payments/orders` | JWT | Create Razorpay order for plan upgrade |
| `POST` | `/payments/verify` | JWT | Verify Razorpay payment signature |
| `POST` | `/webhooks/razorpay` | None | Razorpay webhook (signature-verified) |

---

## Authentication & Authorization

### OAuth Flow (Google)

1. User visits `/auth/google` → redirected to Google
2. After Google login → callback at `/auth/google/callback`
3. Server exchanges code for tokens, creates/updates user
4. Returns JWT access token + refresh token

### JWT Authentication

- **Algorithm**: RS256 (RSA)
- **Access Token TTL**: 15 minutes
- **Refresh Token TTL**: 30 days
- **Token Storage**: Redis (blacklist on logout)

### API Key Authentication

Key format: `mk_live_<prefix>.<secret>`

- **Prefix**: 8 characters, publicly visible
- **Secret**: 32 characters, stored as SHA-256 hash
- Use in Authorization header: `Bearer mk_live_xxx.secret`

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

- **Poll Interval**: Every 2 seconds
- **Batch Size**: 10 jobs per poll
- **Concurrent Jobs**: 10 (configurable via semaphore)
- **Lock Duration**: 30 seconds per job
- **Zombie Recovery**: Every 60 seconds

### Retry Strategy

| Attempt | Backoff |
|---------|---------|
| 1 | 10 seconds |
| 2 | 60 seconds |
| 3 | 300 seconds (5 minutes) |

### Retryable vs Non-Retryable Errors

- **Retryable**: Network errors, 5xx from SendGrid
- **Non-Retryable**: 4xx errors, invalid credentials, rate limits

---

## Rate Limiting

- **Mechanism**: Redis fixed-window counter
- **Key Format**: `rate_limit:<userID>:<YYYY-MM-DD>`
- **Window**: UTC midnight to midnight
- **Per Plan Limits**: See Plans section
- **Response Headers**:
  - `X-RateLimit-Remaining`
  - `X-RateLimit-Reset`
  - `Retry-After` (on 429)

---

## Plans & Limits

| Plan | Daily Limit | Max API Keys | Wait Between Sends | Price (INR) |
|------|-------------|-------------|-------------------|-------------|
| free | 5 emails/day | 1 | 30 seconds | ₹0 (free forever) |
| pro | 10 emails/day | 3 | 5 seconds | ₹299/month |
| max | Unlimited | Unlimited | None | ₹999/month |

> **Note**: `daily_limit = -1` means unlimited. `max_api_keys = -1` means unlimited. `rate_wait_secs = 0` means no wait.

**To upgrade**: Users upgrade through the Pricing page in the dashboard. Razorpay handles payment processing, and the plan is upgraded automatically after successful payment verification (via frontend verification or webhook).

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

### Idempotency

Payment verification is idempotent — if a payment is already marked as `paid`, the plan upgrade is retried and success is returned. This prevents issues with duplicate webhook deliveries or race conditions between frontend verification and webhook.

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

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `APP_ENV` | No | Environment (development/production) | development |
| `DB_URL` | Yes | PostgreSQL connection string | postgres://user:pass@localhost:5433/sendr |
| `REDIS_URL` | Yes | Redis connection string | redis://:pass@localhost:6379/0 |
| `JWT_PRIVATE_KEY_PATH` | Yes | Path to private key | ./private.pem |
| `JWT_PUBLIC_KEY_PATH` | Yes | Path to public key | ./public.pem |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth client ID | xxx.apps.googleusercontent.com |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth secret | GOCSPX-xxx |
| `OAUTH_STATE_SECRET` | Yes | OAuth state encryption secret | random-string |
| `SENDGRID_KEY` | Yes | SendGrid API key | SG.xxx |
| `FROM_EMAIL` | Yes | Sender email address | noreply@example.com |
| `FROM_NAME` | Yes | Sender display name | Sendr |
| `RAZORPAY_KEY_ID` | Yes | Razorpay key ID | rzp_test_xxx |
| `RAZORPAY_KEY_SECRET` | Yes | Razorpay key secret | xxx |
| `RAZORPAY_WEBHOOK_SECRET` | Yes | Razorpay webhook secret | xxx |
| `PORT` | No | Server port (default: 8080) | 8080 |
| `FRONTEND_URL` | No | Frontend origin | http://localhost:5173 |
| `ALLOWED_ORIGINS` | No | CORS origins (comma-separated) | http://localhost:5173 |
| `VITE_API_URL` | Yes | Frontend env: backend API URL | http://localhost:8080 |

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
| Concurrent email jobs | 10 | worker.go:38 |
| Job poll batch size | 10 | worker.go:68 |
| Poll interval | 2 seconds | worker.go:35 |
| DB connections | ~100 (PostgreSQL default) | PostgreSQL config |
| Auth rate limit | 10 req/second | router.go:60 |

### Scaling Strategies

1. **Increase Worker Concurrency**
   ```go
   // backend/internal/worker/worker.go line 38
   sem := make(chan struct{}, 10) // change to 50, 100, etc.
   ```

2. **Increase Poll Batch Size**
   ```go
   // backend/internal/worker/worker.go line 68
   jobs, err := w.repo.ClaimBatch(ctx, 10) // change to 20, 50, etc.
   ```

3. **Horizontal Scaling**
   - Run multiple server instances
   - Use Redis for job coordination
   - Load balancer in front

4. **Database Optimization**
   - Add connection pool settings
   - Add indexes on frequently queried columns
   - Consider read replicas

5. **Rate Limit Increase**
   - Create custom plan in database
   - Update user's plan_id

---

## Security Considerations

### Implemented

- **Password Hashing**: API keys stored as SHA-256 (with salt prefix)
- **Token Blacklisting**: Revoked tokens stored in Redis
- **OAuth State**: CSRF protection with state parameter
- **Input Validation**: Email format, body size limits
- **CORS**: Configurable origin whitelist
- **HTTPS**: Enforced in production

### Recommendations for Production

1. Enable HTTPS/TLS
2. Rotate JWT keys periodically
3. Set up SendGrid webhooks for delivery status
4. Implement IP whitelisting
5. Add request logging/audit trail
6. Use secrets management (Vault, AWS Secrets Manager)
7. Set up monitoring and alerting

---

## Deployment

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 16
- Redis 7
- OpenSSL (for JWT keys)
- migrate CLI (for database migrations)

### Quick Start

```bash
# 1. Clone repository
git clone https://github.com/muhammedfazall/Sendr
cd Sendr

# 2. Start infrastructure
docker compose up -d

# 3. Generate JWT keys
openssl genrsa -traditional -out backend/private.pem 2048
openssl rsa -in backend/private.pem -pubout -out backend/public.pem

# 4. Configure environment
cp backend/.env.example backend/.env
# Edit .env with your values

# 5. Run migrations
migrate -path backend/migrations -database "postgres://sendr:secret@localhost:5433/sendr?sslmode=disable" up

# 6. Start backend
cd backend
go run ./cmd/server

# 7. Start frontend
cd ../frontend
npm install
npm run dev
```

### Production Considerations

- Use environment-specific config
- Set up reverse proxy (nginx)
- Configure logging (structured JSON)
- Set up health checks
- Configure resource limits (Docker)
- Use managed PostgreSQL/Redis if possible

---

## License

MIT License - See LICENSE file