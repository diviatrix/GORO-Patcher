.PHONY: build build-linux build-windows build-all test lint clean bindings frontend

# Build for current platform (Linux)
build: frontend bindings
	./scripts/build.sh linux

# Build for both platforms
build-all: frontend bindings
	./scripts/build.sh all

# Build for Linux only
build-linux: frontend bindings
	./scripts/build.sh linux

# Build for Windows only
build-windows: frontend bindings
	./scripts/build.sh windows

# Regenerate TypeScript bindings
bindings:
	cd src && ~/go/bin/wails3 generate bindings -b
	cp -r src/frontend/bindings src/frontend/dist/

# Copy frontend files to dist
frontend:
	cp src/frontend/src/main.js src/frontend/dist/
	cp -r src/frontend/bindings src/frontend/dist/

# Run tests
test:
	cd src && go test ./...

# Run linter
lint:
	cd src && go vet ./...

# Clean build artifacts
clean:
	rm -rf src/build/bin src/frontend/wailsjs src/build
	rm -f build/GORO-Patcher build/GORO-Patcher.exe
	rm -f build/hashfile build/hashfile.exe
