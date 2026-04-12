---
status: resolved
file: internal/acp/client.go
line: 591
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4094005443,nitpick_hash:9d98c2a59390
review_hash: 9d98c2a59390
source_review_id: "4094005443"
source_review_submitted_at: "2026-04-11T17:12:00Z"
---

# Issue 002: Add debug logging when executable resolution fails.
## Review Comment

When `os.Executable()` or `filepath.EvalSymlinks()` fails, the function silently returns without setting `AGH_BIN` or modifying `PATH`. While the fallback behavior (returning unmodified env) is safe, the silent failure makes it difficult to diagnose issues where child processes cannot locate `agh`.

## Triage

- Decision: `invalid`
- Notes: `daemonMatchedEnv` is a pure environment-shaping helper with a safe fallback of returning the original env unchanged. Adding logging here would require threading logger state into low-level env assembly for a rare fallback path, and no current correctness failure or missing test signal points to a real bug.
