---
status: resolved
file: internal/cli/agent.go
line: 89
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4188693296,nitpick_hash:fc0a36b91f40
review_hash: fc0a36b91f40
source_review_id: "4188693296"
source_review_submitted_at: "2026-04-28T12:24:35Z"
---

# Issue 007: Consolidate duplicated workspace-flag parsing helper.
## Review Comment

`agentWorkspaceFlag` duplicates `skillWorkspaceFlag` behavior in the same package. Reusing one helper avoids divergence in future validation/error-message changes.

## Triage

- Decision: `VALID`
- Notes: `agentWorkspaceFlag` and `skillWorkspaceFlag` contain identical flag parsing, trimming, and empty explicit flag validation. This duplication can diverge. Fix by consolidating both call sites on one shared workspace flag helper.

## Resolution

- Consolidated agent and skill workspace flag parsing on `commandWorkspaceFlag`.
- Verified through targeted CLI tests and `make verify`.
