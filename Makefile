# Run `make help` for the list. PostgreSQL is external: the server applies
# its embedded migrations at startup (up only), and no target here may
# create, reset, or destroy the database.
.NOTPARALLEL:

# Recipes see the same .env the server reads via godotenv
-include .env
export

.PHONY: help bootstrap gen check run build docs _check-tools _web

# Not "./...": web/node_modules can ship vendored .go files we don't own
GO_PKGS := ./cmd/... ./internal/... ./db ./web
GOFMT_PATHS := cmd internal db/embed.go web/embed.go

GO_REQUIRED := $(shell sed -n 's/^go \([0-9]*\.[0-9]*\).*/\1/p' go.mod)
BUN_PINNED := $(shell sed -n 's/.*"packageManager": "bun@\([0-9]*\.[0-9]*\).*/\1/p' web/package.json)
PLAYWRIGHT_INSTALL_ARGS ?=

help: ## List available commands
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  make %-10s %s\n", $$1, $$2}'

_check-tools:
	@command -v go >/dev/null 2>&1 || { echo "error: 'go' is required but was not found in PATH" >&2; exit 1; }
	@go_version="$$(go env GOVERSION)"; \
	case "$$go_version" in \
		go$(GO_REQUIRED)|go$(GO_REQUIRED).*) ;; \
		*) echo "error: this project requires Go $(GO_REQUIRED), but 'go env GOVERSION' reports $$go_version" >&2; exit 1;; \
	esac
	@command -v bun >/dev/null 2>&1 || { echo "error: 'bun' is required but was not found in PATH" >&2; exit 1; }
	@bun_version="$$(bun --version)"; \
	case "$$bun_version" in \
		$(BUN_PINNED).*) ;; \
		*) echo "error: this project requires bun $(BUN_PINNED).x, but 'bun --version' reports $$bun_version" >&2; exit 1;; \
	esac
	@command -v apic >/dev/null 2>&1 || { echo "error: 'apic' is required but was not found in PATH; install with: go install github.com/0to1a/apic/cmd/apic@v0.2.0" >&2; exit 1; }

bootstrap: _check-tools ## Install/download all dependencies (the only target that installs)
	@echo "==> go mod download"
	go mod download
	@echo "==> installing web dependencies"
	cd web && bun install --frozen-lockfile
	@echo "==> installing Playwright browser (used by make docs)"
	cd web && bunx playwright install chromium $(PLAYWRIGHT_INSTALL_ARGS)
	@echo "==> bootstrap done"

# sqlc defaults to the go.mod-pinned tool, compiled on demand by the Go
# toolchain. CI overrides it with a prebuilt binary of the same version.
SQLC ?= go tool sqlc

gen: ## Regenerate code from apic.yaml (Go + TS) and SQL (sqlc)
	@echo "==> generating code (apic, sqlc)"
	apic generate
	$(SQLC) generate

check: _check-tools gen _web ## Done-signal: codegen, lint, tests, both builds (installs nothing)
	@echo "==> apic diff"
	apic diff
	@echo "==> gofmt"
	@unformatted="$$(gofmt -l $(GOFMT_PATHS))"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt found unformatted files:" >&2; \
		echo "$$unformatted" >&2; \
		exit 1; \
	fi
	@echo "==> go vet"
	go vet $(GO_PKGS)
	@echo "==> go test"
	go test $(GO_PKGS)
	@echo "==> web lint"
	cd web && bun run lint
	@echo "==> web typecheck"
	cd web && bun run typecheck
	@echo "==> web unit tests (vitest)"
	cd web && bun run test:unit -- --run --passWithNoTests
	@echo "==> go build"
	go build $(GO_PKGS)
	@echo "==> make check passed"

run: _check-tools gen _web ## Build the frontend once, then run the single Go process
	@echo "==> running server on $${PORT:-8080}"
	go run ./cmd/server

build: _check-tools gen _web ## Produce the single production artifact: bin/server
	@echo "==> building bin/server"
	mkdir -p bin
	go build -o bin/server ./cmd/server

# vite build empties web/dist, including the tracked .gitkeep
_web:
	@echo "==> web production build"
	cd web && bun run build
	@touch web/dist/.gitkeep

docs: _check-tools gen ## Regenerate docs/routes.json, docs/templates.json, route-catalog.html, and screenshots (no backend/DB needed)
	@echo "==> generating route catalog (build + Playwright screenshots) and template manifest"
	cd web && bun run docs
	@touch web/dist/.gitkeep
	@echo "==> wrote docs/routes.json, docs/templates.json, docs/route-catalog.html, docs/screenshots/"
