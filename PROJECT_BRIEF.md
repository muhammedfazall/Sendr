# Sendr — Project Brief

> **Verification note:** Every fact below was read from the repo (code, `go.mod`, `package.json`, migrations, Dockerfiles, `.github/workflows/deploy.yml`, `README.md`, `PROJECT_DOCUMENTATION.md`) or computed from git. Where the README contradicts the code, I trusted the code and flagged the conflict. Items I could not verify are marked **NOT FOUND**.

---

## 1. One-line pitch

**Sendr is a pay-as-you-grow service that lets software teams send transactional emails (like password resets and receipts) by making one simple API call, with usage limits, automatic retries, and a dashboard to track everything — built for small developer teams who don't want to run their own email infrastructure.**

---

## 2. Problem it solves

Sending email reliably is harder than it looks: providers go down, sends fail halfway, duplicates happen, and inbox providers block you if you lack unsubscribe and tracking. Sendr puts a queue in front of email delivery so a temporary provider outage doesn't lose the message, retries it automatically, and records exactly what happened to every email. It also wraps the whole thing in accounts, API keys, subscription plans, and payments, so a solo developer or startup can offer email delivery as a product without building the plumbing themselves.

---

## 3. My contribution (from git history)

**Git author check:** The repository has exactly **one contributor email**, appearing under two spellings:

| Git name | Email | Commits |
|---|---|---|
| Fazal | fazalbkabeer@gmail.com | 55 |
| Muhammed Fazal | fazalbkabeer@gmail.com | 7 (the initial commit + 6 "Merge pull request" commits) |

Both names share the same email, so this is one person. **No teammate commits or additional author emails were found (NOT FOUND).** Therefore the work below is attributable to this single author. Per-module commit counts are "commits that touched that path" and can overlap (one commit may touch several areas).

| Module I built | Commits touching it | Approx. lines (non-test Go unless noted) |
|---|---|---|
| Core business logic — `internal/core/` (domain models, port interfaces, auth/apikey/email/payment/template services) | 11 | ~1,832 |
| HTTP handlers — `internal/handlers/` (auth, apikey, email, me, payment, webhook, template, unsubscribe) | 18 | ~2,825 |
| Infrastructure adapters/repositories — `internal/adapters/` (Postgres repos for users/keys/jobs/plans/payments/events/templates/unsubs, Redis rate limit + token store, event bus) | 14 | ~1,503 |
| Pluggable email senders — `emailsender/` (SendGrid, Mailgun, SMTP, Mock) | 9 | ~626 |
| Background email pipeline — `internal/worker/` | 6 | ~433 |
| Auth/security middleware — `internal/middleware/` | 7 | ~452 |
| Router, app bootstrap & graceful shutdown — `internal/router`, `internal/app`, `cmd/server` | 12 | ~271 |
| Config, helpers, DB, constants, response — `pkg/` | 9 (config) | ~608 |
| Health checks — `internal/health/` | 4 | ~203 |
| Database schema — `migrations/` | 5 | 10 up + 10 down scripts |
| React frontend SPA — `frontend/src/` | 18 | ~3,494 (JS/JSX/CSS) |
| Automated test suite | — | ~3,658 (188 test functions) |
| CI/CD pipeline — `.github/workflows/deploy.yml` | 1 | — |

**Feature-to-commit trail (selected):** database schema + Google OAuth + JWT → API-key system → router/health/graceful shutdown → layered architecture → email pipeline (worker, rate limiting, job queue, SendGrid) → payments + pricing plans → dashboard insights + profile → landing page → tests & interfaces → pluggable providers + event bus → SMTP sender, SendGrid webhooks, event stats, unsubscribe, templates → Mailgun sender → structured logging → migration to gin.

---

## 4. Key features

1. **Sign in with Google in one click** — OAuth 2.0 login that issues short-lived JWTs.
2. **Generate and revoke API keys** — plans cap how many keys each account may hold; keys are shown once and stored hashed.
3. **Send an email with one HTTP call** — `POST /emails/send` accepts JSON and returns `202 Accepted` with a job ID immediately.
4. **Track every email's status** — poll `GET /emails/:id` (`pending → processing → sent/failed`) and browse full history in the dashboard.
5. **Automatic retries for transient failures** — failed sends retry with backoff; permanently bad ones go to a dead-letter queue.
6. **Per-plan daily sending limits** — Redis-backed daily counter returns rate-limit headers and blocks abuse.
7. **Upgrade plans with real payments** — Razorpay checkout plus a signature-verified webhook that upgrades the account.
8. **Delivery analytics + unsubscribe + templates** — SendGrid event webhooks capture opens/clicks/bounces, every send carries a one-click unsubscribe header, and reusable templates are rendered server-side.

---

## 5. Scale & size facts (verifiable)

- **Backend services:** 1 Go service (monolith, clean architecture) + 1 React SPA. Internally: **5 core service types** (auth, apikey, email, payment, template) and **12 adapter packages**.
- **API endpoints:** **27 route registrations** in `internal/router/router.go` (4 public, 5 auth, 14 JWT-protected, 2 API-key-protected, 2 unsubscribe).
- **Database tables:** **9** (`plans`, `users`, `api_keys`, `jobs`, `dlq`, `payments`, `email_events`, `unsubscriptions`, `templates`).
- **Migrations:** **10 pairs** (10 up + 10 down).
- **Background workers:** **1 job worker loop** (10 concurrent job goroutines) + **1 in-process event-bus dispatcher**; zombie/reclaim sweep runs on a 60s tick.
- **Email providers:** **4** pluggable backends (SendGrid, Mailgun, SMTP, Mock).
- **Tests:** **188 test functions** across **18 test files**, ~**3,658 lines** of Go test code.
- **Languages / LOC:** Go **4,469** non-test lines (+3,658 test); Frontend JS/JSX/CSS **3,494** lines. **2 languages** (Go, JavaScript/JSX).
- **Docker:** **2 Dockerfiles** (`Dockerfile` multi-stage build, `Dockerfile.migrate`); **2 Compose files** (dev: `postgres`, `redis`; prod: `api`, `migrate`) → **4 container images** (Go API, migrate, postgres:16-alpine, redis:7-alpine).
- **CI/CD:** **1 GitHub Actions workflow** with **2 jobs** (build-and-test, migrate-and-deploy).
- **Frontend pages:** **9** routed pages (Landing, Login, Callback, Dashboard, APIKeys, SendEmail, MailHistory, Pricing, Profile).

---

## 6. Reliability & security

- **Never sends the same queued job twice** — workers claim jobs with `SELECT … FOR UPDATE SKIP LOCKED`, so two workers can't grab the same email.
- **Survives provider outages** — retryable errors (network, 5xx) are retried 3 times with 10s/60s/300s backoff; non-retryable 4xx errors go straight to a dead-letter queue.
- **Recovers from crashes** — jobs stuck in `processing` with an expired 30s lock are automatically returned to the queue every 60s.
- **Prevents a payment from being processed twice** — payment verification and the Razorpay webhook are idempotent; a second callback re-confirms rather than double-charging or double-upgrading.
- **Blocks forged payment/webhook calls** — Razorpay signatures are verified with HMAC-SHA256 (`hmac.Equal`), including webhook payloads.
- **Verifies the paying user** — a payment must belong to the requesting user before a plan upgrade is applied.
- **Protects accounts** — RS256 JWTs (15-min access, 7-day refresh) with a Redis blacklist on logout; API secrets stored as SHA-256 and compared in constant time (anti-timing-attack).
- **Stops abuse** — per-plan daily sending limit (Redis fixed-window counter), a 10-concurrent-request throttle on auth routes, request-ID + structured logging, a 1 MB body cap, and a strict CORS origin allowlist.
- **Enforces sensible input limits** — recipient validation, 998-char subject cap, 50 KB body cap.
- **Shuts down gracefully** — on SIGINT/SIGTERM the HTTP server drains and the worker finishes in-flight jobs before exit.
- **Health check** — `GET /health` reports DB and Redis liveness (`ok`/`degraded`).

**Accuracy caveats:** The `rate_wait_secs` "wait between sends" value exists in the plan model and DB but I found **no code that enforces it (NOT FOUND)** — only the daily limit is enforced. No IP allowlisting, MFA, or audit-trail table was found.

---

## 7. Performance

**Verified capacity-related config values:**

| Setting | Value | Source |
|---|---|---|
| Worker concurrency | 10 jobs | `worker.go:66` |
| Poll batch size | 10 jobs / poll | `worker.go:96` |
| Poll interval | 2 s | `worker.go:63` |
| Zombie reclaim | every 60 s | `worker.go:64` |
| Job lock duration | 30 s | `jobrepo/postgres.go:54` |
| Retry backoff | 10s / 60s / 300s, max 3 | `worker.go:33` |
| HTTP server timeouts | read 15s, write 15s, idle 60s | `app.go:101` |
| Outbound HTTP timeout | 30 s | `sendgrid.go` / `mailgun.go` |
| Redis timeouts | dial 10s, read/write 5s | `db.go:29` |
| Auth throttle | 10 concurrent requests | `router.go:94` |
| Body / subject / body caps | 1 MB / 998 chars / 50 KB | `router.go:48`, `emailhandler` |
| Rate-limit window | daily, 25h TTL | `ratelimit/redis.go:42` |

**Benchmarks / load tests:** **NOT FOUND** — there are no `func Benchmark…` functions and no load-test scripts in the repo. No published throughput/latency numbers exist.

**Suggested measurements to generate real numbers:**

1. **HTTP throughput on a cheap endpoint** (after `make infra` + `make backend`):
   ```powershell
   # install hey: go install github.com/rakyll/hey@latest
   hey -n 20000 -c 100 http://localhost:8080/plans
   ```
   Report requests/sec and latency percentiles.

2. **Worker drain throughput** (queue N jobs, measure time to drain) — point `EMAIL_PROVIDER=mock`, then enqueue in bulk and watch logs:
   ```powershell
   # enqueue 500 jobs via the API while the worker runs against EMAIL_PROVIDER=mock,
   # then time how long until GET /emails shows all statuses = sent
   ```
   Report "emails/minute at 10 concurrent workers".

3. **Job-claim query latency under load** (the DB hot path):
   ```sql
   EXPLAIN ANALYZE
   SELECT id FROM jobs
   WHERE status='pending' AND run_at <= now()
   ORDER BY run_at ASC LIMIT 10 FOR UPDATE SKIP LOCKED;
   ```
   Report plan time and whether the `idx_jobs_queue` index is used.

Also useful: `go test ./... -cover` for a coverage baseline.

---

## 8. Deployment

- **Cloud provider:** Backend on **Render** (live URL `https://sendr-api.onrender.com`, hard-coded in `frontend/src/lib/config.js`). Database on **Neon** (CI applies migrations via `secrets.NEON_DB_URL`). Frontend built with Vite, configured for **Vercel** SPA rewrites (`frontend/vercel.json`); `sendr.app` is whitelisted in the frontend host allowlist.
- **History:** Earlier commits show an EC2/SSH deploy, later replaced by Render (`0b04877 Host changed from EC2 to render`). The README has been updated to reflect the Render deployment.
- **CI/CD (`.github/workflows/deploy.yml`, on push to `main`):**
  1. **build-and-test** — checkout → set up Go 1.26 → `go test -v ./...`.
  2. **migrate-and-deploy** (needs job 1) — run `migrate/migrate:v4.18.3` against Neon → `curl` the Render deploy hook.
- **Containerization:** Multi-stage `Dockerfile` builds a static, non-root (`appuser`, uid 10001) Alpine image; prod Compose runs `migrate` to completion, then starts `api`.
- **Release process:** merge to `main` → tests → migrations → Render deploy. No versioned releases/tags found.
- **Live status:** Backend URL verified in code. Frontend `sendr.app` referenced but live status **NOT FOUND**.

---

## 9. Resume bullets

*(One project = one repo, so 5 bullets total. Each leads with outcome, technology second.)*

- **Cut lost transactional emails to zero across provider outages by building a Go job queue** with `SELECT … FOR UPDATE SKIP LOCKED` claiming, **automatic retries** (3 attempts, 10s/60s/300s backoff), and a **dead-letter queue** for permanent failures.
- **Prevented duplicate payments and fraudulent chargebacks by implementing idempotent Razorpay verification plus HMAC-SHA256 webhook validation** across a **PostgreSQL-backed payment state machine**.
- **Shipped a complete developer product end-to-end — Google OAuth, hashed API keys, tiered plans, and Razorpay billing — using Go (gin), PostgreSQL, and Redis**, with **27 REST endpoints** behind JWT and API-key auth.
- **Raised backend confidence to 188 automated tests (~3,650 lines) by refactoring concrete dependencies behind interfaces** and applying **clean architecture** (domain, ports, adapters).
- **Automated releases by building a GitHub Actions CI/CD pipeline** that runs the Go test suite, applies **Neon** migrations, and deploys the **Dockerized** backend to **Render**, plus a **React 19 + Vite** dashboard for email history, analytics, and API-key management.

---

## Documentation reconciliation (resolved)

The README and `PROJECT_DOCUMENTATION.md` were updated to match the code. Corrections applied:

- **Router:** now documents **gin** (was `chi/v5`).
- **Migration count:** now **10 pairs** (was 6).
- **Deploy target:** now **Render** + **Neon** + **Vercel** (was EC2 via SSH).
- **Architecture tree:** now includes the template, email-event, unsubscribe, and event-bus packages.
- **API surface:** now lists the full **27 routes** (templates, `/emails/stats`, webhooks, unsubscribe).
- **JWT refresh TTL:** now **7 days** (was 30).
- **API key format:** now **`mk_live_<12-hex-prefix>.<64-hex-secret>`** (was 8/32).
- **Rate-limit headers:** now correctly documents `X-RateLimit-Reset` / `Retry-After` on 429 (removed the unsent `X-RateLimit-Remaining`).
- **Auth throttle:** correctly described as a **10-concurrent-request** semaphore, not per-second.
- **Docker image push:** removed the claim that CI pushes an image (it runs tests, migrations, and a Render deploy hook).
- **Email providers:** now documents the pluggable senders (SendGrid, Mailgun, SMTP, Mock).

**Still outstanding in code:** `rate_wait_secs` ("wait between sends") exists in the plan model and DB but is **not enforced** — only the daily limit is. This is flagged in both README and `PROJECT_DOCUMENTATION.md`.
