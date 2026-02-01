VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  = -s -w -X github.com/jeanpaul/aseity/pkg/version.Version=$(VERSION) -X github.com/jeanpaul/aseity/pkg/version.Commit=$(COMMIT)

.PHONY: build install clean docker docker-up docker-down deps fmt lint

build:
	go build -ldflags="$(LDFLAGS)" -o bin/aseity ./cmd/aseity

install:
	go install -ldflags="$(LDFLAGS)" ./cmd/aseity

deps:
	go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

# Docker
docker:
	docker build -t aseity:latest .

docker-up:
	docker compose up -d ollama
	@echo "Waiting for Ollama..."
	@sleep 5
	docker compose run --rm aseity

docker-up-vllm:
	docker compose --profile vllm up -d
	@echo "Waiting for services..."
	@sleep 10
	docker compose run --rm aseity --provider vllm

docker-down:
	docker compose --profile vllm down

# Model management shortcuts
pull:
	@read -p "Model name: " model; \
	go run ./cmd/aseity pull $$model

models:
	go run ./cmd/aseity models
