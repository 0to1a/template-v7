# Working Rules

- Read one analog first (`user` for backend, `profile` for form patterns) and mirror its shape.
- Ambiguous spec → follow the analog and state the assumption in your summary. Ask only for auth, money, deletion, or destructive migrations.

## Project
- Backend `internal`, contract `apic.yaml`, SQL `db/queries`, migrations `db/migrations/NNNNN_<plural>.sql` (sequential). Domain packages singular.
- Order: SQL → `apic.yaml` → `make gen` (batch edits first) → service → fixture → tests. New domain = one embedded field in `cmd/server/main.go`.
- Never edit `internal/gen/` or `web/src/lib/gen/`.
- `gen.Error(code, msg)` only for conditions the client must distinguish; otherwise return `err`. No `log.Printf` in services.

## Safety
- A route is protected only if its group has `use: [auth]`. New routes go there unless deliberately public.
- Tokens only in `web/src/lib/auth.ts`. Never log secrets. Never create/drop/reset PostgreSQL.

## Verify
- Fixtures in `web/docs/fixtures`, keys `"METHOD /path"`. No browser E2E tests.
- Targeted tests while editing. UI changed → `make docs` once. Last: `make check` once.
