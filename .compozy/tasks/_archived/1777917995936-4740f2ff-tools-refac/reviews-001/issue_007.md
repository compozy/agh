---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/core/tools.go
line: 288
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4204955814,nitpick_hash:44cecd3cfb65
review_hash: 44cecd3cfb65
source_review_id: "4204955814"
source_review_submitted_at: "2026-04-30T12:11:10Z"
---

# Issue 007: Add doc comments for unexported helpers introduced in this file.
## Review Comment

This file adds many unexported functions without comments (`toolDescriptorPayload`, `toolBackendPayload`, `bindToolSearch`, `toolErrorLayer`, etc.). Please add short intent-focused comments for maintainability and policy compliance.

As per coding guidelines, "Comments in Go must explain the 'why' and 'what', not just 'what'. Unexported identifiers must have a comment."

## Triage

- Decision: `INVALID`
- Notes: The loaded canonical AGH Go style says comments should be sparse, explain non-obvious WHY/constraints, and must not restate the WHAT. It does not require comments for every unexported helper. Most listed helpers (`toolDescriptorPayload`, `toolBackendPayload`, `bindToolSearch`, `toolErrorLayer`) are straightforward local converters/binders where comments would repeat identifiers and conflict with the active style rule. No code change is appropriate for this issue as written.
