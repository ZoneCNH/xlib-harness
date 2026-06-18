# broken-trace SPEC

## 1. Summary

Broken trace fixture.

## 2. Goals

- Demonstrate traceability failures.

## 3. Functional Requirements

| ID | Requirement | WHEN | THEN |
| --- | --- | --- | --- |
| FR-001 | closed chain | WHEN checked | THEN AC and TC are present |
| FR-002 | missing chain | WHEN checked | THEN traceability reports the missing FR |

## 4. Acceptance Criteria

| AC ID | FR Ref | Criterion |
| --- | --- | --- |
| AC-001 | FR-001 | TC-001 proves the first chain |

## 5. Tests

| TC ID | Covers | Command |
| --- | --- | --- |
| TC-001 | FR-001 / AC-001 | xlib-harness check fixtures/broken-trace --profile full |

## 6. Boundaries

No forbidden dependencies.
