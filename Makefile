.PHONY: build test e2e-test clean docker-build fmt vet generate

BINARY_NAME=binarylane-cloud-controller-manager
DOCKER_IMAGE=binarylane-cloud-controller-manager
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

all: test build

generate:
	@echo "Generating API client from OpenAPI spec..."
	go generate ./...

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

test:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./...

e2e-test:
	@echo "Running end-to-end tests..."
	@if [ -z "$$BINARYLANE_API_TOKEN" ]; then \
		echo "Error: BINARYLANE_API_TOKEN environment variable is not set"; \
		exit 1; \
	fi
	./scripts/e2e-test.sh

coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Vetting code..."
	go vet ./...

lint: fmt vet
	@echo "Linting complete"

docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

.DEFAULT_GOAL := build
