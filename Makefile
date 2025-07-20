# XDR-Go Build System

.PHONY: build test clean install xdrgen generate-test generate-all

# Build the xdrgen binary
build: bin/xdrgen

# Backwards compatibility alias
xdrgen: bin/xdrgen

bin/xdrgen: bin tools/xdrgen/*.go
	cd tools/xdrgen && go build -o ../../bin/xdrgen .

# Install xdrgen to GOPATH/bin
install:
	cd tools/xdrgen && go install .


# Generate XDR code for test files
generate-test: benchmarks/benchmark_autogen_xdr_test.go benchmarks/benchmark_xdr_test.go
	@echo "Generating XDR code for test files..."
	@if [ -f codegen_test.go ]; then \
		echo "Generating for codegen_test.go..."; \
		$(PWD)/bin/xdrgen codegen_test.go; \
	fi
	@if [ -f xdr_alias_test.go ]; then \
		echo "Generating for xdr_alias_test.go..."; \
		$(PWD)/bin/xdrgen xdr_alias_test.go; \
	fi

# Generate XDR code for codegen_test files (with ignore build tags)
generate-codegen-test: bin/xdrgen
	@echo "Generating XDR code for codegen_test files..."
	@cd codegen_test && go generate -tags=ignore ./...
	@cd codegen_test/alias_chain_test && go generate -tags=ignore ./...

# Use Make's built-in dependency tracking with pattern rules
%_xdr.go: %.go bin/xdrgen
	@echo "Generating XDR for $<"
	@case "$<" in \
		synthetic_test/*) (cd synthetic_test && ../bin/xdrgen "$(patsubst synthetic_test/%,%,$<)") ;; \
		*) ./bin/xdrgen "$<" ;; \
	esac

%_xdr_test.go: %_test.go bin/xdrgen  
	@echo "Generating XDR for $<"
	@./bin/xdrgen "$<"

# Explicit rules for specific benchmark files
benchmarks/benchmark_autogen_xdr_test.go: benchmarks/benchmark_autogen_test.go bin/xdrgen
	@echo "Generating XDR for $<"
	@./bin/xdrgen "$<"

benchmarks/benchmark_xdr_test.go: benchmarks/benchmark_test.go bin/xdrgen
	@echo "Generating XDR for $<"
	@./bin/xdrgen "$<"

# Generate all XDR files using Make's dependency system
.PHONY: generate-all  
generate-all: benchmarks/benchmark_autogen_xdr_test.go benchmarks/benchmark_xdr_test.go
	@echo "Checking for other XDR files that need generation..."
	@# Let Make handle the dependencies for pattern-matched files
	@for src in $$(find . -name "*.go" -not -path "./tools/*" -not -name "*_xdr*.go" -not -name "main.go" -not -name "*_test.go"); do \
		target="$${src%.go}_xdr.go"; \
		$(MAKE) --no-print-directory "$$target" 2>/dev/null || true; \
	done
	@for src in $$(find ./codegen_test -name "*_test.go" -not -name "*_xdr*"); do \
		target="$${src%.go}_xdr_test.go"; \
		$(MAKE) --no-print-directory "$$target" 2>/dev/null || true; \
	done
	@echo "XDR generation complete"

# Run tests
test: generate-test generate-codegen-test
	@echo "Running tests in main workspace..."
	go test -v ./...
	@echo "Running tests in tools/xdrgen workspace..."
	cd tools/xdrgen && go test -v ./...
	@echo "Codegen test files generated successfully (no runtime tests needed)"

# Run tests with race detection
test-race: generate-test generate-codegen-test
	@echo "Running tests with race detection in main workspace..."
	go test -race -v ./...
	@echo "Running tests with race detection in tools/xdrgen workspace..."
	cd tools/xdrgen && go test -race -v ./...
	@echo "Codegen test files generated successfully (no runtime tests needed)"

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

# Lint code with golangci-lint and gosec (matches CI)
lint:
	@echo "Running golangci-lint on main workspace..."
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.2.2 run .
	@echo "Running golangci-lint on xdrgen workspace..."
	@cd tools/xdrgen && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.2.2 run $$(find . -name "*.go" -not -name "*_test.go" | tr '\n' ' ')
	@echo "Running gosec security scanner on main workspace..."
	@go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
	@echo "Running gosec security scanner on xdrgen workspace..."
	@cd tools/xdrgen && go run github.com/securego/gosec/v2/cmd/gosec@latest ./...

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
		echo "=== Running $$(basename $$dir) ==="; \
		(cd "$$dir" && go generate && go run .) || echo "Failed: $$dir"; \
		echo ""; \
	done

# Test all examples (run as demos)
examples-test: examples

# CI workflow
ci: check-format vet test-race lint examples-test