include ./.env
export

.PHONY: run gen build setup vet

run: gen
	go run .

build: gen 
	go build -o ./tmp/main .

gen: fmt
	go generate ./...

fmt:
	go tool swag fmt

setup:
	go mod tidy
	npm install -D tailwindcss

vet:
	go vet ./...
	sqlc vet

serve:
	go tool air

clean:
	rm -f ./db.db
	rm -fr ./docs
