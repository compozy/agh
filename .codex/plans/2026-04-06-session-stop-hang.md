# Fix Session Stop Hanging on ACP Wrapper Children

## Summary

- Reproduced on April 6, 2026 against an isolated daemon on `127.0.0.1:23230`: `POST /api/sessions` succeeded, `DELETE /api/sessions/:id` hung for 66-116 seconds, and session metadata stayed `stopping`.
- Process inspection showed the runtime shape `npm exec @zed-industries/codex-acp -> node -> native codex-acp`. The stop path only terminated the wrapper process, while descendant processes kept the ACP stdio pipes open, so `cmd.Wait()` never completed and `Manager.Stop` never finalized the session.
- Scope is backend lifecycle only. The existing web sidebar/query logic is already reflecting backend state correctly. After the fix, stop should complete promptly and the session should become `stopped`; it may still remain listed, which matches current product behavior.

## Key Changes

- Add ACP subprocess-tree lifecycle helpers under `internal/acp/` to start runtime commands in their own process group on Unix and terminate the full group with `SIGTERM`, then `SIGKILL` on timeout. Keep a non-Unix fallback that preserves direct-process termination so the package still builds everywhere.
- Update `internal/acp/client.go` to use those helpers for agent runtimes instead of relying on `exec.CommandContext` plus top-level `Process.Kill()` only. `Driver.Stop` should still send ACP `session/cancel` first, then terminate the process group, then wait for `proc.Done()` so session finalization stays deterministic.
- Apply the same process-group launch/kill behavior to ACP terminal subprocesses in `internal/acp/handlers.go` so terminal child processes cannot orphan descendants or block shutdown through inherited stdio.
- Keep `internal/session/manager.go`, `internal/httpapi/sessions.go`, and the web stop mutation behaviorally unchanged except that the backend stop path will now complete and transition `stopping -> stopped` without manual intervention.

## Test Plan

- Add a regression in `internal/acp/client_test.go` using a wrapper helper command that mimics the observed `wrapper -> child -> native binary` shape and keeps stdio inherited. Assert that `Driver.Stop` returns within the configured timeout and no child from that process group remains alive.
- Add a regression for terminal cleanup in ACP terminal tests using the same wrapper+child pattern. Assert that `kill`, `release`, and `closeAll` terminate the full subtree.
- Add one session-level regression in `internal/session/manager_integration_test.go` that wires a real `acp.Driver` through a helper wrapper command and verifies `manager.Stop` returns and persists `stopped` instead of leaving metadata in `stopping`.
- Verify with targeted ACP and session tests, then run `make verify`.

## Assumptions

- Unix process-group semantics are acceptable for the supported runtime path on macOS/Linux. Windows keeps a direct-process fallback unless a repo-standard cross-platform process-tree abstraction already exists.
- The intended fix is “stop completes and state becomes `stopped`”, not “remove the session from the sidebar”. Hiding stopped sessions would be a separate UX change.
- The fix should cover all ACP providers launched through wrappers, especially the built-in `npx`-based providers, not just Codex.
