---
status: resolved
file: internal/daemon/daemon.go
line: 875
severity: high
author: claude-code
provider_ref:
---

# Issue 001: Dream consolidation ignores workspace context

## Review Comment

The dream spawner in `makeDreamSpawner()` uses `os.Getwd()` (line 875) to determine the workspace for dream consolidation sessions. For a background daemon process, this returns whatever directory the daemon was started from (often `/` or `$HOME`), not the workspace where agent sessions actually ran.

Additionally, `runtimeDreamTrigger.Trigger()` (line 120) explicitly discards the workspace parameter with `_ string`, meaning the workspace passed from the HTTP/UDS consolidation endpoint is silently ignored.

**Impact**: Workspace-scoped dream memories (`project`, `reference` types) would be written to `<daemon_cwd>/.agh/memory/` instead of `<actual_workspace>/.agh/memory/`. This breaks the core cross-agent knowledge sharing story for workspace-scoped memories.

**Suggested fix**:
1. Pass the workspace through `Trigger()` → `Run()` → `SessionSpawner`
2. In `makeDreamSpawner`, use the workspace parameter instead of `os.Getwd()`
3. For the periodic ticker (no explicit workspace), derive workspace(s) from recent sessions via `sessions.ListAll()` and run consolidation per workspace

```go
// runtimeDreamTrigger should pass workspace through
func (t runtimeDreamTrigger) Trigger(ctx context.Context, workspace string) (bool, string, error) {
    // ... gate checks ...
    if err := t.service.Run(ctx, func(ctx context.Context, goal, prompt string) error {
        return t.spawner(ctx, goal, prompt, workspace) // pass workspace
    }); err != nil {
```

## Triage

- Decision: `valid`
- Root cause: `runtimeDreamTrigger.Trigger()` discards the caller-provided workspace, and the daemon dream spawner falls back to `os.Getwd()`. For a background daemon that means manual consolidation requests and automatic runs do not preserve the originating project workspace.
- Evidence: `internal/daemon/daemon.go` ignores the `workspace` argument in `Trigger`, `makeDreamSpawner()` resolves the workspace with `os.Getwd()`, and queued dream checks only carry a reason string with no workspace context.
- Fix approach: Thread workspace through the daemon dream path, update the spawner to honor an explicit workspace, and for automatic runs without an explicit workspace derive recent non-dream workspaces from session metadata instead of daemon cwd.
- Resolution: Implemented explicit workspace propagation through the daemon dream trigger/run path, queued session-stop checks with their session workspace, and changed automatic dream spawning to derive recent workspaces from session metadata when no explicit workspace is supplied.
