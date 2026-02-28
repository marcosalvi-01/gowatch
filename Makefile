.PHONY: build clean gen setup test lint vet fmt fmt-check dev tailwind templ air tailwind-install
include ./.env
export

# Development mode - runs tailwind, templ, and air in parallel
dev:
	make -j3 tailwind templ air

tailwind: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i ./internal/handlers/static/static/css/input.css -o ./internal/handlers/static/static/css/output.css --minify --watch

templ:
	templ generate --watch --proxy="http://localhost:8090" --open-browser=false

air:
	air 

# Configuration
BUILD_DIR := ./tmp
DB_DIR := db
TAILWIND_VERSION ?= v4.1.12
TAILWIND_BIN_DIR := ./tmp/bin
TAILWIND_BIN := $(TAILWIND_BIN_DIR)/tailwindcss

# File definitions
SQL_FILES := $(DB_DIR)/queries.sql $(DB_DIR)/migrations/*.sql
SQLC_TIMESTAMP := $(DB_DIR)/sqlc/.sqlc-generated

# Version for ldflags
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)

# Build target
build: gen
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X gowatch/cmd.Version=$(VERSION)" -o $(BUILD_DIR)/gowatch .

# Generate all code (sqlc and templ)
gen: 
	$(MAKE) $(SQLC_TIMESTAMP)
	templ generate

# Generate sqlc files when SQL files or config change
$(SQLC_TIMESTAMP): $(SQL_FILES) sqlc.yaml
	@echo 'regenerating sqlc files'
	sqlc generate
	@touch $@

clean:
	rm -rf $(DB_DIR)/sqlc
	rm -f $(BUILD_DIR)/*

setup:
	@echo "Checking prerequisites..."
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed. Please install Go 1.24.3+"; exit 1; }
	@command -v curl >/dev/null 2>&1 || { echo "curl is required but not installed."; exit 1; }
	@echo "Installing Go dependencies..."
	go mod download
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/templui/templui/cmd/templui@latest
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@echo "Installing Tailwind CSS standalone binary..."
	$(MAKE) tailwind-install
	@echo "Setup complete!"

tailwind-install: $(TAILWIND_BIN)

$(TAILWIND_BIN):
	@mkdir -p $(TAILWIND_BIN_DIR)
	@os=$$(uname -s); arch=$$(uname -m); \
	case "$$os/$$arch" in \
		Linux/x86_64) asset="tailwindcss-linux-x64" ;; \
		Linux/aarch64|Linux/arm64) asset="tailwindcss-linux-arm64" ;; \
		Darwin/x86_64) asset="tailwindcss-macos-x64" ;; \
		Darwin/arm64) asset="tailwindcss-macos-arm64" ;; \
		*) echo "Unsupported platform: $$os/$$arch"; exit 1 ;; \
	esac; \
	curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/$$asset" -o "$(TAILWIND_BIN)"
	@chmod +x "$(TAILWIND_BIN)"

# Format code using golangci-lint formatters
fmt:
	golangci-lint fmt -E gofumpt -E goimports ./...

# Check formatting without modifying files
fmt-check:
	golangci-lint fmt --diff -E gofumpt -E goimports ./...

# Run go vet and sqlc vet
vet:
	go vet ./...
	sqlc vet

# Run golangci-lint linters
lint:
	golangci-lint run ./...

# Run all checks before committing
check: fmt-check vet lint test
	@echo "All checks passed!"

# Run tests
test:
	go test ./...
