---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/tool_integration_test.go
line: 53
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4204955814,nitpick_hash:4b9b69220601
review_hash: 4b9b69220601
source_review_id: "4204955814"
source_review_submitted_at: "2026-04-30T12:11:10Z"
---

# Issue 025: The expected payload shares the same client stack as the CLI path.
## Review Comment

`direct := NewClient(...)` and `deps.newClient: NewClient` both exercise the same `internal/cli` request/decoding code, so a regression in the client transport layer can make both sides wrong in the same way and still pass. Prefer a raw UDS/HTTP oracle or build the expected contract directly from the registry fixture.

As per coding guidelines, "Verify tests can fail when business logic changes".

Also applies to: 182-190

## Triage

- Decision: `VALID`
- Notes: The integration test currently computes expected list/search/invoke/toolset payloads with `direct := NewClient(...)`, while the CLI path under test also uses `NewClient` through `deps.newClient`. That shared transport/decoding stack can let a client-layer regression affect both expected and actual values. The fix is to remove the direct client oracle and build expected DTOs from the registry fixture and `core` contract conversion helpers instead.
- Resolution: Removed the direct `NewClient` oracle and built expected DTOs from the registry fixture plus core contract conversion helpers; verified with the focused integration test and `make verify`.
