---
status: resolved
file: internal/cli/hooks.go
line: 83
severity: medium
author: claude-reviewer
---

# Issue 122: Unbounded io.ReadAll on stdin in hook-event command



## Review Comment

The `runHookEventCommand` function reads the entire stdin payload into memory at line 83 with no size limit:

```go
rawPayload, err := io.ReadAll(cmd.InOrStdin())
```

If a caller pipes a very large file or an infinite stream into `agh hook-event`, this will consume unbounded memory and could cause an OOM condition. Hook events are expected to be small JSON payloads (tool use metadata), so a reasonable limit should be enforced.

The same pattern exists in `readOptionalCommandInput` (`internal/cli/roles.go` line 247) which is used by `roles create` to read the system prompt from stdin.

**Suggested fix:** Use `io.LimitReader` to cap the maximum payload size:

```go
const maxHookPayloadSize = 1 << 20 // 1 MB
rawPayload, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxHookPayloadSize))
```

For the roles/playbooks stdin reads, a similar limit would be appropriate (e.g., 1 MB for system prompts, 10 MB for playbook content).

## Triage

- Decision: `valid`
- Notes: Confirmed in `runHookEventCommand`: the CLI reads stdin with unbounded `io.ReadAll(cmd.InOrStdin())`. Hook payloads are expected to be compact JSON envelopes, so an explicit upper bound is appropriate and can share the same limited-read pattern as the role-input fix.
