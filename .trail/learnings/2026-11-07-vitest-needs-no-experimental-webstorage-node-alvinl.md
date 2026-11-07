---
title: "Vitest needs --no-experimental-webstorage: Node 26's localStorage shadows jsdom's"
author: Alvinl
date: 2026-11-07
branch: main
cost: { minutes: 9, tokens: 229691 }
id: l-949465d9
scope: repo
status: active
src: agent
touches:
  - web/vite.config.ts
  - web/src/lib/auth.ts
---

## Decided
web/vite.config.ts passes `execArgv: ['--no-experimental-webstorage']` to vitest. Without it, Node 26's built-in `localStorage` global (undefined without --localstorage-file) shadows jsdom's, and every test touching web/src/lib/auth.ts throws "Cannot read properties of undefined (reading 'removeItem')".

## Tried and ruled out
An in-memory Map fallback inside auth.ts (the old design) — test-only code in the token module; the fix belongs in the test runner config.

## Watch out
The flag needs Node >= 22.4; older Node rejects it as an unknown option.
