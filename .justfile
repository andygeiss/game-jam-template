set dotenv-load

# Build the CLI binary.
build-cli:
    @go build \
    -ldflags "-s -w" \
    -o ./bin/cli ./cmd/cli/main.go

# Build the service binary.
build-service:
    @go build \
    -ldflags "-s -w" \
    -o ./bin/service ./cmd/service/main.go

# Generate an encryption key.
genkey:
    @go run cmd/genkey/main.go

# Runt the CLI
run-cli: build-cli
    @./bin/cli

# Run the service.
run-service: build-service
    @./bin/service

# Test the Go sources (Units).
test:
    @go test -v -coverprofile=.coverprofile.out ./internal/app/...
    @clear
    @echo "🚦 test coverage: $(go tool cover -func=.coverprofile.out | grep total | awk '{print $3}')"
