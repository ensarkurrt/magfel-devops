BINARY_NAME=swarmforge
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-X github.com/ensarkurrt/swarmforge/cmd.Version=$(VERSION) -X github.com/ensarkurrt/swarmforge/cmd.BuildTime=$(BUILD_TIME)"

.PHONY: build install clean test vet fmt release

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

install:
	go install $(LDFLAGS) .

clean:
	rm -f $(BINARY_NAME)
	go clean

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

release: clean
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
