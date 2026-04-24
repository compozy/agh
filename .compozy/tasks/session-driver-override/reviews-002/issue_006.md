---
status: resolved
file: internal/session/log_capture_test.go
line: 20
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167289384,nitpick_hash:c246fdb1eb75
review_hash: c246fdb1eb75
source_review_id: "4167289384"
source_review_submitted_at: "2026-04-24T01:37:12Z"
---

# Issue 006: Use sync.RWMutex for read-heavy record inspection paths.
## Review Comment

`Records()` and `FindByMessage()` are read operations but currently take an exclusive lock. Switching to `RWMutex` + `RLock`/`RUnlock` improves concurrent read behavior.

Aligns with coding guideline: "Use `sync.RWMutex` for read-heavy, `sync.Mutex` for write-heavy shared state."

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: `Records()` and `FindByMessage()` only read shared state but currently take an exclusive lock. I will convert the handler lock to `sync.RWMutex` and use read locks on those paths while preserving write locking in `Handle`.
