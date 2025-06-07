include ./.env
export

.PHONY: run gen build setup vet

run: gen
	go run .

build: clean gen 
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./tmp/app .

build-docker: build
	docker build -t gowatch .

gen: fmt
	go generate ./...

fmt:
	go tool swag fmt

setup:
	go install
	npm install tailwindcss

vet:
	go vet ./...
	go tool sqlc vet

serve:
	go tool air

clean:
	rm -f ./db.db
	rm -fr ./docs
	rm -fr ./tmp
