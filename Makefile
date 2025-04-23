.PHONY: build test clean run

# Build the application
build:
	go build -o mermgen

# Run tests
test:
	go test -v ./...

# Run integration tests
test-integration:
	go test -v ./... -tags=integration

# Clean build artifacts
clean:
	rm -f mermgen
	rm -rf diagrams/

# Run the application
run:
	@if [ -z "$(REPO)" ]; then \
		echo "Usage: make run REPO=github.com/user/repo [OUTPUT=path/to/output]"; \
		exit 1; \
	fi
	@if [ -z "$(OUTPUT)" ]; then \
		./mermgen -repo $(REPO) -output diagrams; \
	else \
		./mermgen -repo $(REPO) -output $(OUTPUT); \
	fi

# Install dependencies
deps:
	go mod download

# Format the code
fmt:
	go fmt ./...

# Run linter
lint:
	go vet ./...

# Build and install the binary
install:
	go install 