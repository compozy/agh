# Issue 9

**Status:** - [x] RESOLVED
**Disposition:** - VALID
**Severity:** - MAJOR
**File:** - `internal/skills/watcher.go:139`
**Thread ID:** `PRRT_kwDOR5y4QM55CB-M`
**Comment URL:** - https://github.com/compozy/agh/pull/1#discussion_r3039610656

## Summary

Avoid making the first watcher scan an unconditional no-op when the registry may already be stale.

## Reviewer Comment

CodeRabbit reported a race where `LoadAll()` can finish, a skill changes, and the watcher's first scan records the new snapshot without refreshing the registry.

## Triage Notes

`VALID`: The watcher currently treats its first scan as pure initialization by copying the current filesystem snapshot and returning `changed=false`. If files change after `LoadAll()` but before that first scan, the registry remains stale and the watcher silently adopts the new state as baseline.

## Resolution

The registry now preserves the global file snapshot produced during `LoadAll`, and `NewWatcher` seeds itself from that baseline so first-scan races are detected correctly. Added `TestNewWatcherSeedsSnapshotsFromRegistryLoadAll` in `internal/skills/watcher_test.go`.
