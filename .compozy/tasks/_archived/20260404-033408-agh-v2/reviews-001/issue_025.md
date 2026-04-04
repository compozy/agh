---
status: resolved
file: internal/config/config.go
line: 380
severity: low
author: claude-code
provider_ref:
---

# Issue 025: godotenv.Load mutates global process environment

## Review Comment

`godotenv.Load()` (called at line ~380 in the `loadDotEnv` path) uses `os.Setenv` under the hood, permanently mutating the process's environment for all goroutines. This is problematic because: (1) the `.env` file is loaded from the workspace root (user-controlled), (2) multiple calls to `Load()` with different workspace roots would accumulate variables, and (3) there is no way to "unload" these variables.

**Suggested fix:** Use `godotenv.Read()` instead to get a map of values, then only extract the specific keys needed (e.g., `AGH_HOME`). Or document that `Load` must only be called once per process lifetime and add a `sync.Once` guard.

## Triage

- Decision: `invalid`
- Notes: Loading `.env` into the process environment is intentional in the current bootstrap flow because `Load()` relies on environment-backed home resolution and runtime credentials, and later subprocesses inherit those values. Replacing it with a local-only map in `config.go` would change established runtime behavior well beyond this review item. The report identifies a design tradeoff, but not a concrete bug in the current usage model.
