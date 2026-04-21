# Assistant-UI Hard Cut with `Tools()` Toolkit

## Summary

- Treat this as a hard-cut migration in one branch/series: no committed dual renderer, no compat shims, and no custom frontend chat message model left behind.
- First, sync `.compozy/tasks/assistant-ui/_techspec.md`, `_tasks.md`, and ADR-001 to the approved implementation choice: `Tools()` with backend-only tool definitions replaces `makeAssistantToolUI`; `makeAssistantDataUI` stays for `data-agh-permission`.
- Keep the Go daemon as the sole AI SDK UI Message Stream encoder; the frontend swap is runtime/rendering/state ownership, not protocol replacement.

## Implementation Changes

- Backend contract hard-cut:
- Replace `SessionTranscriptResponse.messages []transcript.Message` with AI SDK `UIMessage[]` in `internal/api/contract` and `internal/api/contract/responses.go`, then regenerate OpenAPI and generated web contract types.
- Add `internal/transcript.ToUIMessages(...)` that projects persisted events into deterministic AI SDK `UIMessage` objects with stable message/part IDs, reasoning parts, tool-call parts, and `data-agh-permission` / `data-agh-event` data parts.
- Update `SessionManager.Transcript(...)`, HTTP transcript handlers, and transcript tests to use the new edge type while keeping AGH-native replay helpers internal.
- Web runtime and state hard-cut:
- Introduce a keyed `SessionChatRuntimeProvider` in the session route using `useChatRuntime`, `AssistantChatTransport`, `AssistantRuntimeProvider`, and a `ThreadHistoryAdapter.withFormat(...)` loader backed by `/api/sessions/:id/transcript`.
- Remove the current live/transcript stitching path: `useSessionChat`, `useSessionTranscript`, transcript/live/event mappers, streaming buffer, and chat-view virtualization glue stop owning message state.
- Delete the custom frontend `UIMessage` / `TranscriptMessage` model and use canonical AI SDK types from `ai`.
- Reduce the session store to draft persistence only unless a concrete assistant-ui gap forces a narrowly scoped addition.
- Assistant UI surface:
- Vendor the assistant-ui shadcn thread stack into `web/src/components/assistant-ui/` and theme it with `DESIGN.md` and existing AGH tokens, reusing current markdown/code presentation where it still adds value.
- Build a centralized session toolkit with `useAui({ tools: Tools({ toolkit }) })`; define AGH tools as `type: "backend"` with render-only entries that wrap the existing `tool-renderers/*` content components.
- Keep `makeAssistantDataUI` for `data-agh-permission`; render `PermissionPrompt` inline from the data part and keep `data-agh-event` unrendered in-thread for inspector/observability use.
- Rebuild chat header, composer, and inspector integrations around `useAui`, `useAuiState`, thread state, and `useThreadTokenUsage` instead of Zustand-managed chat lifecycle state.
- Legacy deletion:
- Delete the old session renderer after the new route is wired: `chat-view`, `message-bubble`, `thinking-block`, `processing-indicator`, mapper/buffer modules, history/live session-store fields, and tests/stories that only exist for that model.
- Keep and adapt only the AGH-specific pieces that still matter after the swap: tool renderer bodies, permission body, inspector panels, and session actions.

## Public Interfaces and Types

- `GET /api/sessions/:id/transcript` returns AI SDK `UIMessage[]` on the wire; there is no legacy fallback transcript shape.
- Web session types stop exporting custom `UIMessage`, `TranscriptMessage`, and `TranscriptToolResult`; canonical message typing comes from `ai`, plus a typed `AghPermissionData` schema for permission data parts.
- Tool registration becomes one toolkit module registered through `Tools()` and `useAui`; no final per-component legacy tool registrars remain.

## Test Plan

- Backend:
- `internal/transcript` table tests for stable replay IDs, reasoning ordering, tool-call/result pairing, multi-step turns, permission data parts, and live/replay parity.
- HTTP handler and integration tests for the new `/transcript` contract and unchanged `/prompt` stream semantics.
- Frontend:
- Runtime-provider/history-adapter tests for initial replay, session switch remount, cancel, clear, and transcript reload.
- Toolkit renderer tests for each AGH tool status path.
- Session route integration tests for send, stream text, reasoning, tool rendering, inline permission approval, stop, clear, and inspector updates.
- Storybook refresh for the new thread surface and retained AGH-specific renderer bodies.
- Verification:
- Regenerate OpenAPI/client artifacts, run `make web-lint`, `make web-typecheck`, `make web-test`, focused Go tests for touched packages, then `make verify` as the blocking gate.

## Assumptions and Defaults

- Chosen default: adopt assistant-ui’s current `Tools()` API now, using `type: "backend"` render-only tool entries for AGH server-executed tools.
- Keep `makeAssistantDataUI` for permission data parts in this phase; do not expand scope to native AI SDK interrupt/resume yet.
- No committed feature flag or long-lived side-by-side renderer path; intermediate local work can be incremental, but the landed result is a hard cut.
