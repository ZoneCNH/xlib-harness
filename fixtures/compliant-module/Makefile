.PHONY: build test vet ci

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

ci: build test vet
