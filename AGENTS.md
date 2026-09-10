# Working Rules

## Execution
- Think Before Coding: inspect relevant code and one analog.
- Simplicity First: choose the smallest complete solution.
- Surgical Changes: touch only required files.
- Goal-Driven Execution: implement, verify, and finish.

## Project
- Backend: `internal`; contract: `apic.yaml`; SQL: `db/queries`; migrations: `db/migrations`.
- Use `user` for backend patterns and `profile` for form patterns.
- Work in order: SQL, `apic.yaml`, `make gen`, service, fixture, tests.
- A domain is one package under `internal/<domain>` with `service.go` implementing its `gen.Service` methods and `repository.go` behind an interface. A new domain adds one embedded field in `cmd/server/main.go`.
- Return `gen.Error(code, msg)` only for conditions the client must distinguish; return the raw `err` otherwise. Never `log.Printf` in a service.
- Never edit generated files (`internal/gen/`, `web/src/lib/gen/`). Batch SQL and `apic.yaml` edits before `make gen`.
- Frontend calls `api.<method>` from `web/src/lib/client.tsx` inside TanStack Query; JSON fields are snake_case as in `apic.yaml`.

## Safety
- A route is protected only when its group has `use: [auth]` in `apic.yaml`. New routes go in that group unless deliberately public.
- Keep tokens in `web/src/lib/auth.ts`. Never log secrets.
- Never create, drop, or reset PostgreSQL.
- Ask before auth, money, deletion, or destructive migration changes.

## Verify
- Add realistic fixtures for changed routes and states (`web/docs/fixtures`, keys are `"METHOD /path"`).
- Do not add browser E2E tests.
- Use fake Go repositories. Inject clocks when needed.
- Run targeted tests while editing.
- For UI changes, run `make docs` once.
- Run `make check` once, last.
