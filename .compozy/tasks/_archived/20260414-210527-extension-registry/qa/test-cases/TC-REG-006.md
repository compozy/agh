# TC-REG-006: Extension Capability Ceiling Enforced for Marketplace Installs

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Regression |
| **Estimated Time** | 3 min |
| **Module** | `internal/extension/capability.go` |
| **Changed In** | Task 04 — Marketplace Integration |

## Objective

Validate that the capability restriction for marketplace-sourced extensions was not accidentally weakened during the registry integration.

## Preconditions

- Extension installed from marketplace.
- Extension capability evaluation code.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Install extension from marketplace | **Expected:** Source is `SourceMarketplace`. |
| 2 | List allowed capabilities for the extension | **Expected:** Exactly `memory.read`, `observe.read`, `session.read`, `skills.read`, `tool.read`. |
| 3 | Verify no write capabilities are allowed | **Expected:** No `*.write` capabilities in allowed set. |
| 4 | Compare with local extension capabilities | **Expected:** Local has broader set. |

## Regression Risk

High — security boundary. Any loosening of marketplace capabilities is a vulnerability.
