---
status: resolved
file: extensions/bridges/discord/provider_test.go
line: 1540
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4116207448,nitpick_hash:cbfcc3e0b66e
review_hash: cbfcc3e0b66e
source_review_id: "4116207448"
source_review_submitted_at: "2026-04-15T19:14:59Z"
---

# Issue 004: Move these helpers behind t.Helper()/shared testutil instead of unsafe field mutation.

## Review Comment

`bridgepkgToWebhookRequest` hides a marshal failure, and `injectedDiscordSession`/`setUnexportedField` couple the test to private `bridgesdk.Session` layout via `reflect`+`unsafe`. A shared helper that takes `*testing.T`, calls `t.Helper()`, and owns this setup would be much safer.

As per coding guidelines, "Use `t.Helper()` on test helper functions" and "Use shared test helpers from `internal/testutil` and `internal/api/testutil`".

## Triage

- Decision: `valid`
- Notes:
  - `bridgepkgToWebhookRequest` currently ignores `json.Marshal` errors, which can hide malformed test inputs and make failures harder to diagnose.
  - The session-construction helper also lacks `t.Helper()` and exposes raw reflection-based field injection directly in the test file.
  - Root cause: the local test helpers are too loose about failure reporting and encapsulation.
  - Outcome: converted the helpers to `t.Helper()`-annotated helpers, fail-fast on marshal errors, and contained the session-field injection behind a single helper instead of exposing the low-level mutation utility to the rest of the file. Verified with `go test ./extensions/bridges/discord ./extensions/bridges/gchat` and `make verify`.
