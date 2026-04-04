---
status: resolved
file: internal/daemon/daemon.go
line: 434
severity: medium
author: claude-code
provider_ref:
---

# Issue 013: TOCTOU race in boot() already-booted guard

## Review Comment

`boot()` acquires `d.mu`, checks if the daemon is already booted, then unlocks before performing the entire boot sequence (lines 434-439). The mutex is only re-acquired at the end (~line 594) to store state. If two goroutines call `boot()` concurrently, both could pass the guard check and proceed to acquire the lock file, open the registry, etc.

```go
d.mu.Lock()
if d.lock != nil || d.registry != nil || ... {
    d.mu.Unlock()
    return errors.New("daemon: already booted")
}
d.mu.Unlock()
// ... entire boot sequence runs unlocked ...
```

While `Run()` is currently the sole caller, this is a latent concurrency bug.

**Suggested fix:** Use a `booting` flag set under the mutex before releasing it:

```go
d.mu.Lock()
if d.lock != nil || d.booting {
    d.mu.Unlock()
    return errors.New("daemon: already booted")
}
d.booting = true
d.mu.Unlock()
```

## Triage

- Decision: `valid`
- Notes: `boot()` drops `d.mu` after checking the "already booted" guard and performs the whole boot sequence unlocked. Two concurrent callers can both pass the guard and race through startup. The current callers are narrow, but the function itself is not concurrency-safe.
