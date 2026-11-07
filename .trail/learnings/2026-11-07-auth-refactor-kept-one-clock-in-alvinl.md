---
title: Auth refactor kept one clock in JWTManager; three auth issues left open on purpose
author: Alvinl
date: 2026-11-07
branch: main
cost: { minutes: 16, tokens: 109995 }
id: l-c79f3f57
scope: repo
status: active
src: agent
touches:
  - ".github/workflows/ci.yml"
  - Makefile
  - README.md
  - cmd/server/main.go
  - cmd/server/register_frontend.go
  - docs/routes.json
  - internal/auth/delivery.go
  - internal/auth/delivery_test.go
  - internal/auth/firebase.go
  - internal/auth/jwt.go
---

## Decided
- auth.Service has no clock or secret of its own: login codes and tokens use JWTManager's `secret` and `now()`, so tests set one clock (internal/auth/service.go, jwt.go). Giving Service its own again brings back clocks that must be kept in sync.
- Special cases kept on purpose: admin@localhost static code 123456 (dev credential in README + seed migration); RequestLogin swallows delivery errors so it does not reveal whether an account exists; the `noRetry` list in web/src/lib/client.ts (the generated client has no per-route auth metadata).

## Tried and ruled out
- Deriving docs routes by parsing routeTree.gen.ts: this only moves the duplication. web/docs/generate.ts derives `auth` from the `_authenticated` directory instead.
- A Postgres test of the refresh-token SQL: needs a database, and CLAUDE.md forbids creating one.

## Watch out
- Left open for the owner to decide (auth):
  - RequestLogin sends mail synchronously only for existing accounts, with no SMTP timeout, so response time reveals whether an account exists.
  - CreateRefreshToken is `:exec`: for a soft-deleted user it inserts nothing, but the token is still returned.
  - The SMTP username is used as the From address.
- The SPA handler (internal/platform/server) only falls back to index.html when the request has `Accept: text/html`. A plain `curl /login` gets 404 by design.
