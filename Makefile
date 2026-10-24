.PHONY: help install dev dev-api dev-web test test-api test-web build build-api build-web compose-up compose-down
.DEFAULT_GOAL := help

help:
	@echo 'KMS: make install | dev | test | build | compose-up | compose-down'
	@echo 'API: apps/api (Go); Web: apps/web (npm). See README.md for env setup.'

install:
	cd apps/api && go mod download
	npm --prefix apps/web ci

dev:
	$(MAKE) -j2 dev-api dev-web

dev-api:
	cd apps/api && go run .

dev-web:
	npm --prefix apps/web run dev

test: test-api test-web

test-api:
	cd apps/api && go test ./...

test-web:
	npm --prefix apps/web test

build: build-api build-web

build-api:
	cd apps/api && go build -o tmp/kms-api .

build-web:
	npm --prefix apps/web run build

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down
