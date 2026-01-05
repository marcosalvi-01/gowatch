.PHONY: build clean gen setup test lint vet fmt dev tailwind templ air
include ./.env
export

# Development mode - runs tailwind, templ, and air in parallel
dev:
	make -j3 tailwind templ air

tailwind:
	npx tailwindcss -i ./internal/handlers/static/static/css/input.css -o ./internal/handlers/static/static/css/output.css --minify --watch

templ:
	templ generate --watch --proxy="http://localhost:8090" --open-browser=false

air:
	air 

# Configuration
BUILD_DIR := ./tmp
DB_DIR := db

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
	@command -v npm >/dev/null 2>&1 || { echo "npm is required but not installed. Please install Node.js/npm"; exit 1; }
	@echo "Installing Go dependencies..."
	go mod download
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/templui/templui/cmd/templui@latest
	go install github.com/air-verse/air@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
	@echo "Installing npm dependencies..."
	npm install tailwindcss @tailwindcss/cli
	@echo "Setup complete!"

# Format code with gofumpt
fmt:
	gofumpt -l -w .

# Run go vet and sqlc vet
vet:
	go vet ./...
	sqlc vet

# Run golangci-lint (includes vet, formatting checks, and more)
lint: fmt
	golangci-lint run ./...

# Run all checks before committing
check: fmt vet lint test
	@echo "All checks passed!"

# Run tests
test:
	go test ./...
