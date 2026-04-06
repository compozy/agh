# Fix Multi-Agent Session Bootstrap, CLI Guidance, and Live Dashboard Rendering

## Summary

- Align role prompts with actual runtime capabilities so supervisor, advisor, and researcher can use AGH correctly and predictably.
- Correct the bootstrap contract: `root` already exists, `supervisor` and `advisor` already start in `root`, and child workgroups must not receive workers until a master is spawned and the workgroup is active.
- Enforce AGH orchestration at the driver layer by disabling Claude's native `Agent(...)` delegation tool.
- Add a live session-events stream and make the Svelte dashboard keep the current canvas and terminals mounted during background refreshes, so new sessions appear without `Ctrl+R` and TUIs stay visible.

## Key Changes

- Prompt and tool alignment:
  - Extend the prompt tool matrix so `advisor` and `researcher` gain `bash` specifically for AGH CLI control-plane calls; keep their existing read/grep/list tools unchanged.
  - Update the supervisor, advisor, and researcher templates plus shared prompt context to include exact supported command forms, not generic command names.
  - Remove or rewrite any prompt or kickoff text that tells the supervisor to create the first workgroup under `root`.
  - Encode the workgroup lifecycle rule in prompts and examples: create child workgroup, spawn its master into that child, wait until the workgroup is active, then spawn workers, reviewers, or researchers into that child.
  - Encode the advisor reply contract explicitly: when consulted, the advisor must answer through `agh send supervisor "<answer>"` or the caller agent id from context, then update status or mark done.
- Runtime enforcement:
  - Update the Claude driver launch command to pass `--disallowedTools Agent` so Claude cannot open native child agents outside AGH orchestration.
  - Add a focused regression test around the Claude command builder to lock that flag in place.
- Dashboard live data and rendering:
  - Add a lightweight dashboard session stream as a dedicated websocket route for global session invalidation events.
  - Keep `/api/sessions` as the canonical snapshot endpoint; the websocket only signals invalidation and the frontend answers by calling `refresh()` immediately in the background.
  - Change the Svelte session store state from one `loading` flag to `initialLoading` plus `refreshing`.
  - Preserve the current session list and `selectedName` during background refreshes, and do not auto-switch to a newly created session.
  - Change the app gating so the blocking "waiting for topology" state only renders when there is no prior session or topology data yet.
  - Keep terminal websockets attached across session-list refreshes.
- Observability and debugability:
  - Add one end-to-end bootstrap regression that asserts the real event sequence for a fresh session: `session_started`, advisor consultation delivered, advisor reply observed, child workgroup created only when requested, master spawned into the child before any child worker, and no native `Agent(...)` tool usage appears in the transcript.
  - Reuse the existing observability APIs and transcript capture as the verification source of truth.

## Public Interfaces and Types

- Add a global dashboard websocket route dedicated to session invalidation events.
- Update `SessionsState` to expose `initialLoading` and `refreshing`.
- Update the prompt capability contract so `advisor` and `researcher` include AGH-capable `bash`, and make the shared CLI guidance authoritative for exact command syntax and orchestration rules.
- Update the Claude driver contract so native agent delegation is always disabled.

## Test Plan

- Prompt and unit tests:
  - Verify assembled master, advisor, and researcher prompts contain the exact supported AGH command forms and the corrected bootstrap and workgroup rules.
  - Verify the kickoff message no longer instructs creating the first workgroup under `root`.
- Driver tests:
  - Verify Claude command construction includes `--disallowedTools Agent`.
- Kernel and dashboard integration tests:
  - Verify session invalidation events are emitted for session start, resume, stop, and session summary count changes.
  - Verify a fresh session can consult the advisor and the advisor reply is visible to the supervisor through AGH message and status flows.
  - Verify child workgroups are not populated out of order in the expected orchestration path.
- Frontend tests:
  - Verify session background refresh keeps prior items and selection intact while `refreshing` is true.
  - Verify a live session invalidation triggers immediate refresh without full page reload.
  - Verify the app does not replace the mounted canvas with the blocking empty state during background refresh once data exists.
  - Verify terminal components remain mounted and PTY connections are not churned by session-list refreshes.
- Manual validation:
  - Start a new session from the UI or CLI and confirm the new session appears in the sidebar without `Ctrl+R`.
  - Open the session and confirm supervisor and advisor terminals render immediately, advisor replies through AGH, and the observability event and transcript views show the expected sequence.

## Assumptions and Defaults

- Keep the currently selected session stable unless it disappears.
- Use AGH-only orchestration for control-plane roles and do not rely on Claude native sub-agents.
- The existing observability spine remains the canonical debug source.
- No prompt-only workaround is acceptable for native delegation or missing reply paths; runtime and prompt contracts must agree.
