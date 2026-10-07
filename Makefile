.PHONY: deps check fmt test coverage build build-bin clean

deps:
	go mod tidy
	go mod download
	go mod verify

check:
	golangci-lint fmt ./...
	go fix ./...
	go vet ./...
	golangci-lint run ./...

test:
	go test ./...

coverage:
	mkdir -p build
	go test -coverprofile=build/coverage.out ./...
	go tool cover -html=build/coverage.out -o build/coverage.html
	@echo "Coverage report generated: build/coverage.html"

build:
	go build ./...

build-bin:
	mkdir -p build
	GOOS=darwin GOARCH=arm64 go build -o ./build/desync-darwin-arm64 ./cmd/desync
	GOOS=linux GOARCH=amd64 go build -o ./build/desync-linux-amd64 ./cmd/desync
	GOOS=linux GOARCH=arm64 go build -o ./build/desync-linux-arm64 ./cmd/desync

clean:
	rm -rf ./build
