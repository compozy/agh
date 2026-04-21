---
status: resolved
file: internal/codegen/openapits/generate.go
line: 29
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4149203771,nitpick_hash:000420bee3ac
review_hash: 000420bee3ac
source_review_id: "4149203771"
source_review_submitted_at: "2026-04-21T16:10:13Z"
---

# Issue 003: Avoid hardcoding external codegen command/tool names.
## Review Comment

`bunx`, `openapi-typescript`, and `oxfmt` are embedded in code. Consider injecting these via options/config to keep tooling portable and environment-safe.

As per coding guidelines, "Never hardcode configuration — use TOML config or functional options."

## Triage

- Decision: `invalid`
- Reasoning: this internal package exists specifically to run the repository's fixed OpenAPI generation toolchain (`bunx openapi-typescript` followed by `bunx oxfmt`). Those executable names are part of the implementation contract of this private codegen helper, not deploy-time configuration. Injecting them via options would broaden the API and test matrix without a real caller requirement or portability bug to fix.
