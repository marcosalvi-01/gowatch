include ./.env
export

.PHONY: run gen build setup vet

run: gen
	go run .

build: gen 
	go build -o ./tmp/main .

gen:
	go generate ./...

setup:
	go mod tidy

vet:
	go vet ./...

air:
	go tool air

clean:
	rm -f ./db.db
