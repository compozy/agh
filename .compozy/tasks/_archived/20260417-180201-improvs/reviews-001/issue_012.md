---
status: resolved
file: internal/registry/installer_checksum.go
line: 115
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:ee9f81b5bc7f
review_hash: ee9f81b5bc7f
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 012: Don’t canonicalize symlink targets before hashing.
## Review Comment

`filepath.Clean(target)` can collapse different symlink payloads into the same normalized value, weakening checksum fidelity. Hash the raw `Readlink` target (optionally slash-normalized only).

## Triage

- Decision: `VALID`
- Notes:
  The checksum path canonicalizes symlink targets with `filepath.Clean`, which
  collapses distinct link payloads into the same normalized string. That weakens
  checksum fidelity for install payloads. Plan: hash the raw `Readlink` target
  while keeping slash normalization only, and add a regression test that proves
  distinct symlink payloads produce distinct checksums.
