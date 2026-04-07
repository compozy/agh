---
status: resolved
file: internal/store/globaldb/global_db.go
line: 75
severity: high
author: claude-code
provider_ref:
---

# Issue 004: GlobalDB has no close-state guard

## Review Comment

Unlike `SessionDB` which has an `atomic.Int32` state guard and `acceptMu`, `GlobalDB` has no close-state tracking. Calling `Close()` then any method (e.g., `RegisterSession`) will use the closed `*sql.DB`, producing obscure `"sql: database is closed"` errors instead of a clear `ErrClosed`. Calling `Close()` twice produces an error from `db.Close()`. The `checkReady` method only checks for nil receiver and nil context, not closed state.

**Fix:** Add an `atomic.Int32` or `sync.Once` for close tracking and check it in `checkReady`:

```go
type GlobalDB struct {
    db     *sql.DB
    path   string
    now    func() time.Time
    closed atomic.Int32
}

func (g *GlobalDB) checkReady(ctx context.Context, action string) error {
    if g == nil { return errors.New("store: global database is required") }
    if g.closed.Load() != 0 { return store.ErrClosed }
    if ctx == nil { return errors.New("store: context is required") }
    return nil
}
```

## Triage

- Decision: `valid`
- Root cause: `GlobalDB` methods only guard nil receivers and nil contexts. After `Close`, callers fall through to the closed `*sql.DB`, which produces opaque downstream errors instead of the store-level closed sentinel.
- Fix approach: Add explicit close-state tracking to `GlobalDB`, check it from `checkReady`, make repeated `Close` calls idempotent, and cover the closed-state behavior in tests.
- Resolution: Implemented with closed-state guards and regression coverage; full repository verification passed.
