---
title: ruleport false positives here have code-level causes; buttonVariants needs cn()
author: Alvinl
date: 2026-11-24
branch: main
cost: { minutes: 10, tokens: 104834 }
id: l-493c4ddf
scope: repo
status: active
src: agent
touches:
  - web/docs/generate.ts
  - web/src/routes/_authenticated/profile.tsx
---

## Decided
- ruleport -scan . is clean (0 issues) without suppressions; every finding was fixed in code.
- typescript:S2187 "Add some tests to this file" on web/docs/generate.ts was a misclassification: ruleport treats an identifier named `context` (Mocha global) as a test file. Renamed Playwright's `context` to `browserContext`. Don't add a fake test.
- profile.tsx no longer syncs server data via setState in render (S6442): `draft ?? data?.display_name ?? ''`. Use `??` so a cleared field stays empty.
- generate.ts renders templates, routes and states with Promise.all (each has its own browser context). Fills on one page stay sequential via a reduce chain.

## Watch out
- `buttonVariants(...)` on a plain element must be wrapped in `cn()` (tailwind-merge), as Button does. Without it, `border-transparent` beats `border-border` and the outline button loses its border.
- docs screenshots profile/error.png and login/error.png differ between make docs runs even with the original sequential generator (submit timing). Don't chase byte diffs there as regressions.
