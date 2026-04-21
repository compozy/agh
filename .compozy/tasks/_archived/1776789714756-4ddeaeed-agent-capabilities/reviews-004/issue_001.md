---
status: resolved
file: internal/config/agent_capabilities_test.go
line: 131
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4141178396,nitpick_hash:ef698a8f269b
review_hash: ef698a8f269b
source_review_id: "4141178396"
source_review_submitted_at: "2026-04-20T15:11:23Z"
---

# Issue 001: Consider using subtests for each agent assertion.
## Review Comment

Each agent check (coder, pairer, reviewer) is logically independent. Using `t.Run` would improve failure isolation and make it clearer which precedence case failed.

As per coding guidelines: "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests".

## Triage

- Decision: `valid`
- Root cause: `TestLoadWorkspaceAgentDefsPreservesPrecedenceWithCapabilities` currently validates three independent precedence cases (`coder`, `pairer`, `reviewer`) inside one top-level test body, so the first failure hides the rest and does not follow the repo's default table/subtest pattern.
- Why this is valid: the shared fixture setup is fine, but the assertions themselves are logically independent outcomes and should be isolated with `t.Run(...)` cases for clearer failure reporting and easier extension.
- Fix approach: keep the shared agent-loading setup, then move the per-agent assertions into a small table of subtests so each precedence case fails independently without changing behavior.

## Resolution

- Reworked `TestLoadWorkspaceAgentDefsPreservesPrecedenceWithCapabilities` into three `Should...` subtests so the `coder`, `pairer`, and `reviewer` precedence assertions fail independently while reusing the same fixture setup.
- Verification:
  - `go test ./internal/config -count=1`
  - `make verify`
