set dotenv-load

# Build the wasm client and server binary.
build: copy
    @GOOS=js GOARCH=wasm tinygo build -target wasm -opt=z -o bin/game.wasm cmd/client/main.go
    @wasm-opt -Oz --strip-debug --strip-producers -o bin/game-opt.wasm bin/game.wasm
    @cp bin/game.wasm cmd/server/assets/game.wasm

    @go build \
    -ldflags "-s -w" \
    -o ./bin/server ./cmd/server/main.go

# Copy the assets into the server directory.
copy:
    @cp assets/*.ogg cmd/server/assets/
    @cp assets/*.wav cmd/server/assets/
    @cp assets/*.png cmd/server/assets/

# Run the service.
run: build
    @./bin/server

# Setup tinygo environment and copy wasm_exec.js.
setup:
    @brew tap tinygo-org/tools
    @brew install tinygo
    @cp $TINYGOROOT/targets/wasm_exec.js cmd/service/assets/wasm_exec.js

# Test the Go sources.
test:
    @go test -v -coverprofile=.coverprofile.out ./internal/app/...
    @clear
    @echo "🚦 test coverage: $(go tool cover -func=.coverprofile.out | grep total | awk '{print $3}')"
