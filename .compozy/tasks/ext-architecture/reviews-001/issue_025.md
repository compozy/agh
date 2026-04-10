---
status: resolved
file: internal/extension/manager.go
line: 1349
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAak,comment:PRRC_kwDOR5y4QM62zlsv
---

# Issue 025: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Do not inherit the daemon’s full environment into extension subprocesses by default.**

Starting from `os.Environ()` leaks every process secret/credential to user, workspace, and marketplace extensions unless each variable is manually scrubbed elsewhere. This needs an explicit allowlist or opt-in passthrough model instead of unconditional inheritance.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/manager.go` around lines 1328 - 1349, resolveEnvMap
currently appends os.Environ(), which unintentionally leaks the daemon's full
environment into extension subprocesses; change resolveEnvMap to stop inheriting
the full environment by default and instead construct the subprocess env only
from the resolved env map plus a minimal safe baseline (e.g., PATH, HOME, and
other intentionally safe vars) or require explicit opt-in/passthrough (e.g., a
Manager.AllowHostEnv flag or a special env key) before merging os.Environ();
update usages of resolveEnvMap and document the opt-in flag so extensions cannot
receive daemon secrets unless explicitly allowed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `resolveEnvMap` currently starts from `os.Environ()`, so every daemon secret becomes visible to extension subprocesses by default. That violates least-privilege for user, workspace, and marketplace extensions.
  Fix approach: stop inheriting the full host environment by default and instead build a minimal safe baseline plus the manifest-declared overrides.
  Additional test scope needed: `internal/extension/manager_test.go` is outside the batch file list but is the minimal place to verify the subprocess launch environment.
