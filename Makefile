# XDR-Go Build System

.PHONY: build test clean install xdrgen

# Build the xdrgen binary
build: xdrgen

xdrgen: bin
	cd tools/xdrgen && go build -o ../../bin/xdrgen .

# Install xdrgen to GOPATH/bin
install:
	cd tools/xdrgen && go install .

# Run tests
test:
	go test -v ./...

# Run tests with race detection
test-race:
	go test -race -v ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Create bin directory
bin:
	mkdir -p bin

# Format code
fmt:
	go fmt ./...

# Format imports with gci
format-imports:
	go run github.com/daixiang0/gci@latest write --skip-generated -s standard -s default -s "prefix(github.com/tempusfrangit/go-xdr)" .

# Check import formatting
check-imports:
	go run github.com/daixiang0/gci@latest diff --skip-generated -s standard -s default -s "prefix(github.com/tempusfrangit/go-xdr)" .

# Vet code
vet:
	go vet ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Run all checks
check: fmt format-imports vet test-race

# Build all platforms for release
build-all: bin
	cd tools/xdrgen && GOOS=linux GOARCH=amd64 go build -o ../../bin/xdrgen-linux-amd64 .
	cd tools/xdrgen && GOOS=linux GOARCH=arm64 go build -o ../../bin/xdrgen-linux-arm64 .
	cd tools/xdrgen && GOOS=darwin GOARCH=amd64 go build -o ../../bin/xdrgen-darwin-amd64 .
	cd tools/xdrgen && GOOS=darwin GOARCH=arm64 go build -o ../../bin/xdrgen-darwin-arm64 .
	cd tools/xdrgen && GOOS=windows GOARCH=amd64 go build -o ../../bin/xdrgen-windows-amd64.exe .

# Development workflow
dev: fmt vet test

# CI workflow
ci: fmt format-imports vet test-race lint