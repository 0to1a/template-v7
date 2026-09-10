# Template v7

An AI-first full-stack template with a **Go 1.27, apic (REST + JSON), and PostgreSQL**
backend; a **React 19, TanStack Router (SPA), and shadcn/ui** frontend; and a
single production binary. One `apic.yaml` describes every route and generates both
the Go server plumbing (`internal/gen/api`) and the TypeScript client
(`web/src/lib/gen/api.ts`). Working rules for agents and contributors live in
[`AGENTS.md`](AGENTS.md) (symlinked as `CLAUDE.md`).

## Setup

Requirements: Go 1.27 (version pinned in [`go.mod`](go.mod)), Bun (version
pinned as `packageManager` in [`web/package.json`](web/package.json)), apic
(`go install github.com/0to1a/apic/cmd/apic@latest`), and an external
PostgreSQL. CI pins the same versions; see `.github/workflows/ci.yml`.

```bash
cp .env.example .env     # set DATABASE_URL and a >=32-byte JWT_SECRET
make bootstrap           # installs all dependencies (the only target that does)
make run                 # build the frontend, then run the single server process
```

Schema migrations are embedded in the binary and applied automatically at
server startup (up only); no target here creates, drops, or resets the
database itself.

## Make targets

| Target | Purpose |
|---|---|
| `make bootstrap` | Install/download all dependencies (Go modules, Bun packages, Playwright's Chromium). The only target that installs anything. |
| `make gen` | Regenerate code from `apic.yaml` (Go + TS) and SQL (sqlc). |
| `make check` | Done-signal: codegen, apic diff, gofmt, go vet, go test, web lint, typecheck, vitest, web build, go build. Run this once as the final gate before calling work done. |
| `make run` | Build the frontend once, then run the single Go server process. |
| `make build` | Produce `bin/server` with the SPA embedded. |
| `make docs` | Regenerate `docs/routes.json` (tracked), `docs/route-catalog.html`, and `docs/screenshots/` (both gitignored, regenerable) — no backend/DB needed. |
| `make help` | List all commands. |

## Security notes

- The seeded `admin@localhost` account accepts the static OTP `123456`
  (exact-match only). Remove or protect it before any untrusted deployment.
- Other accounts use a 5-minute TOTP derived from `JWT_SECRET`; no email
  provider is wired up unless `MAIL_URL` is set.
- An OTP can be replayed within its own 5-minute step (documented limitation).
- The bearer token is stored in `localStorage`; an XSS in this origin could
  read it.

## Notes

- `GET /health` now returns the apic envelope `{"code":0,"status":"OK","data":{"status":"ok"}}`
  rather than v6's bare `{"status":"ok"}`; adjust liveness probes that match
  the body.
