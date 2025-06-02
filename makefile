include ./.env
export

.PHONY: run gen build setup vet

run: gen
	go run .

gen:
	go generate ./...

setup:
	go mod tidy

vet:
	go vet ./...

air:
	go tool air
