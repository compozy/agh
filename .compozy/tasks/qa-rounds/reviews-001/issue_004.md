---
status: resolved
file: internal/api/core/settings_test.go
line: 1009
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4188693296,nitpick_hash:157dafdb6b61
review_hash: 157dafdb6b61
source_review_id: "4188693296"
source_review_submitted_at: "2026-04-28T12:24:35Z"
---

# Issue 004: Assert the /sandboxes/:name response body too.
## Review Comment

This case currently proves only that the handler delegated to `CollectionSandboxes`. It would still pass if `:name` selection broke and the endpoint returned the wrong item. Please decode the response and assert the returned sandbox `name`/`profile` for this renamed route.

As per coding guidelines, "`**/*_test.go`: Always assert both HTTP status code AND response body (never status-code-only) in Go tests".

Also applies to: 1097-1208

## Triage

- Decision: `VALID`
- Notes: The collection handler table only checks HTTP 200 and service delegation for `/api/settings/sandboxes/local`; it does not decode the body, so a route-name lookup regression could return the wrong sandbox and still pass. Fix by adding response-body assertions, including the sandbox `name` and `profile.backend`.

## Resolution

- Added response-body assertions for settings collection handlers, including sandbox name and backend.
- Verified through targeted Go tests and `make verify`.
