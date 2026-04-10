---
status: resolved
file: internal/daemon/boot.go
line: 367
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAaS,comment:PRRC_kwDOR5y4QM62zlsd
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Synchronize access to `state.extensions` before exposing it to async hook rebuilds.**

This provider closes over `state.extensions`, but `bootExtensions()` writes that field later and the skills watcher can invoke `hooks.Rebuild()` asynchronously once `bootHooks()` returns. That introduces a real read/write race during boot when skills are enabled.


As per coding guidelines, "Use `sync.RWMutex` for read-heavy, `sync.Mutex` for write-heavy shared state" and "Run tests with `-race` flag before committing — zero tolerance for race conditions".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/boot.go` around lines 364 - 367, The provider closure passed
to extensionDeclarationProvider currently reads state.extensions without
synchronization while bootExtensions() may write it and hooks.Rebuild() can run
async, causing a race; add a sync.RWMutex (e.g., state.extMu) to the state
struct, use extMu.RLock()/RUnlock() inside the extensionDeclarationProvider
closure when accessing/returning state.extensions, and use extMu.Lock()/Unlock()
around the assignment in bootExtensions() (and any other writers) so reads and
writes are properly synchronized for hooks.Rebuild(),
extensionDeclarationProvider, and bootExtensions.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `bootHooks` closes over `state.extensions`, and `bootExtensions` assigns that field later while the skills watcher can trigger asynchronous `hooks.Rebuild()` calls. That is a real read/write race under `-race`. I will synchronize access to the boot-time extension runtime reference and add a concurrent regression test.
