---
status: resolved
file: internal/cli/client.go
line: 574
severity: low
author: claude-code
provider_ref:
---

# Issue 023: Unbounded io.ReadAll on error responses from daemon

## Review Comment

`readAPIError` at line 574 calls `io.ReadAll(response.Body)` without any size limit. If the daemon returns a pathologically large error response (or a misbehaving proxy sits between CLI and daemon), this could consume unbounded memory.

**Suggested fix:** Use `io.LimitReader`:

```go
body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20)) // 1MB
```

## Triage

- Decision: `valid`
- Notes: `readAPIError()` uses unbounded `io.ReadAll(response.Body)`. A malicious or broken peer can force unnecessary memory growth on the CLI error path. The body read should be capped.
