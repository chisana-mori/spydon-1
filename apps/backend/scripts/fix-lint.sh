#!/bin/bash

# Fix linting issues script
echo "Running automatic linting fixes..."

# Format Go files using gofumpt
echo "Running gofumpt..."
find . -name "*.go" -type f ! -path "./vendor/*" | xargs gofumpt -w

# Format imports using goimports
echo "Running goimports..."
find . -name "*.go" -type f ! -path "./vendor/*" | xargs goimports -w

# Run go fmt as fallback
echo "Running go fmt..."
go fmt ./...

echo "Automatic fixes completed."
echo "Run 'golangci-lint run' again to see remaining issues."
