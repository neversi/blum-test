ifneq ("$(wildcard .env.local)","")
	include .env.local
	export
endif

NAME := exchange-rate-calculator

.PHONY: build
build:
	go build -race -mod=vendor -o "$(NAME)"

.PHONY: run
run:
	go run cmd/main.go

.PHONY: local-test
local-test:
	go test -timeout 30s -tags=local ./internal/...

.PHONY: dep
dep:
	go mod vendor

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt-check
fmt-check:
	gofmt -l .

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: swag
swag:
	swag init