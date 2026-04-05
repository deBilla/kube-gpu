BINARY_NAME=kube-gpu
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-s -w -X github.com/deBilla/kube-gpu/cmd.version=$(VERSION) -X github.com/deBilla/kube-gpu/cmd.commit=$(COMMIT) -X github.com/deBilla/kube-gpu/cmd.date=$(DATE)

.PHONY: build test lint install clean release-snapshot

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) .

install:
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" .

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ dist/

release-snapshot:
	goreleaser build --snapshot --clean
