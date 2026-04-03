---
status: resolved
file: internal/drivers/opencode/opencode.go
line: 1201
severity: low
author: claude-reviewer
---

# Issue 161: OpenCode driver's isReadyOutput pattern is too broad and fragile



## Review Comment

The `isReadyOutput` function for the OpenCode driver matches very generic substrings:

```go
func isReadyOutput(output []byte) bool {
    text := strings.ToLower(string(output))
    return strings.Contains(text, "opencode") ||
        strings.Contains(text, "press ? for help") ||
        strings.Contains(text, "server.connected") ||
        strings.Contains(text, "session selector")
}
```

The first check, `strings.Contains(text, "opencode")`, would match on any output containing the word "opencode", including error messages like "opencode: failed to start" or "opencode configuration invalid". This could cause the driver to falsely report readiness when the process is actually failing during startup.

The same concern applies less severely to "session selector" which could appear in error context.

**Suggested fix**: Use more specific patterns that only match actual readiness indicators, not substrings that could appear in error messages. For example, check for the combination of patterns or use a regex that matches the actual TUI prompt format.

## Triage

- Decision: `valid`
- Notes:
  - `isReadyOutput` currently treats any output containing the substring `opencode` as readiness, which can misclassify startup failures or configuration errors as a successful boot.
  - That is a concrete false-positive readiness bug because `DetectReady` can return success before the TUI is actually usable.
  - The fix should narrow readiness matching to specific UI markers instead of broad branding text.
  - Resolution: narrowed OpenCode readiness detection to actual UI markers and added regression coverage for false-positive error output in `internal/drivers/opencode/opencode_test.go`.
