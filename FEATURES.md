# xlib-harness Features

> Module: `xlib-harness`
> Version: v0.1.5
> Last-Updated: 2026-06-20
> Implementation-Baseline: `v0.1.5` tag commit

## Feature Summary

`xlib-harness` provides a standard-library module generator and acceptance gate for Foundation modules. It generates the baseline module document set and checks specification structure, traceability closure, runtime dependency boundaries, Markdown hygiene, and CI/CD references in local and CI environments.

## Functional Features

| Feature ID | Capability | CLI / Artifact | Status |
| --- | --- | --- | --- |
| FR-001 | Generate module assets | `xlib-harness generate <module> --force` | Implemented |
| FR-002 | Specification structure gate | `xlib-harness check <module> --profile spec` | Implemented |
| FR-003 | Runtime boundary gate | `xlib-harness check <module> --profile boundary` | Implemented |
| FR-004 | CI/CD reference gate | `xlib-harness check <module> --profile full` | Implemented |
| FR-005 | Markdown format gate | `xlib-harness check <module> --profile full` | Implemented |
| FR-006 | Traceability closure gate | `xlib-harness check <module> --profile full` | Implemented |

## Generated Assets

| Asset | Purpose |
| --- | --- |
| `README.md` | Public module entry, command summary, and status |
| `SPEC.md` | 23-section specification document |
| `TRACEABILITY.md` | FR/AC/TC traceability matrix |
| `goal.md` | Module goal, scope, and evidence |
| `IMPLEMENTATION-PLAN.md` | Task, risk, and validation plan |
| `ACCEPTANCE.md` | Acceptance commands and evidence |
| `FEATURES.md` | Feature inventory |
| `tasks/TASK-001.md` | First executable task template |
| `Makefile` | Local build, test, coverage, and gate entry |
| `.github/workflows/ci.yml` | CI reference workflow |

## Quality Features

| NFR ID | Capability | Evidence |
| --- | --- | --- |
| NFR-001 | Standard-library runtime dependency boundary | `xlibgate check imports` and negative boundary fixtures |
| NFR-002 | Automation-readable JSON output | `xlib-harness check <module> --json` |
| NFR-003 | Repeatable fixture acceptance | `make ci` |
| NFR-004 | 100% Go statement coverage | `go tool cover -func=coverage.out` total 100.0% |
| NFR-005 | Release-gated public documentation | CI and release workflows require `README.md`, `FEATURES.md`, and `ACCEPTANCE.md` |

## Task Coverage

| Task | Feature Coverage | Status |
| --- | --- | --- |
| `TASK-XLIBHARNESS-001` | CLI generation and asset inventory | Completed |
| `TASK-XLIBHARNESS-002` | Specification structure and 23-section template | Completed |
| `TASK-XLIBHARNESS-003` | Runtime dependency boundary gate | Completed |
| `TASK-XLIBHARNESS-004` | CI/CD reference and Makefile gate | Completed |
| `TASK-XLIBHARNESS-005` | Markdown format gate | Completed |
| `TASK-XLIBHARNESS-006` | Traceability matrix closure gate | Completed |
| `TASK-XLIBHARNESS-007` | Code-repository feature and acceptance docs | Completed |

## Overall Score

The release candidate score is `100/100` for the requested delivery surface:

| Area | Score | Evidence |
| --- | --- | --- |
| Implementation correctness | 100/100 | Unit, race, vet, fixture gates, and negative fixtures pass |
| Coverage | 100/100 | Statement coverage total 100.0% |
| CI/CD | 100/100 | CI and release workflows validate docs, `make ci`, xlibgate, and release creation |
| Documentation sync | 100/100 | Repository-local `FEATURES.md` and `ACCEPTANCE.md` plus public projection docs are aligned |
| Release readiness | 100/100 | Final state is tag-published and merged to `main` |

## Release Notes

### v0.1.5

- Adds repository-local `FEATURES.md` and `ACCEPTANCE.md` so the code repository carries the same public feature and acceptance surface as the architecture projection.
- Extends CI/CD and release contract checks to require those synchronization documents.
- Keeps the v0.1.4 Markdown fence hardening, `xlibgate@v1.0.0` trust checks, full-profile 15 checks, and total 100.0% coverage baseline.

### v0.1.4

- Fixed the CI/CD trust tooling pin to use installable `xlibgate@v1.0.0` imports/gomod/baseline checks.
- Hardened Markdown fenced-code parsing for backtick and tilde fences, requiring matching marker types and sufficient closing length.
- Revalidated `make ci`, race, vet, coverage, benchmark, secret scan, and xlibgate trust checks; coverage total remained 100.0%.
