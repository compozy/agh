---
status: resolved
file: internal/cli/config.go
line: 1146
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:6fe262ddd3c8
review_hash: 6fe262ddd3c8
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 006: Replace strings.Fields() with shellquote.Split() to properly handle EDITOR/VISUAL values with spaces and quotes.
## Review Comment

`strings.Fields()` splits on any whitespace without respecting shell quoting or escapes. This breaks real-world cases like:
- `EDITOR="/path with spaces/vim"` → incorrectly splits into separate arguments
- `EDITOR="vim --option 'value with spaces'"` → loses quoted context

The codebase already uses `github.com/kballard/go-shellquote` extensively for this exact pattern (see `internal/acp/client.go` and `internal/session/manager_hooks.go`). Replace with `shellquote.Split()` to handle POSIX shell word-splitting rules correctly.

## Triage

- Decision: `VALID`
- Notes: `runConfigEditor` uses `strings.Fields`, which breaks valid `VISUAL`/`EDITOR` commands containing quoted arguments or spaces in the executable path. The module already depends on `github.com/kballard/go-shellquote`; use `shellquote.Split` and add coverage for quoted editor commands.
