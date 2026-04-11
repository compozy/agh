---
status: resolved
file: internal/network/envelope.go
line: 202
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:a0d2ba480701
review_hash: a0d2ba480701
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 017: Add compile-time assertions for the Body implementations.
## Review Comment

This file introduces the `Body` interface plus seven concrete body types, but there are no `var _ Body = ...` checks. Adding them makes receiver/signature drift fail at compile time instead of surfacing later through call sites.

As per coding guidelines, "Use compile-time interface verification: var _ Interface = (*Type)(nil)".

## Triage

- Decision: `valid`
- Root cause: the `Body` interface is new, but the concrete body types are not compile-time asserted against it, so receiver/signature drift would only show up later.
- Fix approach: add explicit `var _ Body = ...` assertions for each concrete body type.
