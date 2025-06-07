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
