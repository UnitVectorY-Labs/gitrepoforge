
# Commands for gitrepoforge
default:
  @just --list
# Build gitrepoforge with Go
build:
  go build ./...

# Run tests for gitrepoforge with Go
test:
  go clean -testcache
  go test ./...
