# XDR-Go Build System

.PHONY: build test clean install xdrgen generate-test generate-all

# Build the xdrgen binary
build: xdrgen

xdrgen: bin
	cd tools/xdrgen && go build -o ../../bin/xdrgen .

# Install xdrgen to GOPATH/bin
install:
	cd tools/xdrgen && go install .

# Generate XDR code for test files
generate-test:
	@echo "Generating XDR code for test files..."
	@if [ -f codegen_test.go ]; then \
		echo "Generating for codegen_test.go..."; \
		$(PWD)/bin/xdrgen codegen_test.go; \
	fi
	@if [ -f xdr_alias_test.go ]; then \
		echo "Generating for xdr_alias_test.go..."; \
		$(PWD)/bin/xdrgen xdr_alias_test.go; \
	fi
	@if [ -f benchmarks/benchmark_autogen_test.go ]; then \
		echo "Generating for benchmarks/benchmark_autogen_test.go..."; \
		$(PWD)/bin/xdrgen benchmarks/benchmark_autogen_test.go; \
	fi

# Generate XDR code for all files (regular + test files)
generate-all:
	@echo "Generating XDR code for all files..."
	@go generate ./...
	@$(MAKE) generate-test

# Run tests
test: generate-test
	go test -v ./...

# Run tests with race detection
test-race: generate-test
	go test -race -v ./...

# Run benchmarks (with build tags)
bench: generate-test
	go test -tags=bench -bench=. -benchmem ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Create bin directory
bin:
	mkdir -p bin

# Format code with gci (comprehensive formatting)
format:
	@go run github.com/daixiang0/gci@latest write --skip-generated -s standard -s default .
	@go run github.com/daixiang0/gci@latest write --skip-generated -s standard -s default ./tools/xdrgen

# Check code formatting with gci
check-format:
	@go run github.com/daixiang0/gci@latest diff --skip-generated -s standard -s default .
	@go run github.com/daixiang0/gci@latest diff --skip-generated -s standard -s default ./tools/xdrgen

# Vet code
vet:
	go vet ./...

# Lint code with golangci-lint v2
lint:
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --timeout=5m

# Run all checks
check: check-format vet test-race

# Build all platforms for release
build-all: bin
	cd tools/xdrgen && GOOS=linux GOARCH=amd64 go build -o ../../bin/xdrgen-linux-amd64 .
	cd tools/xdrgen && GOOS=linux GOARCH=arm64 go build -o ../../bin/xdrgen-linux-arm64 .
	cd tools/xdrgen && GOOS=darwin GOARCH=amd64 go build -o ../../bin/xdrgen-darwin-amd64 .
	cd tools/xdrgen && GOOS=darwin GOARCH=arm64 go build -o ../../bin/xdrgen-darwin-arm64 .
	cd tools/xdrgen && GOOS=windows GOARCH=amd64 go build -o ../../bin/xdrgen-windows-amd64.exe .

# Development workflow
dev: format vet test

# Build and run all examples
examples: xdrgen
	@echo "Building and running all examples..."
	@for dir in examples/*/; do \
		echo "=== Testing $$(basename $$dir) ==="; \
		cd "$$dir" && go generate && go run . && cd ../..; \
		echo ""; \
	done

# CI workflow
ci: check-format vet test-race lint