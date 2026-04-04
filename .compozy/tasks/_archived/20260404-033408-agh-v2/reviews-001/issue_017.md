---
status: resolved
file: internal/config/provider.go
line: 38
severity: medium
author: claude-code
provider_ref:
---

# Issue 017: npx -y with caret version ranges allows supply-chain attacks

## Review Comment

Several builtin provider commands use `npx -y @package@^x.y.z` (lines 38-69), which: (1) `-y` auto-confirms installation without user prompt, and (2) `^` allows any compatible minor/patch version. If a malicious version is published to npm within the caret range, the daemon would auto-download and execute it with the daemon's full privileges and access to API keys.

**Suggested fix:** Pin exact versions (e.g., `@0.24.2` instead of `@^0.24.2`) for default builtin providers. Users can override versions via config. Add a comment explaining the security tradeoff.

## Triage

- Decision: `valid`
- Notes: The built-in `npx` provider commands use floating version specifiers for ACP launchers. That allows automatic drift to newly published packages, which is an unnecessary supply-chain risk for default runtime commands. The built-ins should be pinned to exact versions.
