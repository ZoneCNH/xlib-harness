# broken-trace SPEC

## 1. Summary

Broken trace fixture.

## 2. Goals

- Demonstrate traceability failures while keeping spec structure valid.

## 3. Non-Goals

- Do not fail the spec gate.

## 4. Stakeholders

- Fixture maintainers.

## 5. Glossary

- FR: Functional requirement.
- AC: Acceptance criterion.
- TC: Test case.

## 6. Functional Requirements

| ID | Requirement | WHEN | THEN |
| --- | --- | --- | --- |
| FR-001 | closed chain | WHEN checked | THEN AC and TC are present |
| FR-002 | missing chain | WHEN checked | THEN traceability reports the missing FR |

## 7. Business Rules

| ID | Rule |
| --- | --- |
| BR-001 | Broken trace remains intentional. |

## 8. Acceptance Criteria

| AC ID | FR Ref | Criterion |
| --- | --- | --- |
| AC-001 | FR-001 | TC-001 proves the first chain. |
| AC-002 | FR-002 | TC-002 proves the missing chain is detected. |

## 9. Tests

| TC ID | Covers | Command |
| --- | --- | --- |
| TC-001 | FR-001 / AC-001 | xlib-harness check fixtures/broken-trace --profile full |
| TC-002 | FR-002 / AC-002 | xlib-harness check fixtures/broken-trace --profile full |

## 10. Traceability

TRACEABILITY.md intentionally omits FR-002 / AC-002 / TC-002.

## 11. Interfaces

No code interface is required.

## 12. Data Model

No persisted data is required.

## 13. Error Handling

Gate failures report itemized traceability gaps.

## 14. Security

No credentials are stored.

## 15. Privacy

No private data is stored.

## 16. Performance

Fixture checks complete locally.

## 17. Observability

Check output includes pass/fail details.

## 18. Operations

Use the fixture only for local test coverage.

## 19. CI Gates

Run go test ./... and xlib-harness check fixtures/broken-trace --profile full.

## 20. Migration

No migration is required.

## 21. Risks

- Fixture drift from production rules.

## 22. Open Questions

- None.

## 23. Changelog

- Added canonical section coverage with an intentional traceability gap.
