# Sendr Frontend

The React single-page dashboard for [Sendr](../README.md), the transactional email API.

## Stack

| Concern | Technology |
|---------|------------|
| Framework | React 19 |
| Build tool | Vite 8 |
| Routing | React Router 7 |
| Styling | Tailwind CSS v4 (`@tailwindcss/vite`) |
| Linting | ESLint 10 (`eslint-plugin-react-hooks`, `react-refresh`) |
| Hosting | Vercel (SPA rewrites via `vercel.json`) |

## Getting Started

**Prerequisites:** Node 20+

```bash
npm install
npm run dev      # start the Vite dev server (http://localhost:5173)
```

The backend must be running at `http://localhost:8080` (default). Point the app elsewhere with `VITE_API_URL`:

```bash
# .env.local
VITE_API_URL=http://localhost:8080
```

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
| `/callback` | `Callback` | Public | OAuth callback handler |
| `/dashboard` | `Dashboard` | Protected | Stats overview |
| `/keys` | `APIKeys` | Protected | API key management |
| `/send` | `SendEmail` | Protected | Email composer |
| `/history` | `MailHistory` | Protected | Email history |
| `/pricing` | `Pricing` | Protected | Plan comparison + Razorpay upgrade |
| `/profile` | `Profile` | Protected | User profile |

## Project Layout

```
src/
├── pages/           # Route components
├── components/      # Layout + shared UI
├── lib/
│   ├── api.js           # Fetch wrapper around the backend API
│   ├── auth.jsx         # AuthProvider (token storage, refresh, logout)
│   ├── auth-context.js  # React context + useAuth hook
│   └── config.js        # API base URL resolution + host allowlist
├── index.css        # Design tokens + Tailwind entry
├── landing.css      # Landing-page-specific styles
├── App.jsx          # Router + route definitions
└── main.jsx         # React entry point
```

## API Base URL

`src/lib/config.js` resolves the backend URL and rejects any host not on the allowlist:

- Production default: `https://sendr-api.onrender.com`
- Development default: `http://localhost:8080`
- Override: `VITE_API_URL`
- Allowed hosts: `localhost:8080`, `127.0.0.1:8080`, `sendr.app`, `sendr-api.onrender.com`

## Deployment

Build with `npm run build` and deploy the `dist/` output. `vercel.json` configures SPA rewrites so client-side routes resolve to `index.html`.
