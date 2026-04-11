# Sendr
A email service enabling developers to authenticate, generate API keys and send emails



internal/
├── auth/
│   ├── handler.go       ← HTTP: cookies, redirects, JSON responses
│   ├── service.go       ← Business logic: OAuth flow, JWT signing
│   ├── repository.go    ← Data access: user upsert query
│   └── models.go        ← Types: GoogleUser, AuthTokens
├── apikey/
│   ├── handler.go       ← HTTP: parse body, extract claims, respond
│   ├── service.go       ← Business logic: key generation, orchestration
│   ├── repository.go    ← Data access: create, list, revoke queries
│   └── models.go        ← Types: APIKey, CreateKeyRequest/Response
├── health/
│   ├── handler.go       ← HTTP: status code mapping, JSON response
│   └── service.go       ← Logic: ping DB & Redis, return HealthStatus
├── me/
│   ├── handler.go       ← HTTP: extract claims, encode JSON (replaces old me.go)
│   └── service.go       ← Logic: return profile from claims
├── middleware/
│   └── auth.go          ← Unchanged
└── router/
    └── router.go        ← Updated: builds repo→service→hndlr chains



