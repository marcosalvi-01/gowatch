include ./.env
export

.PHONY: serve run build build-docker gen fmt setup vet clean

serve:
	go tool air

run: gen
	go run .

build: gen 
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./tmp/app .

build-docker: build
	docker build -t gowatch .

gen: fmt
	go generate ./...

fmt:
	go tool swag fmt

setup:
	@echo "Checking prerequisites..."
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed. Please install Go 1.24.3+"; exit 1; }
	@command -v npm >/dev/null 2>&1 || { echo "npm is required but not installed. Please install Node.js/npm"; exit 1; }
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing npm dependencies..."
	npm install tailwindcss

vet:
	go vet ./...
	go tool sqlc vet

clean:
	rm -f ./db.db
	rm -fr ./docs
	rm -fr ./tmp
