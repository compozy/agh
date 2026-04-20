---
status: resolved
file: scripts/inspect-acp-toolcalls.go
line: 252
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4058705969,nitpick_hash:017497673e5e
review_hash: 017497673e5e
source_review_id: "4058705969"
source_review_submitted_at: "2026-04-04T17:43:33Z"
---

# Issue 005: Consider logging marshal errors instead of inline error messages.
## Review Comment

The `renderBlocks` function silently continues on marshal errors, printing only to stdout. For a debugging/inspection tool, this is acceptable, but the error message format could be more consistent.

## Triage

- Decision: `invalid`
- Reasoning: `renderBlocks` is part of a human-readable inspection transcript that this script intentionally writes to stdout alongside the block payloads, prompt, and command preview. Routing per-block marshal failures to a separate logger or stream would detach the error from the block/update it belongs to and make the inspection output harder to follow. Continuing after one marshal failure is also intentional here, because this tool is meant to inspect ACP behavior without hiding later updates behind a single malformed block.
- Resolution: no code change is required in `scripts/inspect-acp-toolcalls.go`; the review comment is a formatting preference, not a correctness or observability defect on the current branch.
- Verification: `make verify` (`0 issues`; `DONE 2416 tests, 1 skipped`; build succeeded; `All verification checks passed`).
