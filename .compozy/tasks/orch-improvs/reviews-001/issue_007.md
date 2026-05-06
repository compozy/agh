---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/cli/client.go
line: 2630
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:bf64d7db5ca4
review_hash: bf64d7db5ca4
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 007: Add nil checks for pointer request parameters before marshaling.
## Review Comment

Lines 2633, 2654, 2711, and 2751 accept `*...Request` pointer parameters and pass them through `doJSON`, which marshals using `interface{}`. In Go, a typed nil pointer stored in an interface{} compares non-nil, so `json.Marshal` will encode it as JSON `null` instead of rejecting it at the client. Add a nil check at the entry of each method to fail fast.

## Triage

- Decision: `valid`
- Notes:
  - The client methods still accept pointer request arguments and pass them straight into `doJSON`.
  - A typed nil pointer inside `interface{}` marshals as JSON `null`, so the client currently converts obvious caller mistakes into backend requests instead of failing locally.
  - Planned fix: add explicit nil checks to the affected client entry points and exercise them with focused client/CLI tests.
  - Resolved: the affected client helpers now reject nil pointer requests with explicit errors, and client tests cover all guarded methods.
