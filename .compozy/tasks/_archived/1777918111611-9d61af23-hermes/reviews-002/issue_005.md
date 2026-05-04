---
status: resolved
file: internal/cli/mcp_auth.go
line: 55
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLiX,comment:PRRC_kwDOR5y4QM67SmDc
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Don't swallow the DB close error on startup rollback.**

If `mcpauth.NewService` fails, `_ = db.Close(ctx)` drops the rollback failure and the returned error also lacks operation context. That makes constructor failures harder to diagnose and violates the repo's explicit-error policy.

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification" and "`**/*.go`: Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/mcp_auth.go` around lines 47 - 55, When mcpauth.NewService(db)
fails, don't ignore the db.Close(ctx) error or return the raw err; capture the
close error and return a wrapped error with context. Replace `_ = db.Close(ctx)`
with code that does `closeErr := db.Close(ctx)` and if closeErr != nil return
nil, nil, fmt.Errorf("mcpauth.NewService failed: %w; db.Close failed: %v", err,
closeErr)`, otherwise return nil, nil, fmt.Errorf("mcpauth.NewService failed:
%w", err). Refer to globaldb.OpenGlobalDB, mcpauth.NewService, db.Close, ctx and
homePaths.DatabaseFile to locate the relevant block.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `defaultMCPAuthClient` ignores `db.Close(ctx)` when `mcpauth.NewService` fails and returns the raw constructor error, losing rollback diagnostics and operation context.
- Fix approach: close the DB explicitly on service construction failure, join close failure context into the returned error, and wrap the constructor failure with MCP auth startup context.
