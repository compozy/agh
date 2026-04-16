---
status: resolved
file: internal/config/agent_resource.go
line: 48
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:b5e127ab2a2c
review_hash: b5e127ab2a2c
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 026: Use errors.Join() to preserve the validation error in the unwrap chain.
## Review Comment

Line 49 wraps `resources.ErrValidation` with `%w`, but formats the validation error using `%v`, which removes it from the error chain. This breaks `errors.As()` for the original validation failure. Use `errors.Join()` instead to preserve both errors, consistent with the codebase's error handling pattern.

## Triage

- Decision: `VALID`
- Notes:
  - The reviewed file has moved, but the current equivalent decode path is `internal/config/agent.go:287-297`.
  - Root cause: `decodeAgentFrontmatter` wraps `yamlErr` with `%w` but formats `tomlErr` with `%v`, which drops the TOML parser failure from the unwrap chain.
  - Intended fix: join both parser errors into the wrapped cause set and add regression coverage in `internal/config/agent_test.go`.
  - Result: updated the live agent decode path to join YAML and TOML failures and added `TestParseAgentDefPreservesParserErrorsInDecodeChain`; verified with `go test ./internal/config` and `make verify`.
