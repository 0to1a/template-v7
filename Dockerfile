# Mirrors `make build`: gen → web build → go build, one binary with the SPA
# and migrations embedded. PostgreSQL is external; config comes from env vars.

FROM golang:1.27@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS gen
WORKDIR /src
# Same apic version as CI
# No BuildKit cache mounts: Railway rejects them without a service-specific id
RUN go install github.com/0to1a/apic/cmd/apic@v0.2.0
COPY go.mod go.sum ./
RUN go mod download
COPY apic.yaml sqlc.yaml ./
COPY cmd ./cmd
COPY db ./db
COPY internal ./internal
COPY web/embed.go ./web/
RUN apic generate && go tool sqlc generate

FROM oven/bun:1.4.0@sha256:5ff609364c049b54eb0ff560ec96319729a972078ef2c755d758f0c6ef89c2d6 AS web
WORKDIR /src/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile --ignore-scripts
COPY web/ ./
COPY --from=gen /src/web/src/lib/gen ./src/lib/gen
RUN bun run build

FROM gen AS build
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# static-debian12:nonroot: static + CA certs (needed for SMTP TLS), runs as nonroot
FROM gcr.io/distroless/static-debian12@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
LABEL maintainer="0to1a" description="template-v7 server: Go API with embedded SPA" version="0.0.1"
COPY --from=build /server /server
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/server"]
