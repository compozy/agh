# AGH Release Network Regression Suite

**Date:** 2026-04-24
**Suite type:** Smoke + targeted + full release regression.
**Execution status:** Not fully run yet.

## Smoke Suite

| ID            | Priority | Scenario                   | Command / method             | Expected                                              |
| ------------- | -------: | -------------------------- | ---------------------------- | ----------------------------------------------------- |
| SMOKE-NET-001 |       P0 | Network package unit suite | `go test ./internal/network` | Passes without race or goroutine retry churn symptoms |
| SMOKE-REL-001 |       P0 | Full repository gate       | `make verify`                | fmt, lint, tests, builds all pass                     |
| SMOKE-WEB-001 |       P0 | Web network surface builds | included in `make verify`    | lint/typecheck/test/build pass                        |

## Targeted Network Regression

| ID         | Priority | Scenario                                                                                   | Evidence                                                      |
| ---------- | -------: | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------- |
| TC-NET-001 |       P0 | Direct and broadcast network messages route only to valid peers and preserve metadata      | router/manager integration tests; CLI/API e2e                 |
| TC-NET-002 |       P0 | PromptNetwork failure requeues message and retries with capped backoff, not immediate loop | `TestDeliveryCoordinatorRetriesPromptFailuresAfterWorkerExit` |
| TC-NET-003 |       P0 | Busy sessions receive one queued network message per turn end                              | delivery integration test                                     |
| TC-NET-004 |       P0 | Audit records accepted, rejected, sent and delivered network messages                      | globaldb audit/timeline tests; API timeline                   |
| TC-NET-005 |       P0 | Network-origin task ingress preserves origin/channel and resumes owner                     | daemon task runtime/integration tests                         |
| TC-NET-006 |       P1 | Web network channels/peers/timeline survive reload                                         | Playwright/browser evidence                                   |
| TC-NET-007 |       P1 | Invalid payloads/expired/duplicate messages are rejected without local delivery            | router tests and audit entries                                |

## Full Regression Commands

1. `make deps`
2. `make verify`
3. `make test-integration`
4. `make test-e2e-runtime`
5. `make test-e2e-web`
6. Real LLM smoke if environment supports it.

## Known Execution Notes

- `make verify` is the required release gate.
- Integration/e2e lanes may take longer and may require local binaries/browsers. Any blocked lane must include exact blocker output.
- Real LLM tests must not print API keys or credential values.
