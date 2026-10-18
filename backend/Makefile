-include .env
export

TEST_DATABASE_URL ?= postgres:///samacode_test?host=/tmp

.PHONY: run migrate test test-unit fmt vet build

run:            ## Lance l'API (migrations auto en dev)
	go run ./cmd/api

migrate:        ## Applique les migrations sur DATABASE_URL
	go run ./cmd/migrate

test:           ## Tous les tests (base de test requise, paquets en série)
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -p 1 -count=1 ./...

test-unit:      ## Tests sans base de données
	TEST_DATABASE_URL= go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

build:
	go build -o bin/api ./cmd/api && go build -o bin/migrate ./cmd/migrate
