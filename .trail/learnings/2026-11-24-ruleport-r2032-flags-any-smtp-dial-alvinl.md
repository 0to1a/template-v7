---
title: "ruleport R2032 flags any smtp.Dial; Dockerfile pins digests only, tag in comment"
author: Alvinl
date: 2026-11-24
branch: chore/ruleport-clean
cost: { minutes: 3, tokens: 25339 }
id: l-2b390279
scope: repo
status: active
src: agent
touches:
  - internal/mail/mail.go
  - apic.yaml
  - sqlc.yaml
  - web/embed.go
---

## Decided
- internal/mail Send dials manually and requires STARTTLS for non-loopback hosts (no silent plaintext downgrade). Loopback stays plaintext for local relays (localhost:1025). Port 465 implicit TLS is not supported.
- Dockerfile base images pinned by multi-arch index digest. For distroless the tag lives in a comment: tag+digest together triggers ruleport docker:R4624.
- Dockerfile gen stage copies explicit paths (apic.yaml, sqlc.yaml, cmd, db, internal, web/embed.go) instead of `COPY . .` (docker:R2136). A new top-level Go dir must be added there.

## Watch out
- ruleport go:R2032 still flags mail.go's smtp.Dial even though TLS is enforced; it's a static false positive, don't chase it.
- Remaining known false positives: otp.go totpSecretMessagePrefix (R2076), repository.go `if err := ...; err != nil` (R4572), distroless :nonroot already non-root (R2137).
- Digests must be bumped manually (no Renovate yet).
