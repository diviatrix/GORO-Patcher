.PHONY: dev build test lint clean

dev:
	go run .

build:
	wails3 build

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf build/bin frontend/wailsjs
