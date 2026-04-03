---
status: resolved
file: internal/plugins/pi/agh-hook.ts
line: 14
severity: low
author: claude-code
provider_ref:
---

# Issue 004: Silent error swallowing in Pi and OpenCode hook plugins

## Review Comment

Both the Pi and OpenCode TypeScript plugins use empty `catch {}` blocks that silently discard all errors from hook forwarding:

Pi (`pi/agh-hook.ts:13-14`):
```typescript
try {
    execSync(`${aghBin} hook-event --agent ${agent}`, { ... })
} catch {}
```

OpenCode (`opencode/agh-hook.ts:13`):
```typescript
try {
    await $`echo ${json} | ${aghBin} hook-event --agent ${agent}`.quiet()
} catch {}
```

While fire-and-forget semantics are correct (hook failures should not crash the agent), completely swallowing errors makes it impossible to diagnose hook connectivity issues. Add a minimal `console.error` or `console.warn` so operators can see failures in agent logs when debugging:

```typescript
} catch (err) {
    if (process.env.AGH_DEBUG) console.error("[agh-hook]", err)
}
```

## Triage

- Decision: `invalid`
- Notes:
  - The current hook integrations are intentionally best-effort and quiet. `internal/plugins/codex/agh-forwarder.sh` already redirects hook-forwarding stderr to `/dev/null` and returns success, which matches the design goal that hook delivery must not disturb the agent runtime.
  - Emitting `console.error` or `console.warn` from the Pi/OpenCode extensions would write unsolicited diagnostics into the agent/plugin process output stream during ordinary tool execution failures. In this codebase, that output is observable by the runtime and can interfere with otherwise non-fatal hook forwarding.
  - No repository documentation or existing tests require debug logging from hook plugins, and the suggestion also reaches beyond this scoped file into the OpenCode plugin. Closing as `invalid` because the proposed logging changes would alter the intended quiet failure mode rather than fix a demonstrated defect.
