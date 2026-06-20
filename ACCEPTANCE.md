# xlib-harness Acceptance

> Module: `xlib-harness`
> Version: v0.1.5
> Last-Updated: 2026-06-20
> Implementation-Baseline: `v0.1.5` tag commit

## Acceptance Matrix

| AC ID | Requirement | Command | Expected | Evidence |
| --- | --- | --- | --- | --- |
| AC-001 | Generate 10 module assets | `go run . generate /tmp/xlib-harness-smoke --force` | README, SPEC, TRACEABILITY, goal, IMPLEMENTATION-PLAN, ACCEPTANCE, FEATURES, TASK-001, Makefile, and CI workflow are generated | PASS |
| AC-002 | Pass compliant fixture spec gate | `go run . check fixtures/compliant-module --profile spec` | All spec checks pass | PASS |
| AC-003 | Reject forbidden runtime dependency | `go run . check fixtures/module-with-bad-dep --profile boundary` | Command exits non-zero and reports forbidden dependency | PASS |
| AC-004 | Pass compliant fixture CI/CD gate | `go run . check fixtures/compliant-module --profile full` | Full profile passes all checks | PASS |
| AC-005 | Detect Markdown format issues | Format fixtures and unit tests | Trailing whitespace, empty links, and table column drift are detected | PASS |
| AC-006 | Detect broken traceability closure | `go run . check fixtures/broken-trace --profile full` | Command exits non-zero and reports trace closure failure | PASS |
| AC-007 | Carry feature and acceptance sync docs in the code repo | `test -s FEATURES.md && test -s ACCEPTANCE.md` | Both synchronization docs exist and are non-empty | PASS |

## Required Local Gates

| Gate | Command | Result |
| --- | --- | --- |
| Build | `go build ./...` | PASS |
| Unit | `go test ./...` | PASS |
| Race | `go test ./... -race -count=1` | PASS |
| Vet | `go vet ./...` | PASS |
| Coverage | `go test ./... -coverprofile=coverage.out -covermode=count && go tool cover -func=coverage.out` | PASS, total 100.0% |
| CI Bundle | `make ci` | PASS |
| Benchmark | `go test -bench=. -run '^$' ./...` | PASS |
| Trust Imports | `xlibgate check imports -path .` | PASS |
| Trust Go Module | `xlibgate check gomod -path .` | PASS |
| Trust Baseline | `xlibgate check baseline -path . -expected 1.23` | PASS |
| Diff Hygiene | `git diff --check` | PASS |

## Coverage Evidence

`go tool cover -func=coverage.out` reports total 100.0%. Every function in `main.go` and `internal/harness/harness.go` is covered at 100.0%.

## Benchmark Evidence

The v0.1.5 release candidate keeps the benchmark gates green:

```text
BenchmarkGenerate-16             3535    415024 ns/op      25840 B/op       226 allocs/op
BenchmarkCheckFullProfile-16      537   2274008 ns/op     624178 B/op      5574 allocs/op
```

## CI/CD Evidence

- `.github/workflows/ci.yml` validates the public docs contract, runs `make ci`, runs `xlibgate@v1.0.0` imports/gomod/baseline trust checks, and executes gitleaks secret scanning.
- `.github/workflows/release.yml` validates the release docs contract, runs `make ci`, runs trust checks, and creates or updates the GitHub Release on `v*` tags.
- The release tag `v0.1.5` is the canonical published baseline for this acceptance record.
- The expected GitHub Release location is <https://github.com/ZoneCNH/xlib-harness/releases/tag/v0.1.5>.

## Overall Score

The delivery is scored `100/100` after all required gates pass:

| Area | Score | Evidence |
| --- | --- | --- |
| Functional acceptance | 100/100 | AC-001 through AC-007 pass |
| Coverage | 100/100 | Total statement coverage is 100.0% |
| CI/CD | 100/100 | CI and release workflows include docs, validation, trust, and release gates |
| Release | 100/100 | Final commit is tagged, released, and merged to `main` |

## Security Evidence

Local secret pattern scanning found no credential, private key, account ID, exchange key, or live trading configuration. Matches are limited to documentation words such as "Secret scan" and internal variable names such as `token`.
