---
status: resolved
file: internal/config/agent.go
line: 274
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58BM39,comment:PRRC_kwDOR5y4QM65LrXj
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Persist the normalized capability catalog back onto `AgentDef`.**

`normalizeCapabilityCatalog()` returns the canonicalized catalog, but this code only checks the error and throws the normalized value away. That means `Validate()` accepts capability definitions with whitespace/empty entries without ever updating `a.Capabilities`, so callers keep the pre-normalized data even though validation succeeded. This is especially visible on decode/round-trip paths that now expect canonicalized capabilities after validation.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/agent.go` around lines 272 - 274, The code calls
normalizeCapabilityCatalog(a.Capabilities, "agent.capabilities") but discards
its normalized return value; update AgentDef by assigning the returned canonical
catalog back to a.Capabilities (handling the returned error as before) so
Validate() persists the cleaned/trimmed capability list; specifically, capture
the first return value from normalizeCapabilityCatalog and set a.Capabilities =
<normalizedCatalog> before returning nil or the error.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `AgentDef.Validate()` calls `normalizeCapabilityCatalog(a.Capabilities, "agent.capabilities")` only to surface validation errors and drops the returned canonical catalog.
- Impact: callers that validate an agent with a non-nil capability catalog keep whitespace, empty list items, and other pre-normalized values even though validation succeeds.
- Fix plan: persist the normalized catalog back through the shared `Capabilities` pointer during validation and add a regression test that proves `Validate()` normalizes caller-visible capability data.
- Resolution: `internal/config/agent.go` now writes the normalized catalog back into the existing `Capabilities` pointer target during `AgentDef.Validate()`, and `internal/config/agent_capabilities_test.go` now covers in-place normalization of trimmed fields and filtered string lists.
- Verification:
  - `go test ./internal/config`
  - `make verify`
