APP_NAME=go-admin-base
BUILD_DIR=build
MAIN_FILE=cmd/server/main.go
WORKER_FILE=cmd/worker/main.go
SCHEDULER_FILE=cmd/scheduler/main.go

.PHONY: all build clean run test worker scheduler

all: build

build:
	@echo "Building server..."
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Building worker..."
	@go build -o $(BUILD_DIR)/$(APP_NAME)-worker $(WORKER_FILE)
	@echo "Building scheduler..."
	@go build -o $(BUILD_DIR)/$(APP_NAME)-scheduler $(SCHEDULER_FILE)

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

run:
	@echo "Running server..."
	@go run $(MAIN_FILE)

worker:
	@echo "Running worker..."
	@go run $(WORKER_FILE)

test:
	@echo "Running tests..."
	@go test -v ./...

deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

lint:
	@echo "Running linter..."
	@golangci-lint run

dev:
	@echo "Running server in development mode..."
	@air -c .air.toml

.PHONY: migrate
migrate:
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go

.PHONY: seed
seed:
	@echo "Running database seeds..."
	@go run cmd/seed/main.go

.PHONY: docker-build docker-run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=go-admin-base
MAIN_PATH=cmd/server/main.go

all: lint test build

build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

run:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)
	./$(BINARY_NAME)

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

lint:
	golangci-lint run

tidy:
	$(GOMOD) tidy

docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run -p 8080:8080 $(BINARY_NAME)

migrate-up:
	go run cmd/migration/main.go up

migrate-down:
	go run cmd/migration/main.go down

dev:
	air -c .air.toml

help:
	@echo "make - Build and run tests"
	@echo "make build - Build the binary"
	@echo "make run - Run the application"
	@echo "make test - Run tests"
	@echo "make clean - Clean build files"
	@echo "make lint - Run linter"
	@echo "make tidy - Tidy go.mod"
	@echo "make docker-build - Build Docker image"
	@echo "make docker-run - Run Docker container"
	@echo "make migrate-up - Run database migrations"
	@echo "make migrate-down - Rollback database migrations"
	@echo "make dev - Run with hot reload"

.PHONY: all build clean test lint migrate seed build-linux build-windows build-darwin

# Build variables
BINARY_NAME=go-admin
VERSION=1.0.0
BUILD_DIR=build
MAIN_FILE=cmd/server/main.go

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

all: lint test build

build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_FILE)

# Production builds for different platforms
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)

# Build for all platforms
build-all: build-linux build-windows build-darwin

# Create release package
release: build-all
	mkdir -p $(BUILD_DIR)/release
	cp configs/config.prod.yaml $(BUILD_DIR)/release/config.yaml
	cp -r docs $(BUILD_DIR)/release/
	cd $(BUILD_DIR) && tar -czf release.tar.gz release/

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

test:
	$(GOTEST) -v ./...

lint:
	$(GOLINT) run

migrate:
	$(GOCMD) run cmd/migrate/main.go

seed:
	$(GOCMD) run cmd/migrate/main.go -seed

# Dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

# Run the application
run:
	$(GOCMD) run $(MAIN_FILE)

# Run with live reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air 