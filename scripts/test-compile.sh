#!/bin/bash

# Test compilation script for CasDash
set -e

echo "=== CasDash Compilation Test ==="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Change to project directory
cd "$(dirname "$0")/.."

print_status "Project directory: $(pwd)"

# Check Go version
print_status "Checking Go version..."
go version

# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found!"
    exit 1
fi

print_status "go.mod found"

# Download dependencies
print_status "Downloading dependencies..."
if ! go mod download; then
    print_error "Failed to download dependencies"
    exit 1
fi

print_status "Dependencies downloaded successfully"

# Tidy dependencies
print_status "Tidying dependencies..."
if ! go mod tidy; then
    print_error "Failed to tidy dependencies"
    exit 1
fi

print_status "Dependencies tidied"

# Check for syntax errors
print_status "Checking syntax..."
if ! go vet ./...; then
    print_error "Syntax errors found"
    exit 1
fi

print_status "Syntax check passed"

# Try to build
print_status "Attempting to build..."
if ! go build -o casdash-test main.go; then
    print_error "Build failed"
    exit 1
fi

print_status "Build successful!"

# Clean up test binary
if [ -f "casdash-test" ]; then
    rm casdash-test
    print_status "Test binary cleaned up"
fi

# Run tests if any exist
if [ -n "$(find . -name '*_test.go' 2>/dev/null)" ]; then
    print_status "Running tests..."
    if ! go test ./...; then
        print_warning "Some tests failed"
    else
        print_status "All tests passed"
    fi
else
    print_warning "No tests found"
fi

print_status "Compilation test completed successfully!"
print_status "Project structure:"
find . -name "*.go" | head -20

echo
print_status "=== Summary ==="
print_status "✓ Dependencies downloaded"
print_status "✓ Syntax check passed"
print_status "✓ Build successful"
print_status "✓ Project ready for development"