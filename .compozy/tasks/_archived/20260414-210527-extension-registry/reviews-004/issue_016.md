---
status: resolved
file: internal/registry/installer.go
line: 243
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107849208,nitpick_hash:ae9b2fd067bc
review_hash: ae9b2fd067bc
source_review_id: "4107849208"
source_review_submitted_at: "2026-04-14T17:14:35Z"
---

# Issue 016: Consider making stale temp cleanup non-fatal.
## Review Comment

Failing the entire install because a cleanup of *old* stale temp directories failed seems overly strict. A user could be blocked from installing a package because an unrelated old temp directory can't be removed (e.g., permission issues on a leftover directory). Consider logging this as a warning instead of returning an error.

## Triage

- Decision: `valid`
- Root cause: installer startup can fail because cleanup of unrelated stale temp directories is treated as fatal even though the current install may still proceed safely.
- Evidence: [`internal/registry/installer.go`](internal/registry/installer.go) lines 341-342 return immediately on stale-directory removal failure before the new temp root is created.
- Fix plan: make stale temp removal best-effort for delete failures, keep the rest of the stale-directory scan strict, and add a regression test showing install still succeeds when stale cleanup cannot remove an old directory.
- Resolution: Stale temp directory removal is now best-effort for delete failures, with regression coverage for a successful install despite stale cleanup errors. Verified with package tests and `make verify`.
