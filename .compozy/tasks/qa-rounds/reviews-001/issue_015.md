---
status: resolved
file: internal/observe/observer_test.go
line: 98
severity: minor
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:bddf8f7e5a68
review_hash: bddf8f7e5a68
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 015: Isolate the live-source path in this test.
## Review Comment

Registering the session in `h.registry` makes this case pass even if the live-source lookup breaks, because `recoverSessionSnapshot` can still succeed through the registry fallback. Leave the registry unseeded here, or force the fallback path to fail, so this test actually proves the live-source branch.

As per coding guidelines, `**/*_test.go`: "Ensure tests can fail when business logic changes."

## Triage

- Decision: `VALID`
- Notes: The live-source recovery test seeds both the live source and the registry, so it can pass through the registry fallback even if live-source recovery is broken. Fix by leaving the registry unseeded for the live-source subtest in the consolidated recovery test.
