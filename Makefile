# Binary name
BINARY_NAME=bin/db_terminal

# Build settings
GOOS=linux
GOARCH=arm64
CGO_ENABLED=0

.PHONY: all build clean run test

# Default target
all: build

# Build the application
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY_NAME)

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean

# Run the application
run: build
	./$(BINARY_NAME)

# Run tests
test:
	go test ./...

# Build for development (local architecture)
build-dev:
	go build -o $(BINARY_NAME)