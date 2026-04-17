---
status: resolved
file: internal/daemon/daemon_bridge_extension_integration_test.go
line: 28
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130477247,nitpick_hash:535cdb302b04
review_hash: 535cdb302b04
source_review_id: "4130477247"
source_review_submitted_at: "2026-04-17T16:34:12Z"
---

# Issue 003: Consider using t.Run subtests for distinct test phases.
## Review Comment

The test covers multiple distinct phases (extension installation, bridge creation, first ingress, second ingress, conformance validation) that would benefit from `t.Run("Should...")` grouping. This improves failure diagnostics by clearly indicating which phase failed.

As per coding guidelines: "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `invalid`
- Reasoning: this test is one sequential end-to-end scenario over a shared runtime harness, one bridge instance, and one reused session route. Splitting it into synthetic `t.Run` phases would not isolate state or improve concurrency; it would only add nesting around a single linear flow.
- Repository fit: the workspace guidance says table-driven tests should use subtests by default, not that every single-case integration narrative must be wrapped in `t.Run("Should...")`.
- Resolution: no code change required.
