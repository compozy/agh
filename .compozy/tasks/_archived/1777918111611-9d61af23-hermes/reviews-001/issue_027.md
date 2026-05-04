---
status: resolved
file: internal/session/liveness.go
line: 32
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:d735bfd4fa8a
review_hash: d735bfd4fa8a
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 027: Preserve an existing meta.Failure instead of overwriting it.
## Review Comment

These recovery branches always replace `meta.Failure` with a new `{Kind, Summary}` pair. If the persisted session already carried richer failure data, such as a crash-bundle path, restart recovery silently discards it. Prefer starting from `meta.Failure` and only filling missing `Kind`/`Summary` before normalizing.

Also applies to: 42-45, 52-55

## Triage

- Decision: `valid`
- Root cause: each interrupted recovery branch constructs a fresh `SessionFailure` and assigns it to `meta.Failure`, discarding existing normalized fields such as `CrashBundlePath` that may have been persisted by the original failure path.
- Fix approach: clone any existing failure, fill only missing `Kind` and `Summary` values needed for recovery classification, then normalize. Add coverage that an interrupted active/stopping/starting meta preserves an existing crash-bundle path while still repairing missing classification fields.
