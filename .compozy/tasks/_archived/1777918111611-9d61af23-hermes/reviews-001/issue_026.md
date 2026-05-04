---
status: resolved
file: internal/session/crash_bundle.go
line: 173
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:1ca82812a7c5
review_hash: 1ca82812a7c5
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 026: Preserve the timestamp when truncating crash bundle names.
## Review Comment

`base[:crashBundleNameMaxBytes]` trims from the front of the assembled name, so a long `sessionID` can chop off the `UnixNano` suffix entirely. Repeated failures for the same session/kind then reuse the same filename and overwrite earlier crash bundles.

## Triage

- Decision: `valid`
- Root cause: `crashBundleFileName` appends the timestamp to the generated base name and then truncates the front of that full string. A sufficiently long session ID can consume the limit and remove the timestamp suffix, causing repeated failures with the same session/kind to collide and overwrite bundles.
- Fix approach: reserve space for the kind/timestamp suffix before truncating the session-name prefix, then add a regression test proving long session IDs preserve distinct timestamp-bearing filenames.
