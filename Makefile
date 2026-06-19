.PHONY: build test race vet coverage boundary ci clean

COVERAGE_PROFILE ?= coverage.out

build:
	go build ./...

test:
	go test ./...

race:
	go test ./... -race -count=1

vet:
	go vet ./...

coverage:
	go test ./... -coverprofile=$(COVERAGE_PROFILE) -covermode=count
	go tool cover -func=$(COVERAGE_PROFILE) | awk '/^total:/ { if ($$3 != "100.0%") { printf("coverage %s, want 100.0%%\n", $$3); exit 1 } found=1 } END { if (!found) exit 1 }'

boundary:
	go run . check fixtures/compliant-module --profile full
	if go run . check fixtures/module-with-bad-dep --profile boundary; then echo "expected forbidden dependency fixture to fail" >&2; exit 1; fi
	if go run . check fixtures/broken-trace --profile full; then echo "expected broken trace fixture to fail" >&2; exit 1; fi

ci: build test race vet coverage boundary

clean:
	rm -f coverage.out coverage.html
