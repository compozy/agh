# Fix Prompt Stream Stall On First Tool Call

## Summary
- Make session prompts survive client disconnects; the browser stream may end, but the server-side prompt must continue until terminal completion or explicit cancel.
- Bring the HTTP prompt stream into AI SDK v6 UI-message protocol compliance so `useChat` does not abort on the first tool event.
- Add explicit prompt cancellation to preserve stop semantics after detaching prompt execution from the HTTP request.
- Stop `Status()` from crash-classifying sessions that are still in the `pending` startup window.

## Public Interfaces / Behavior Changes
- Add `CancelPrompt(ctx, sessionID)` to the session-manager surface and expose `POST /api/sessions/:id/prompt/cancel` on both HTTP and UDS.
- Keep the existing `POST /api/sessions/:id/prompt` route, but change its HTTP wire format to AI SDK-compliant UI-message SSE parts driven by the JSON `type` field, not AGH-specific SSE event names.
- Keep UDS prompt streaming behavior AGH-native, but apply the same detached lifetime and explicit cancel semantics there.

## Implementation Changes
- In HTTP and UDS prompt handlers, call `Sessions.Prompt` with `context.WithoutCancel(c.Request.Context())`; keep the writer loop itself bound to the request context so disconnect stops streaming only, not prompt execution.
- Implement `CancelPrompt` in `session.Manager` by resolving the active process and delegating to `driver.Cancel`; make it idempotent when no prompt is active so stop-button races do not surface as user-facing failures.
- Change the web chat stop path to call the new cancel endpoint instead of `chat.stop()`, letting the original streaming request stay open until the backend emits a terminal event.
- Update the HTTP prompt stream emitter to send AI SDK tool parts as a complete sequence:
- `tool-input-start`
- `tool-input-available` with normalized tool input from the ACP tool payload
- `tool-output-available`
- Keep `data-agh-permission` as a custom data part; keep `data-agh-event` only as additive debug metadata, never as a required signal for tool rendering or liveness.
- Treat sessions in `m.pending` as still starting inside `Status()`/`readMeta()` and skip `repairInactiveMeta()` until the session is neither active nor pending.
- Preserve the existing repair behavior for genuinely stale on-disk `starting` metadata belonging to inactive sessions.

## Test Plan
- HTTP prompt stream integration: a prompt that produces assistant text plus a tool call must stream `start`, text/reasoning parts, `tool-input-start`, `tool-input-available`, `tool-output-available`, `finish`, and `[DONE]`, with no premature request completion.
- HTTP disconnect regression: abort the client after the first tool call and assert the server still records `tool_result` plus terminal `done`/`error` in session history.
- Prompt cancel regression: call `POST /api/sessions/:id/prompt/cancel` during an active prompt, assert the ACP cancel notification is sent once, and assert the prompt ends cleanly.
- Web chat regression: `useSessionChat.stop()` must cancel via API without leaving the UI stuck in `Running...`.
- Session status regression: polling a session during the `pending` startup window must not rewrite metadata to `"start did not complete"`, while a truly stale inactive `starting` session must still be repaired to `stopped`.

## Assumptions
- There is at most one active prompt per session, so prompt cancellation remains session-scoped and does not require a turn ID.
- The chat surface’s compatibility target is AI SDK v6 data-stream protocol; AGH-specific custom stream parts remain supplementary.
- Fixing the HTTP framing and decoupling prompt lifetime from request cancellation is the root-cause correction; no sleeps, retries, or client-side reconnect hacks should be introduced.
