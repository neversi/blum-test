ifneq ("$(wildcard .env.local)","")
	include .env.local
	export
endif

NAME := exchange-rate-calculator

.PHONY: build
build:
	go build -race -mod=vendor -o "$(NAME)" cmd/$(NAME)/main.go

.PHONY: run
run:
	go run cmd/$(NAME)/main.go

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
	swag init -g cmd/$(NAME)/main.go -d ./

.PHONY: swag-fmt
swag-fmt:
	swag fmt -d ./cmd/$(NAME)/,./

.PHONY: docker-up
docker-up:
	docker-compose up -d 

.PHONY: docker-down
docker-down:
	docker-compose down