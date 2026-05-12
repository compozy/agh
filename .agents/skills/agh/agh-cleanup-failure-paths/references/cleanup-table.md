# Resource → Cleanup Pairings

Canonical pairs to write defer-adjacent (or release explicitly on every error path).

| Allocation | Cleanup |
|------------|---------|
| `ctx, cancel := context.WithCancel(parent)` | `defer cancel()` next line |
| `ctx, cancel := context.WithTimeout(parent, d)` | `defer cancel()` next line |
| `ctx, cancel := context.WithDeadline(parent, t)` | `defer cancel()` next line |
| `detached := context.WithoutCancel(ctx)` | re-attach deadline if needed; pair with explicit `CancelPrompt`/`Stop` API |
| `f, err := os.Open(...)` | `defer f.Close()` next line; check Close error if write |
| `tx, err := db.Begin(...)` | `defer tx.Rollback()` next line; explicit `tx.Commit()` on success |
| `lis, err := net.Listen(...)` | `defer lis.Close()` next line |
| `resp, err := client.Do(req)` | `defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()` next line |
| `mu.Lock()` | `defer mu.Unlock()` next line |
| `wg.Add(1)` then `go func() { ... }()` | `defer wg.Done()` inside the goroutine |
| `proc, err := acp.Start(...)` | `defer func() { stopCtx, c := context.WithTimeout(...); proc.Stop(stopCtx); c() }()` |
| `claim, err := task.ClaimNextRun(...)` | `defer func() { if err != nil { task.Release(claim, "abort") } }()` |
| `lease, err := lease.Acquire(...)` | `defer lease.Release()` next line |
| `regHandle := registry.Register(...)` | `defer regHandle.Unregister()` next line |
| `tmp, err := os.CreateTemp(...)` | `defer os.Remove(tmp.Name())` and `defer tmp.Close()` |
| `entry := observe.StartSpan(...)` | `defer entry.End(err)` next line; pass current `err` into End |

## Cancel-then-grace stop semantics

Subprocess stop respects both context cancellation AND graceful shutdown:

```go
defer func() {
    stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
    defer stopCancel()
    if err := proc.Shutdown(stopCtx); err != nil {
        // forced kill below
    }
    select {
    case <-proc.Done():
        return
    case <-stopCtx.Done():
        proc.Kill()
    }
}()
```

For ACP wrappers (npm exec → node → native), kill the entire process group — not just the wrapper. See `internal/procutil/process_group_unix.go` for the helper. Windows uses forced-exit fallback (`internal/procutil/process_tree_windows.go`).

## Sequencing rules

- **Public flip after private cleanup.** When state has both a public-visible flip (e.g., registry-visible disable) and a private resource (in-memory hook unregister), the public flip happens AFTER the private cleanup, not before. See `docs/_memory/_synthesis.md` extension manager L3 lesson.
- **Boot recovery before scheduler accepts traffic.** When initializing the daemon, recovery completes before claim/wake traffic begins. (See autonomy `_techspec.md` lease invariants.)
- **Reaper releases leases before stopping a child session.** Lease release is a precondition to stop, not a side effect.

## Drain rules (HTTP)

- Always drain even on non-2xx responses; otherwise the connection is poisoned for keep-alive.
- For SSE consumption, the reader handles drain on its own; do not double-drain.
