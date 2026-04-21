# TechSpec: Adopt `assistant-ui` for the AGH Chat Surface

## Executive Summary

This TechSpec replaces the hand-rolled chat rendering stack in `web/src/systems/session` with [`assistant-ui`](https://www.assistant-ui.com) wired through the AI SDK v6 runtime (`useChatRuntime`). The AGH daemon already speaks the AI SDK UI Message Stream protocol on `POST /api/sessions/:id/prompt` (`internal/api/httpapi/prompt.go` emits `text-delta`, `reasoning-delta`, `tool-input-start`, `tool-input-available`, `tool-output-available`, `data-agh-permission`, `data-agh-event`, `finish` frames with the `x-vercel-ai-ui-message-stream: v1` header). The migration swaps the frontend renderer, not the wire protocol.

The change deletes roughly 2,500 LOC of custom message bubbles, markdown, thinking collapsibles, tool cards, streaming buffers, live/history mergers, and virtualizers. It keeps the AGH-specific surface (inspector, permission prompt body, tool-specific renderers, multi-session drafts) by plugging it into assistant-ui's composition primitives, `makeAssistantToolUI`, and `makeAssistantDataUI`. The transcript replay endpoint is unified on the AI SDK `UIMessage` shape so live and replay paths share one contract.

Greenfield discipline applies (`CLAUDE.md`, `web/CLAUDE.md`): the legacy chat renderer is deleted in the same series of PRs that introduce the new one. No dual paths, no compat shims, no `transcript.Message` frontend type carried forward.

## System Architecture

### Component Overview

`web/src/systems/session` reshapes around three boundaries:

- **Runtime provider** owns the AI SDK `useChat` instance plus assistant-ui adapters (history, attachments, speech). Implemented once per session route via `useChatRuntime` from `@assistant-ui/react-ai-sdk`.
- **Presentation layer** is a copy-paste shadcn `Thread` component under `web/src/components/assistant-ui/` that the project owns and themes with AGH design tokens. The Thread uses assistant-ui primitives (`ThreadPrimitive`, `MessagePrimitive`, `ComposerPrimitive`, `ActionBarPrimitive`).
- **AGH extensions** are registered as sibling components inside the `AssistantRuntimeProvider` tree:
  - per-tool UIs via `makeAssistantToolUI` that re-export the existing bash/read/write/edit/search/generic renderers.
  - custom data parts via `makeAssistantDataUI` for `data-agh-permission` (and any future AGH-specific parts such as `data-agh-event`, `data-agh-usage`).
  - an inspector pane that continues to live outside the thread viewport and reads from TanStack Query + `useThreadTokenUsage`.

`internal/api/httpapi` keeps a single responsibility: translate ACP `session/update` notifications into AI SDK UI Message Stream frames. It already does this; only two refinements apply — `transcript.Message` is replaced by AI SDK `UIMessage[]` on `GET /api/sessions/:id/transcript`, and `data-agh-*` names are normalized (described in Data Models).

`internal/transcript` stops being a frontend-visible shape and becomes an internal projection layer that emits AI SDK `UIMessage`s.

Data flow, live path:

1. Frontend opens the session route. `useChatRuntime({ transport: new AssistantChatTransport({ api: "/api/sessions/:id/prompt" }), adapters: { history } })` instantiates the runtime.
2. User submits a message. assistant-ui calls `chatHelpers.sendMessage` → AI SDK POSTs to `/prompt`.
3. The daemon emits UI Message Stream frames. `useChat` appends to `chat.messages`. assistant-ui renders text, reasoning, tool-call, and data parts in place.
4. Tool UIs registered via `makeAssistantToolUI` render per `toolName` with status `running | requires-action | incomplete | complete`.
5. `data-agh-permission` parts render the permission prompt via `makeAssistantDataUI`. User decision POSTs to `/api/sessions/:id/approve`. The runtime keeps streaming.
6. `finish` frame closes the turn. `useChatRuntime` transitions `isRunning=false`. `onFinish` invalidates session/history/transcript queries.

Data flow, replay path:

1. Session switch triggers the `ThreadHistoryAdapter.withFormat("ai-sdk/v6")` `load()` implementation.
2. `load` calls `GET /api/sessions/:id/transcript` and returns `{ messages: UIMessage[] }`.
3. assistant-ui rehydrates the thread. No client-side translation layer exists because the backend emits the AI SDK shape directly.

## Implementation Design

### Core Interfaces

Frontend runtime wiring (in `web/src/routes/_app/session.$id.tsx` or a slim `SessionChatRuntimeProvider`):

```ts
import { AssistantRuntimeProvider } from "@assistant-ui/react";
import { AssistantChatTransport, useChatRuntime } from "@assistant-ui/react-ai-sdk";

export function SessionChatRuntimeProvider({ sessionId, children }: Props) {
  const runtime = useChatRuntime({
    transport: new AssistantChatTransport({
      api: `/api/sessions/${sessionId}/prompt`,
    }),
    adapters: { history: createSessionHistoryAdapter(sessionId) },
    onFinish: () => invalidateSessionQueries(queryClient, sessionId),
  });
  return <AssistantRuntimeProvider runtime={runtime}>{children}</AssistantRuntimeProvider>;
}
```

Per-tool renderers become thin wrappers over the existing `tool-renderers/*-content.tsx` modules:

```ts
import { makeAssistantToolUI } from "@assistant-ui/react";
import { BashContent } from "@/systems/session/components/tool-renderers/bash-content";

export const BashToolUI = makeAssistantToolUI<BashArgs, BashResult>({
  toolName: "Bash",
  render: ({ args, status, result, isError }) => (
    <BashContent args={args} status={status.type} result={result} isError={isError} />
  ),
});
```

Permission prompt registration:

```ts
import { makeAssistantDataUI } from "@assistant-ui/react";
import { PermissionPrompt } from "@/systems/session/components/permission-prompt";

export const AghPermissionUI = makeAssistantDataUI<AghPermissionData>({
  name: "agh-permission",
  render: ({ data }) => <PermissionPrompt {...data} />,
});
```

`ThreadHistoryAdapter` (critical: `withFormat` is mandatory under `useChatRuntime`):

```ts
export function createSessionHistoryAdapter(sessionId: string): ThreadHistoryAdapter {
  return {
    load: async () => ({ headId: null, messages: [] }),
    append: async () => {},
    withFormat: (fmt) => ({
      load: async () => {
        const messages = await fetchSessionTranscript(sessionId); // already AI SDK UIMessage[]
        return { messages };
      },
      append: async () => {
        // no-op: persistence handled server-side per SSE turn
      },
    }),
  };
}
```

Backend transcript shape (replacement in `internal/api/httpapi/sessions.go`):

```go
type uiMessage struct {
    ID        string         `json:"id"`
    Role      string         `json:"role"` // "user" | "assistant" | "system"
    Parts     []uiMessagePart `json:"parts"`
    Metadata  map[string]any `json:"metadata,omitempty"`
}

type uiMessagePart struct {
    Type       string          `json:"type"` // "text" | "reasoning" | "tool-<name>" | "data-<name>"
    ID         string          `json:"id,omitempty"`
    Text       string          `json:"text,omitempty"`
    State      string          `json:"state,omitempty"` // "input-streaming" | "input-available" | "output-available" | "output-error"
    ToolCallID string          `json:"toolCallId,omitempty"`
    Input      json.RawMessage `json:"input,omitempty"`
    Output     json.RawMessage `json:"output,omitempty"`
    Data       json.RawMessage `json:"data,omitempty"`
}
```

`transcript.Message` and `transcript.ToolResult` remain internal to `internal/transcript` as intermediate projection types. A new `transcript.ToUIMessages(events []store.Event) []uiMessage` function is added. The HTTP handler no longer serializes `transcript.Message`.

### Data Models

Wire-level AI SDK v6 frames, unchanged (emitted by `internal/api/httpapi/prompt.go`):

- `start`, `text-start`, `text-delta`, `text-end`
- `reasoning-start`, `reasoning-delta`, `reasoning-end`
- `tool-input-start`, `tool-input-available`, `tool-output-available`
- `data-agh-permission` (custom)
- `data-agh-event` (fallback for currently unmapped ACP events; kept but narrowed to observability use)
- `error`, `finish`, `[DONE]`

New derived rules:

- Every SSE frame continues to flow through `promptStreamState.emit()`; no new event kinds are added in this phase.
- Transcript replay emits the same canonical parts. The replay projection in `internal/transcript` must keep stable `id` values per part so assistant-ui de-duplicates correctly across history + live streams.
- Permission payload fields (`request_id`, `turn_id`, `tool_call_id`, `action`, `resource`, `decision`) become first-class keys in the `data-agh-permission` data object; the current `agentEventPayload` shape is kept for SSE compatibility but typed on the frontend via a dedicated `AghPermissionData` schema in `web/src/systems/session/types.ts`.

Deprecated data structures:

- `UIMessage` and `UIMessageRole` custom types in `web/src/systems/session/types.ts` are deleted; the AI SDK `UIMessage<AghMessageMetadata>` type becomes canonical.
- `live-message-mapper.ts`, `streaming-buffer.ts`, `transcript-mapper.ts`, and the dual-stream state in `session-store.ts` are removed.
- `event-mapper.ts` collapses into `permission-parts.ts` (permission data schema + helpers) once data-part rendering takes over.

### API Endpoints

No new endpoints in this phase. Behavior changes only:

- `GET /api/sessions/:id/transcript` changes response shape from `transcript.Message[]` to AI SDK `UIMessage[]`. This is a hard break; consumer is the new history adapter.
- `POST /api/sessions/:id/prompt` is unchanged.
- `POST /api/sessions/:id/prompt/cancel`, `POST /resume`, `POST /clear`, `POST /approve`, `GET /events`, `GET /history`, `GET /stream` are unchanged.

A later phase may adopt AI SDK v6's native human-in-the-loop (`interrupt` / `resume` on a tool part) to replace `data-agh-permission`. It is intentionally out of scope here (see ADR-003).

## Integration Points

No external services are introduced. Internal touch points:

- `web/src/systems/session/**` — the main refactor surface.
- `web/src/components/assistant-ui/` — new directory for shadcn-installed `Thread`, `Composer`, `ToolFallback`, `Markdown`, `Reasoning` components owned by the app.
- `web/src/routes/_app/session.$id.tsx` — wraps the session pane in `SessionChatRuntimeProvider`.
- `internal/api/httpapi/sessions.go`, `internal/api/httpapi/handlers.go`, `internal/api/core/handlers.go` — `/transcript` handler shape change.
- `internal/transcript/transcript.go` — new `ToUIMessages` projection function; `Message` and `ToolResult` demoted to internal types.
- `internal/api/contract/contract.go` — `TranscriptMessage` contract replaced by the AI SDK `UIMessage` contract; OpenAPI schema regenerated.
- `openapi/agh.json` — regenerated from updated contract.
- `packages/ui` — no new primitives; existing `MessageMarkdown`, `CodeBlock`, `Collapsible` remain reused by the copied assistant-ui components.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|----------------------|-----------------|
| `web/src/systems/session/components/chat-view.tsx`, `message-bubble.tsx`, `message-markdown.tsx`, `thinking-block.tsx`, `processing-indicator.tsx`, `message-composer.tsx` | deleted | ~960 LOC of hand-rolled rendering. Low risk once assistant-ui primitives cover the same states. | Delete after new Thread reaches visual + behavioral parity. |
| `web/src/systems/session/hooks/use-session-chat.ts`, `use-chat-view-rows.ts`, `use-chat-view-content.ts` | deleted | ~490 LOC of chat state glue. Runtime now owns messages and scroll. | Delete with the chat UI. |
| `web/src/systems/session/lib/live-message-mapper.ts`, `streaming-buffer.ts`, `event-mapper.ts`, `transcript-mapper.ts` | deleted or collapsed | Mapping and buffering happens inside the runtime. `event-mapper` permission extraction moves into a data-part schema. | Replace with small `permission-parts.ts`. |
| `web/src/systems/session/stores/session-store.ts` | modified | Drops `liveMessages`, `historyMessages`, `isStreaming`, `awaitingTranscriptSync`. Keeps `pendingPermission`, `drafts`, `activeSessionId`. | Shrink state and update consumers. |
| `web/src/systems/session/components/permission-prompt.tsx` | modified | Body stays; it now receives props from `makeAssistantDataUI` instead of a Zustand-backed modal. | Rewire props. |
| `web/src/systems/session/components/session-inspector.tsx`, `tool-call-card.tsx`, `chat-header.tsx` | modified | Inspector and chat-header remain but consume `useAui` / `useThreadRuntime` state instead of the custom store. | Thin adapters over runtime state. |
| `web/src/systems/session/components/tool-renderers/*` | modified | Inputs stay, wrappers change from `UIMessage` props to `ToolCallContentPartProps`. Medium risk on argument shape. | Introduce per-tool `makeAssistantToolUI` wrappers. |
| `web/src/components/assistant-ui/**` | new | shadcn-installed Thread / Composer / Markdown / Reasoning / ToolFallback owned by the app. | Install via assistant-ui CLI, theme with AGH tokens. |
| `web/src/routes/_app/session.$id.tsx` | modified | Wraps the session pane in `AssistantRuntimeProvider`; loads transcript via history adapter. | Replace the ad-hoc render pipeline. |
| `internal/api/httpapi/sessions.go` (and `handlers.go`) | modified | `/transcript` returns AI SDK `UIMessage[]`. Hard break on the wire. | Update handler and contract types. |
| `internal/transcript/transcript.go` | modified | Add `ToUIMessages` projection; remove frontend-visible `Message`/`ToolResult` exports via contract. | Keep internally, drop from contract. |
| `internal/api/contract/contract.go`, `openapi/agh.json` | modified | Regenerate schemas. | Regenerate. |
| Storybook + Vitest suites under `web/src/systems/session/**` | modified | ~11 stories and 3 integration tests tied to deleted components. | Replace with stories/tests against the new Thread surface. |
| `package.json` (web) | modified | Adds `@assistant-ui/react`, `@assistant-ui/react-ai-sdk` via `bun add`. Removes nothing (AI SDK already present). | Install + lockfile update. |

## Testing Approach

### Unit Tests

- Permission data schema parses valid payloads and rejects malformed ones (`web/src/systems/session/lib/permission-parts.test.ts`).
- Transcript projection (`internal/transcript/transcript_test.go`) maps persisted event streams into AI SDK `UIMessage[]`, preserving: stable part IDs, tool-call ↔ tool-result pairing, reasoning-before-text ordering, and role boundaries.
- `createSessionHistoryAdapter` fetches the transcript, returns `UIMessage[]`, and reports errors upwards (`history-adapter.test.ts`).
- Each per-tool renderer wrapper renders the existing content component with correct status mapping (`tool-renderers/*-tool-ui.test.tsx`).

### Integration Tests

- New `thread.integration.test.tsx` covers: initial replay, user send, streaming assistant text, reasoning-delta rendering, tool call + tool result, permission prompt approve/deny, cancel mid-stream, session switch resets the thread.
- `session-inspector.integration.test.tsx` keeps coverage of trace/usage/memory/files tabs; rewired to runtime state.
- Backend `internal/api/httpapi/sessions_integration_test.go` asserts the new `/transcript` shape (AI SDK `UIMessage[]`), stable IDs across replay + live, and that tool-call and tool-result produce paired parts in replay.
- Daemon harness lane continues to cover the prompt SSE stream end-to-end (no changes to prompt.go semantics).

### Coverage Target

- Frontend: existing Vitest coverage gate. No regression below current baseline after migration.
- Backend: >=80% for `internal/transcript` and modified `internal/api/httpapi` files.
- All tests must pass under `make verify` and `make web-test`.

## Development Sequencing

### Build Order

1. Install `@assistant-ui/react` and `@assistant-ui/react-ai-sdk` (via `bun add`) and copy the shadcn Thread / Composer / Markdown / Reasoning / ToolFallback components into `web/src/components/assistant-ui/`, themed with AGH tokens from `DESIGN.md`. No wiring yet.
2. Change `GET /api/sessions/:id/transcript` to emit AI SDK `UIMessage[]`; add `transcript.ToUIMessages`; regenerate contract + OpenAPI; delete the `TranscriptMessage` contract type. Backend tests updated.
3. Introduce `SessionChatRuntimeProvider` with `useChatRuntime`, `AssistantChatTransport`, and `ThreadHistoryAdapter.withFormat("ai-sdk/v6")`. Render the new `Thread` inside the session route. Gate behind a local branch; the old renderer still runs alongside in dev for parity review only (no production flag).
4. Register per-tool UIs via `makeAssistantToolUI` for Bash, Read, Write, Edit, Grep/Search, and a generic fallback. Integrate inside the runtime provider.
5. Register `AghPermissionUI` via `makeAssistantDataUI` and wire the approve/deny actions to `POST /approve`. Remove the Zustand-backed pending-permission modal mount.
6. Integrate session lifecycle controls (stop/resume/clear) into the `ComposerPrimitive.Cancel` flow and the chat header. Verify `useAui`-based session switching triggers `cancelRun()` when needed.
7. Delete the old chat renderer and its hooks, stores fields, mappers, and stories. Run `make verify` + `make web-test` + `make web-lint` + `make web-typecheck`. Fix anything red.
8. Rewrite Storybook stories and integration tests against the new Thread. Add `thread.integration.test.tsx`.
9. Daemon harness lane reruns end-to-end; visual QA pass across common sessions (long replay, mid-stream cancel, tool approval, network peer session).

Each step is self-contained and land-in-one-PR sized. Steps 2 and 3 can be opened in parallel once step 1 has landed, provided integration tests do not run against the old handler shape after step 2.

### Technical Dependencies

- AI SDK v6 (`ai@^6.0.168`, `@ai-sdk/react@^3.0.170`) — already present.
- Tailwind v4 with shadcn tokens — already present (`packages/ui/src/tokens.css`, `web/src/styles.css`).
- assistant-ui requires the `"use client"` convention in copied files; irrelevant for Vite SPA but kept by the installer for portability.

## Monitoring and Observability

- Log `/transcript` projection errors (`transcript.ToUIMessages`) with `session_id` and the offending event ID.
- Keep the existing daemon logs for SSE emission; no frame renames mean existing dashboards and correlation IDs continue to work.
- No new metrics, no new dashboards in this phase.

## Technical Considerations

### Key Decisions

- Decision: adopt `assistant-ui` with `useChatRuntime` over `ExternalStoreRuntime`.
  - Rationale: the Go daemon already emits AI SDK UI Message Stream frames; `useChatRuntime` consumes that wire natively with zero translation cost.
  - Trade-offs: thread shape is opinionated; deep layout changes require copying and editing the Thread component we already own.
  - Alternatives rejected: `ExternalStoreRuntime` (redundant translation), custom renderer (status quo, 2.5K LOC to maintain).

- Decision: keep the Go daemon as the sole AI SDK wire encoder.
  - Rationale: single-binary is a core AGH constraint; existing `prompt.go` already maps ACP to AI SDK frames; LanguageModelV3 providers flatten session state.
  - Trade-offs: none relative to alternatives; Go must track AI SDK v6 frame schema.
  - Alternatives rejected: Node sidecar running `@mcpc-tech/acp-ai-provider`, self-hosted LanguageModel provider inside Go.

- Decision: render AGH-specific surfaces as AI SDK data parts consumed via `makeAssistantDataUI`.
  - Rationale: `data-agh-permission` already exists on the wire; the assistant-ui hook registers renderers without touching the runtime.
  - Trade-offs: bypasses AI SDK v6's native `tool-approval-request` chunk and the `interrupt`/`resume` flow. We accept that asymmetry until a second feature justifies migrating.
  - Alternatives rejected: migrate permission to native interrupts in the same phase (scope creep, parallel backend work).

- Decision: unify `/transcript` on AI SDK `UIMessage[]`.
  - Rationale: single replay/live shape removes `transcript-mapper.ts` and avoids drift between replay and live renderers; aligns with greenfield rules in `CLAUDE.md`.
  - Trade-offs: a hard break for the current frontend transcript consumer in the same series of PRs; internal `transcript.Message` stays as an implementation type.
  - Alternatives rejected: keep `transcript.Message` on the wire with client-side adapter (adds the very mapper we are trying to delete).

### Known Risks

- Risk: part-ID stability drifts between replay and live, causing duplicated messages when a live turn lands on top of a recent history load.
  - Mitigation: require `transcript.ToUIMessages` to compute IDs deterministically from stored event IDs; assert parity in an integration test that replays a captured turn and then streams the same turn live.

- Risk: copied shadcn Thread drifts from `DESIGN.md` tokens over time.
  - Mitigation: put theming through `packages/ui/src/tokens.css` only; forbid ad-hoc hex in the copied components; add an oxlint rule or grep check if drift becomes real.

- Risk: tool-renderer APIs (`ToolCallContentPartProps`) differ slightly from our current `UIMessage`-based props and cause runtime surprises on partial inputs.
  - Mitigation: introduce per-tool adapter layer at the wrapper boundary; cover each tool renderer with a dedicated Vitest assertion over each status (`running`, `requires-action`, `incomplete`, `complete`).

- Risk: the `ThreadHistoryAdapter.withFormat` contract changes between assistant-ui minor releases (the project has explicitly tightened this requirement in recent changesets).
  - Mitigation: pin `@assistant-ui/*` with a tilde range and add a smoke test that the provider mounts with the adapter.

- Risk: session switching leaves a dangling SSE connection or a stale `useChat` instance.
  - Mitigation: key the `SessionChatRuntimeProvider` by `sessionId` so the subtree remounts on change; rely on assistant-ui's `cancelPendingToolCallsOnSend: true` default.

## Architecture Decision Records

- [ADR-001: Adopt `assistant-ui` with AI SDK `useChatRuntime`](adrs/adr-001.md) — Replaces the hand-rolled chat renderer with assistant-ui consuming the existing AI SDK UI Message Stream.
- [ADR-002: Keep the Go Daemon as the Sole AI SDK Wire Encoder](adrs/adr-002.md) — Rejects a Node sidecar or `LanguageModelV3` provider for ACP.
- [ADR-003: Render AGH-Specific Flows as AI SDK Data Parts via `makeAssistantDataUI`](adrs/adr-003.md) — Keeps `data-agh-permission` on the wire and registers a UI handler without touching the runtime.
- [ADR-004: Unify Transcript Replay on the AI SDK `UIMessage` Shape](adrs/adr-004.md) — Deletes the frontend-visible `transcript.Message` contract and emits `UIMessage[]` from `/transcript`.
