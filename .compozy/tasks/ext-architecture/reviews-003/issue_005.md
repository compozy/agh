---
status: resolved
file: internal/api/core/handlers.go
line: 629
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Q1zn,comment:PRRC_kwDOR5y4QM620raj
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
rg -n 'type HomePaths struct|ResolveHomePaths|ResolvePath|HomeDir|UserHome' internal/config
rg -n 'resolveUserHomeDir|os\.UserHomeDir' internal/api/core/handlers.go internal/api/httpapi/handlers_test.go internal/api/core/more_coverage_test.go
```

Repository: compozy/agh

Length of output: 10300

---

🏁 Script executed:

```shell
head -30 internal/api/core/handlers.go | grep -E "^import|^package"
```

Repository: compozy/agh

Length of output: 76

---

🏁 Script executed:

```shell
head -50 internal/api/core/handlers.go | tail -30
```

Repository: compozy/agh

Length of output: 1161

---

🏁 Script executed:

```shell
sed -n '595,629p' internal/api/core/handlers.go
```

Repository: compozy/agh

Length of output: 1000

---

🏁 Script executed:

```shell
sed -n '30,100p' internal/api/core/handlers.go | grep -A 30 "type BaseHandlers struct"
```

Repository: compozy/agh

Length of output: 1123

---

🏁 Script executed:

```shell
sed -n '36,50p' internal/config/home.go
```

Repository: compozy/agh

Length of output: 333

---

🏁 Script executed:

```shell
sed -n '52,72p' internal/config/home.go
```

Repository: compozy/agh

Length of output: 604

---

🏁 Script executed:

```shell
sed -n '1,30p' internal/api/core/handlers.go | grep -E "^import|aghconfig"
```

Repository: compozy/agh

Length of output: 118

---



</details>

**Use `h.HomePaths.HomeDir` instead of `resolveUserHomeDir()`.**

The handler already receives `HomePaths` in its config, and the `.HomeDir` field is available. The custom `resolveUserHomeDir()` function calls `os.UserHomeDir()` directly, which fails entirely in service environments where `HOME`/`USERPROFILE` is unset or sanitized, unnecessarily failing the entire `/daemon/status` endpoint.

Either use `h.HomePaths.HomeDir` directly (already resolved with proper AGH_HOME override support), or call `aghconfig.ResolveHomeDir()` if you need to recompute it. Do not rely on `os.UserHomeDir()` alone or make the endpoint fail on this optional field.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/handlers.go` around lines 595 - 629, The /daemon/status
handler calls resolveUserHomeDir(), which uses os.UserHomeDir() and can fail in
sanitized service environments; replace that call to use the already-resolved
path from the handler config (h.HomePaths.HomeDir) or call
aghconfig.ResolveHomeDir() if recomputation is required. Update the code that
sets UserHomeDir in the DaemonStatusPayload to use h.HomePaths.HomeDir (or the
result of aghconfig.ResolveHomeDir()) and remove or deprecate
resolveUserHomeDir() usage from this handler so the endpoint no longer fails
when HOME/USERPROFILE are unset.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The reported reliability problem is real: `/daemon/status` should not fail just because `os.UserHomeDir()` is unavailable in a sanitized environment.
  - The suggested exact replacement is not correct as written, because `h.HomePaths.HomeDir` is the AGH home directory (for example `~/.agh` or `AGH_HOME`), not the raw user home directory exposed in `daemon.user_home_dir`.
  - Root cause: `DaemonStatus` treats user-home resolution as mandatory even though the payload field is optional UI convenience data.
  - Fix plan: make user-home resolution best-effort instead of fatal, preserve the actual user-home semantics when resolvable, derive a safe fallback from the canonical `.../.agh` layout when available, and add a focused regression test for the fallback behavior.
  - Implemented: `/daemon/status` now resolves `daemon.user_home_dir` on a best-effort basis, logs when lookup is unavailable, and falls back to the parent of a canonical `.../.agh` path when that inference is safe; the handler no longer returns HTTP 500 for this optional field.
  - Test coverage: added the minimal out-of-scope file `internal/api/core/handlers_internal_test.go` because the scoped HTTP handler test cannot reliably force `os.UserHomeDir()` failure across platforms.
  - Verification: `go test ./cmd/agh-codegen ./internal/acp ./internal/api/core ./internal/api/httpapi ./internal/api/spec -count=1`; `make verify`.
