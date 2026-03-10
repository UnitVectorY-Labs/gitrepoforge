# List all available commands
default:
  @just --list

# Build the Go application
build:
  go build ./...

# Run the Go tests
test:
  go clean -testcache
  go test ./...
