# Mirrors `make build`: gen → web build → go build, one binary with the SPA
# and migrations embedded. PostgreSQL is external; config comes from env vars.

FROM golang:1.27 AS gen
WORKDIR /src
# Same apic version as CI
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go install github.com/0to1a/apic/cmd/apic@v0.2.0
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    apic generate && go tool sqlc generate

FROM oven/bun:1.4.0 AS web
WORKDIR /src/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
COPY --from=gen /src/web/src/lib/gen ./src/lib/gen
RUN bun run build

FROM gen AS build
COPY --from=web /src/web/dist ./web/dist
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# static + CA certs (needed for SMTP TLS), runs as nonroot
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /server /server
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/server"]
