# Sendr Frontend

The React single-page dashboard for [Sendr](../README.md), the transactional email API. It covers Google sign-in, email composition and history, API-key management, plan upgrades, and profile settings.

## Stack

| Concern | Technology |
|---------|------------|
| Framework | React 19.2 |
| Build tool | Vite 8.0 |
| Routing | React Router 7.14 (`react-router-dom`) |
| Styling | Tailwind CSS v4 (`@tailwindcss/vite`) + CSS design tokens |
| HTTP | Native `fetch` wrapper (`src/lib/api.js`) |
| Linting | ESLint 10 (`eslint-plugin-react-hooks`, `eslint-plugin-react-refresh`) |
| Hosting | Vercel (SPA rewrites via `vercel.json`) |

## Getting Started

**Prerequisites:** Node 20+ and a running Sendr backend (default `http://localhost:8080`).

```bash
npm install
npm run dev      # start the Vite dev server (http://localhost:5173)
```

The repository's `.env` points at the hosted API:

```bash
VITE_API_URL=https://sendr-api.onrender.com
```

For local development, create `.env.local` (it takes precedence) and point it at your backend:

```bash
# .env.local
VITE_API_URL=http://localhost:8080
```

The backend's `FRONTEND_URL`/`ALLOWED_ORIGINS` must include `http://localhost:5173` for OAuth redirects and CORS to work.

## Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start the Vite dev server |
| `npm run build` | Production build to `dist/` |
| `npm run preview` | Preview the production build locally |
| `npm run lint` | Run ESLint |

## Routes

| Route | Component | Auth | Description |
|-------|-----------|------|-------------|
| `/` | `Landing` | Public | Product landing page |
| `/login` | `Login` | Public | Google OAuth sign-in |
| `/callback` | `Callback` | Public | Reads `?token=` from the OAuth redirect and stores the session |
| `/dashboard` | `Dashboard` | Protected | Usage + delivery stats overview |
| `/keys` | `APIKeys` | Protected | API key creation, listing, revocation |
| `/send` | `SendEmail` | Protected | Email composer with delivery polling |
| `/history` | `MailHistory` | Protected | Paginated email history with status filter |
| `/pricing` | `Pricing` | Protected | Plan comparison + Razorpay upgrade |
| `/profile` | `Profile` | Protected | User profile |

Unknown routes redirect to `/`. Protected routes redirect to `/login` when there is no session; an expired session shows a "Session expired" screen instead of looping through login.

## Authentication & Session

- The access JWT is stored in `sessionStorage` (cleared when the tab closes).
- The refresh token lives in an HttpOnly cookie scoped to `/auth`; requests use `credentials: 'include'`.
- `api.js` refreshes the token automatically when the JWT is within 60 seconds of expiry and retries a request once after a `401`.
- If refresh fails, local state is cleared and a `sendr:session-expired` event shows the session-expired screen.
- Logout calls `POST /auth/logout` (best effort) and always clears local state.

## API Base URL

`src/lib/config.js` resolves the backend URL and rejects any host not on the allowlist:

- Production default: `https://sendr-api.onrender.com`
- Development default: `http://localhost:8080`
- Override: `VITE_API_URL` (must be `http` or `https`)
- Allowed hosts: `localhost:8080`, `127.0.0.1:8080`, `sendr.app`, `sendr-api.onrender.com`

Adding a new deployment host requires updating the `allowedHosts` set in `config.js`.

## Backend Calls

`src/lib/api.js` wraps the endpoints used by the UI:

| Method | Endpoint | Used by |
|--------|----------|---------|
| `GET` | `/me` | Dashboard, Profile, Layout |
| `PATCH` | `/me` | Profile |
| `GET`/`POST` | `/apikeys` | APIKeys |
| `DELETE` | `/apikeys/:id` | APIKeys |
| `POST` | `/emails/send` | SendEmail |
| `GET` | `/emails/:id` | SendEmail polling |
| `GET` | `/emails` | MailHistory |
| `GET` | `/emails/stats` | Dashboard |
| `GET` | `/plans` | Pricing |
| `POST` | `/payments/orders` | Pricing |
| `POST` | `/payments/verify` | Pricing |
| `POST` | `/auth/logout` | Auth provider |

## Project Layout

```
public/              # favicon.svg + icons.svg sprite
src/
├── pages/           # Route components
├── components/      # Layout + shared UI
├── assets/          # Landing imagery
├── lib/
│   ├── api.js           # Fetch wrapper, token refresh, session-expiry event
│   ├── auth.jsx         # AuthProvider (token storage, refresh, logout)
│   ├── auth-context.js  # React context + useAuth hook
│   └── config.js        # API base URL resolution + host allowlist
├── index.css        # Design tokens + Tailwind entry
├── landing.css      # Landing-page-specific styles
├── App.css          # Shared component styles
├── App.jsx          # Router + route definitions
└── main.jsx         # React entry point
```

## Design System

Tokens are defined in `src/index.css`:

| Token | Value | Usage |
|-------|-------|-------|
| `--bg` | `#0a0a0a` | Page background |
| `--surface` | `#111111` | Card backgrounds |
| `--border` | `#1f1f1f` | Borders/dividers |
| `--accent` | `#00d084` | Brand green (CTAs) |
| `--text` | `#e8e8e8` | Primary text |
| `--muted` | `#666666` | Secondary text |
| `--danger` | `#ff4d4d` | Errors |

Typography: DM Sans for UI, DM Mono for code and numeric values.

## Deployment

Build with `npm run build` and deploy the `dist/` output. `vercel.json` configures SPA rewrites so client-side routes resolve to `index.html`. Set `VITE_API_URL` in the hosting provider if the backend is not the default Render URL.
