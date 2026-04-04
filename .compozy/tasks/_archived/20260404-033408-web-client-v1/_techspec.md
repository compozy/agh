# TechSpec: AGH Web Client V1

## Executive Summary

The AGH Web Client V1 is a React 19 SPA that provides a visual chat interface for interacting with AI agents managed by the AGH daemon. The implementation uses a hybrid architecture: Vercel AI SDK (`useChat`) as the SSE transport layer for consuming the daemon's streaming API, with a custom `SimpleStreamingBuffer` (adapted from `.resources/harnss`) and Zustand store for fine-grained rendering control at 60fps. TanStack Query handles server state (agents, sessions lists), while TanStack Router provides URL-based session navigation.

**Primary trade-off:** We accept coupling to the Vercel AI SDK stream format in exchange for battle-tested SSE transport, reconnection, and error handling — a safe bet since the daemon was explicitly designed to output `x-vercel-ai-ui-message-stream: v1`.

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Browser (localhost:3000)                                       │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  TanStack Router                                         │   │
│  │  __root.tsx → _app.tsx (layout) → session.$id.tsx        │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌──────────┐  ┌─────────────────────────┐  ┌──────────────┐   │
│  │ systems/ │  │ systems/session/         │  │ systems/     │   │
│  │ agent/   │  │                          │  │ daemon/      │   │
│  │          │  │  useChat (AI SDK)        │  │              │   │
│  │ Sidebar: │  │       ↓                  │  │ Health       │   │
│  │ agent    │  │  SimpleStreamingBuffer   │  │ polling      │   │
│  │ list,    │  │       ↓                  │  │              │   │
│  │ session  │  │  Zustand sessionStore    │  │ Connection   │   │
│  │ list     │  │       ↓                  │  │ indicator    │   │
│  │          │  │  ChatView (virtualized)  │  │              │   │
│  │ TanStack │  │  ToolCards, Permissions  │  │ TanStack     │   │
│  │ Query    │  │  Composer                │  │ Query        │   │
│  └──────────┘  └─────────────────────────┘  └──────────────┘   │
│                                                                 │
│  ── Vite proxy: /api → localhost:2123 ──────────────────────    │
└─────────────────────────────────────────────────────────────────┘
                              │
                    HTTP/SSE REST API
                              │
                 ┌────────────┴────────────┐
                 │  AGH Daemon (:2123)     │
                 │  internal/httpapi/       │
                 │  Sessions, Agents,      │
                 │  Observe, Prompt SSE    │
                 └─────────────────────────┘
```

### Data Flow

1. **Agent/Session lists**: TanStack Query → `GET /api/agents` / `GET /api/sessions` → cache → sidebar components
2. **Session creation**: Mutation → `POST /api/sessions` → invalidates sessions query → router navigates to `/session/:id`
3. **Prompt streaming**: `useChat` → `POST /api/sessions/:id/prompt` (SSE) → custom stream callbacks → `SimpleStreamingBuffer` → rAF flush → Zustand `sessionStore` → React renders
4. **Session history**: On navigation to `/session/:id` → TanStack Query fetches `GET /api/sessions/:id/history` → transforms `TurnHistoryPayload[]` to `UIMessage[]` → Zustand store
5. **Permission flow**: SSE emits `data-agh-permission` → Zustand `pendingPermission` → PermissionPrompt component → `POST /api/sessions/:id/approve` → daemon resolves

## Implementation Design

### Core Interfaces

#### UIMessage — Unified Message Model

```typescript
// systems/session/types.ts
// Adapted from harnss UIMessage, scoped to daemon event types

interface UIMessage {
  id: string
  role: "user" | "assistant" | "tool_call" | "tool_result" | "system"
  content: string
  toolName?: string
  toolInput?: Record<string, unknown>
  toolResult?: ToolUseResult
  toolError?: boolean
  thinking?: string
  thinkingComplete?: boolean
  isStreaming?: boolean
  timestamp: number
}

interface ToolUseResult {
  stdout?: string
  stderr?: string
  filePath?: string
  content?: string
  structuredPatch?: unknown[]
  error?: string
}
```

#### SessionStore — Zustand Store

```typescript
// systems/session/stores/session-store.ts

interface SessionState {
  activeSessionId: string | null
  messages: UIMessage[]
  isStreaming: boolean
  pendingPermission: PermissionRequest | null
  setActiveSession: (id: string, messages: UIMessage[]) => void
  appendMessage: (msg: UIMessage) => void
  updateLastMessage: (partial: Partial<UIMessage>) => void
  setPendingPermission: (req: PermissionRequest | null) => void
  clearSession: () => void
}
```

#### AgentEventPayload — Daemon SSE Event

```typescript
// systems/session/types.ts
// Mirrors internal/httpapi/prompt.go agentEventPayload

interface AgentEventPayload {
  type: string
  session_id?: string
  turn_id?: string
  timestamp?: string
  text?: string
  title?: string
  tool_call_id?: string
  stop_reason?: string
  action?: string
  resource?: string
  decision?: string
  error?: string
  usage?: TokenUsagePayload
  raw?: unknown
}

interface PermissionRequest {
  requestId: string
  toolName: string
  toolInput: Record<string, unknown>
  action: string
  resource: string
}
```

### Data Models

#### API Response Types

```typescript
// systems/agent/types.ts

interface AgentPayload {
  name: string
  provider: string
  command?: string
  model?: string
  tools?: string[]
  permissions?: string
  prompt?: string
}

// systems/session/types.ts

interface SessionPayload {
  id: string
  name: string
  agent_name: string
  workspace: string
  state: "starting" | "active" | "stopping" | "stopped"
  acp_session_id?: string
  acp_caps?: ACPCaps
  created_at: string
  updated_at: string
}

interface SessionEventPayload {
  id: string
  session_id: string
  sequence: number
  turn_id: string
  type: string
  agent_name: string
  content: unknown
  timestamp: string
}
```

### API Endpoints (Consumed)

The web client consumes these existing daemon endpoints (no backend changes needed):

| Method | Path | Description | System |
|--------|------|-------------|--------|
| `GET` | `/api/agents` | List all configured agents | agent |
| `GET` | `/api/agents/:name` | Get single agent details | agent |
| `GET` | `/api/sessions` | List all sessions | session |
| `POST` | `/api/sessions` | Create new session `{agent_name, name?, workspace?}` | session |
| `GET` | `/api/sessions/:id` | Get session status | session |
| `POST` | `/api/sessions/:id/prompt` | Send message, returns SSE stream | session |
| `GET` | `/api/sessions/:id/events` | Query session events (pagination) | session |
| `GET` | `/api/sessions/:id/history` | Get session turn history | session |
| `DELETE` | `/api/sessions/:id` | Stop session | session |
| `POST` | `/api/sessions/:id/resume` | Resume stopped session | session |
| `GET` | `/api/observe/health` | Daemon health status | daemon |

### SSE Stream Protocol

The prompt endpoint (`POST /api/sessions/:id/prompt`) returns an SSE stream with header `x-vercel-ai-ui-message-stream: v1`. Event sequence for a typical turn:

```
data: {"type":"start","messageId":"msg-turn-1"}

data: {"type":"reasoning-start","id":"msg-turn-1-reasoning"}
event: thought
data: {"type":"reasoning-delta","id":"msg-turn-1-reasoning","delta":"Let me think..."}
data: {"type":"reasoning-end","id":"msg-turn-1-reasoning"}

data: {"type":"text-start","id":"msg-turn-1-text"}
event: agent_message
data: {"type":"text-delta","id":"msg-turn-1-text","delta":"I'll read the file first."}
data: {"type":"text-end","id":"msg-turn-1-text"}

event: tool_call
data: {"type":"tool-input-start","toolCallId":"tc-1","toolName":"Read"}
event: tool_call
data: {"type":"data-agh-event","data":{...agentEventPayload},"toolCallId":"tc-1"}
event: tool_result
data: {"type":"tool-output-available","toolCallId":"tc-1","output":{...agentEventPayload}}

event: permission
data: {"type":"data-agh-permission","data":{...agentEventPayload}}

event: done
data: {"type":"finish","stopReason":"end_turn"}
data: [DONE]
```

### File Structure

```
web/src/
├── routes/
│   ├── __root.tsx              # Providers shell (Query, Tooltip, Toaster)
│   ├── _app.tsx                # Layout route: Sidebar + main area
│   ├── _app/
│   │   ├── index.tsx           # Empty state (no session selected)
│   │   └── session.$id.tsx     # Chat view for session
│   └── index.tsx               # Redirect to /_app or connection check
├── systems/
│   ├── agent/                  # Agent domain module
│   │   ├── index.ts            # Public barrel
│   │   ├── types.ts            # AgentPayload
│   │   ├── adapters/
│   │   │   └── agent-api.ts    # GET /api/agents, GET /api/agents/:name
│   │   ├── lib/
│   │   │   ├── query-keys.ts   # ['agents']
│   │   │   └── query-options.ts
│   │   ├── hooks/
│   │   │   └── use-agents.ts   # useAgents(), useAgent(name)
│   │   └── components/
│   │       ├── agent-sidebar-group.tsx   # Collapsible agent + sessions
│   │       └── agent-icon.tsx            # Provider-based icon
│   │
│   ├── session/                # Session domain module (largest)
│   │   ├── index.ts            # Public barrel
│   │   ├── types.ts            # UIMessage, SessionPayload, PermissionRequest, ToolUseResult
│   │   ├── adapters/
│   │   │   └── session-api.ts  # All session CRUD + event history endpoints
│   │   ├── lib/
│   │   │   ├── query-keys.ts   # ['sessions'], ['session', id, 'history']
│   │   │   ├── query-options.ts
│   │   │   ├── streaming-buffer.ts    # SimpleStreamingBuffer (from harnss)
│   │   │   ├── event-mapper.ts        # SSE AgentEventPayload → UIMessage transforms
│   │   │   └── tool-labels.ts         # Tool name → icon, label (active/past/failure)
│   │   ├── hooks/
│   │   │   ├── use-sessions.ts        # useSessions(), useSession(id)
│   │   │   ├── use-session-chat.ts    # useChat wrapper + streaming buffer + rAF flush
│   │   │   ├── use-session-history.ts # Load event history → UIMessage[]
│   │   │   └── use-session-actions.ts # createSession, stopSession, resumeSession mutations
│   │   ├── stores/
│   │   │   └── session-store.ts       # Zustand: messages, streaming state, permissions
│   │   └── components/
│   │       ├── chat-view.tsx           # Virtualized message list
│   │       ├── chat-header.tsx         # Session info bar
│   │       ├── message-bubble.tsx      # User/assistant message rendering
│   │       ├── message-composer.tsx    # Input + send button
│   │       ├── thinking-block.tsx      # Collapsible reasoning display
│   │       ├── tool-call-card.tsx      # Collapsible tool card (from harnss)
│   │       ├── tool-renderers/
│   │       │   ├── expanded-tool-content.tsx  # Router to specific renderers
│   │       │   ├── bash-content.tsx
│   │       │   ├── read-content.tsx
│   │       │   ├── write-content.tsx
│   │       │   ├── edit-content.tsx
│   │       │   ├── search-content.tsx
│   │       │   └── generic-content.tsx
│   │       ├── permission-prompt.tsx   # Allow/reject UI
│   │       ├── session-sidebar-item.tsx # Session item in sidebar
│   │       └── processing-indicator.tsx # Streaming animation
│   │
│   └── daemon/                 # Daemon status module
│       ├── index.ts            # Public barrel
│       ├── types.ts            # Health payload
│       ├── adapters/
│       │   └── daemon-api.ts   # GET /api/observe/health
│       ├── lib/
│       │   ├── query-keys.ts
│       │   └── query-options.ts
│       ├── hooks/
│       │   └── use-daemon-health.ts  # Health polling + connection status
│       └── components/
│           └── connection-status.tsx  # Connected/disconnected indicator
│
├── components/
│   └── ui/                     # shadcn components (existing, 57 components)
├── hooks/
│   └── use-mobile.ts           # Existing responsive hook
├── integrations/
│   └── tanstack-query/
│       └── root-provider.tsx   # Existing QueryClient setup
├── lib/
│   └── utils.ts                # Existing cn() helper
├── styles.css                  # Existing OKLCH theme
└── main.tsx                    # Existing app entry
```

### Key Implementation Patterns

#### Convention: Zustand Store Access in Components

Per `web/CLAUDE.md`, components should be presentational and receive data via props. The Zustand `sessionStore` is a scoped exception: streaming-hot components (`chat-view`, `message-bubble`) may use `useSessionStore` selectors directly for performance (avoiding prop-drilling through virtualized lists). Route-level components (`session.$id.tsx`) orchestrate store initialization and pass non-streaming data as props.

#### 1. Streaming Buffer + rAF Flush

Adapted from `.resources/harnss/src/lib/streaming-buffer.ts`:

```typescript
// systems/session/lib/streaming-buffer.ts

export class SimpleStreamingBuffer {
  messageId: string | null = null
  private textChunks: string[] = []
  private thinkingChunks: string[] = []
  thinkingComplete = false

  appendText(text: string): void {
    this.textChunks.push(text)
  }

  appendThinking(text: string): void {
    const current = this.thinkingChunks.join("")
    this.thinkingChunks = [mergeStreamingChunk(current, text)]
  }

  getText(): string { return this.textChunks.join("") }
  getThinking(): string { return this.thinkingChunks.join("") }

  reset(): void {
    this.messageId = null
    this.textChunks = []
    this.thinkingChunks = []
    this.thinkingComplete = false
  }
}
```

Usage in `use-session-chat.ts`:

```typescript
const bufferRef = useRef(new SimpleStreamingBuffer())
const rafRef = useRef<number | null>(null)

const scheduleFlush = () => {
  if (rafRef.current === null) {
    rafRef.current = requestAnimationFrame(() => {
      const buf = bufferRef.current
      sessionStore.updateLastMessage({
        content: buf.getText(),
        thinking: buf.getThinking(),
        thinkingComplete: buf.thinkingComplete,
      })
      rafRef.current = null
    })
  }
}
```

#### 2. Event Mapping (SSE → UIMessage)

```typescript
// systems/session/lib/event-mapper.ts

export function mapAgentEventToUIMessage(
  event: AgentEventPayload,
  existingId?: string,
): Partial<UIMessage> {
  switch (event.type) {
    case "agent_message":
    case "thought":
      // Handled by streaming buffer, not direct mapping
      return {}
    case "tool_call":
      return {
        id: event.tool_call_id ?? existingId,
        role: "tool_call",
        toolName: event.title,
        toolInput: event.raw as Record<string, unknown>,
        timestamp: Date.now(),
      }
    case "tool_result":
      return {
        id: event.tool_call_id,
        role: "tool_result",
        toolResult: parseToolResult(event),
        timestamp: Date.now(),
      }
    case "permission":
      return {} // Handled via Zustand pendingPermission
    default:
      return {}
  }
}
```

#### 3. ChatView Virtualization

Uses `@tanstack/react-virtual` (need to add as dependency):

```typescript
// systems/session/components/chat-view.tsx

type RowDescriptor =
  | { kind: "message"; msg: UIMessage }
  | { kind: "tool_group"; tools: UIMessage[] }
  | { kind: "processing" }

// Pure function to build rows from messages
function buildRows(messages: UIMessage[], isStreaming: boolean): RowDescriptor[] {
  // Groups consecutive tool_call/tool_result into tool_group rows
  // Adds processing indicator when streaming
}

// Virtualizer setup
const virtualizer = useVirtualizer({
  count: rows.length,
  getScrollElement: () => scrollRef.current,
  estimateSize: (i) => estimateRowHeight(rows[i]),
  overscan: 10,
})
```

#### 4. Tool Card Pattern

Adapted from `.resources/harnss/src/components/ToolCall.tsx`:

```typescript
// systems/session/components/tool-call-card.tsx

// Collapsible card with:
// - Trigger: icon + tool name + compact summary + chevron
// - Content: ExpandedToolContent router (Bash, Read, Write, Edit, etc.)
// - Auto-expand on result arrival, auto-collapse after 2s
// - Persisted expand state via localStorage
// - Three status modes: executing (shimmer), success (past tense), error (red)
```

#### 5. Permission Prompt Pattern

```typescript
// systems/session/components/permission-prompt.tsx

// Inline card in chat area when pendingPermission is set
// Shows: tool name, action, resource
// Buttons: Allow Once, Allow Always, Reject Once, Reject Always
// POST /api/sessions/:id/approve with permission response
// Clears Zustand pendingPermission on response
```

#### 6. Sidebar Agent → Sessions Hierarchy

```typescript
// systems/agent/components/agent-sidebar-group.tsx

// Uses shadcn SidebarGroup + SidebarMenu + SidebarMenuSub
// Agent item: collapsible with provider icon + name
// Session items: SidebarMenuSubButton with title, state badge, permission indicator
// "New Session" button per agent group
```

### Routing Structure

```
URL Pattern              Route File                    Description
/                        routes/index.tsx               Redirect to /_app (or show connection check)
/session/:id             routes/_app/session.$id.tsx    Chat view for specific session
(no session selected)    routes/_app/index.tsx           Empty state / welcome
```

The `_app.tsx` layout route renders the sidebar + main content area. It wraps all child routes with the sidebar layout so the sidebar is always visible.

```typescript
// routes/_app.tsx
function AppLayout() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <AppHeader />
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  )
}
```

### Markdown Rendering

For rendering agent markdown responses (code blocks, headings, lists, links):

- Use `react-markdown` with `remark-gfm` (GitHub-flavored markdown)
- Code blocks with syntax highlighting via `shiki` or `prism-react-renderer`
- Memoize rendered markdown to avoid re-parsing during streaming (only re-render when content changes)

### Theme Support

The OKLCH design token system already supports light/dark via `.dark` CSS class. Use `next-themes` (already in package.json) for:
- System preference detection
- Manual toggle (light/dark/system)
- Persistence via localStorage
- `useTheme()` hook for components that need conditional rendering

## Integration Points

### AGH Daemon HTTP API

- **Connection**: Vite dev proxy (`/api → localhost:2123`). In production, configure reverse proxy or same-origin serving.
- **Authentication**: None (daemon is local-first, no auth required).
- **Error handling**: HTTP status codes from daemon map to UI states:
  - `404` → Session not found (navigate away, show toast)
  - `409` → Max sessions reached (show error in creation dialog)
  - `500` → Internal error (show toast, retry option)
- **Retry strategy**: TanStack Query default retry (3 retries with exponential backoff) for GET endpoints. No retry for POST mutations (user-initiated).

### SSE Stream Reconnection

- AI SDK `useChat` handles reconnection internally
- For the event stream (`GET /api/sessions/:id/stream`), use `Last-Event-ID` header for resume. The daemon supports sequence-based resumption.
- If connection is lost mid-prompt-stream, the prompt stream ends. User can re-send the message. (The daemon records events, so history is not lost.)

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `web/src/routes/` | new | New route files for app layout and session view | Create `_app.tsx`, `_app/index.tsx`, `_app/session.$id.tsx` |
| `web/src/systems/agent/` | new | Agent domain module with API adapter and sidebar components | Create full system directory |
| `web/src/systems/session/` | new | Session domain module — largest system, handles chat, streaming, tools, permissions | Create full system directory |
| `web/src/systems/daemon/` | new | Daemon health module with connection indicator | Create full system directory |
| `web/src/routes/__root.tsx` | modified | Add theme provider wrapper | Low risk — additive change |
| `web/src/routes/index.tsx` | modified | Replace placeholder with redirect to /_app | Low risk |
| `web/package.json` | modified | Add dependencies: @ai-sdk/react, ai, @tanstack/react-virtual, react-markdown, remark-gfm | Low risk |
| Backend (`internal/httpapi/`) | modified | `POST /api/sessions/:id/approve` needs implementation (currently 501 stub). Requires adding pending permission storage to ACP, blocking handler, and approval routing through session stack. | Implement interactive permission approval (task_07) |

## Testing Approach

### Unit Tests

- **StreamingBuffer**: Test `appendText`, `appendThinking`, `mergeStreamingChunk` with overlap detection edge cases
- **event-mapper**: Test every SSE event type → UIMessage transform, including unknown event types (graceful degradation)
- **buildRows**: Test row descriptor generation from message arrays (tool grouping, processing indicator)
- **tool-labels**: Test tool name → label mapping for all known tools + unknown fallback
- **Zustand store**: Test state transitions (setActiveSession, appendMessage, updateLastMessage, setPendingPermission)

### Integration Tests

- **API adapters**: Test against mock HTTP server (msw) — verify request format, response parsing, error handling for each endpoint
- **useChat integration**: Test SSE stream consumption with mock SSE server emitting the exact event sequence from the daemon's prompt handler
- **Session navigation**: Test loading history events → UIMessage[] reconstruction on session switch

### Manual Testing Protocol

- Full loop: create session → send message → see streaming response → tool cards render → approve permission → navigate between sessions
- Cross-browser: Chrome, Firefox, Safari
- Responsive: Test sidebar collapse on mobile viewport (< 768px)
- Dark/light mode: Test theme toggle and persistence
- Connection loss: Kill daemon mid-stream → verify reconnection and error state

## Development Sequencing

### Build Order

1. **Foundation: Types + API Adapters** — no dependencies
   - Create `systems/session/types.ts`, `systems/agent/types.ts`, `systems/daemon/types.ts`
   - Create all API adapters (`agent-api.ts`, `session-api.ts`, `daemon-api.ts`)
   - Create TanStack Query keys and options for all endpoints
   - Add new dependencies to `package.json` (`@ai-sdk/react`, `ai`, `@tanstack/react-virtual`, `react-markdown`, `remark-gfm`)

2. **Routing + Layout Shell** — depends on step 1
   - Create `routes/_app.tsx` with sidebar provider + inset layout
   - Create `routes/_app/index.tsx` (empty state)
   - Create `routes/_app/session.$id.tsx` (placeholder)
   - Update `routes/index.tsx` to redirect to `/_app`
   - Update `routes/__root.tsx` to add theme provider

3. **Daemon System: Connection Status** — depends on step 1
   - Create `systems/daemon/` with health polling hook
   - Create `connection-status.tsx` component
   - Wire into app header

4. **Agent System: Sidebar** — depends on steps 1, 2
   - Create `systems/agent/` with `useAgents()` hook
   - Create `agent-sidebar-group.tsx` and `agent-icon.tsx`
   - Wire agent list into sidebar layout

5. **Session System: Session List + CRUD** — depends on steps 1, 2, 4
   - Create `systems/session/` with `useSessions()`, `useSessionActions()` hooks
   - Create `session-sidebar-item.tsx` with state badges
   - Wire session list under each agent in sidebar
   - Implement create/stop/resume mutations

6. **Session System: Streaming Buffer + Chat Hook** — depends on steps 1, 5
   - Create `streaming-buffer.ts` (adapted from harnss)
   - Create `event-mapper.ts` for SSE → UIMessage transforms
   - Create `session-store.ts` (Zustand)
   - Create `use-session-chat.ts` (useChat wrapper + buffer + rAF flush)

7. **Session System: Chat View** — depends on step 6
   - Create `chat-view.tsx` with `@tanstack/react-virtual`
   - Create `message-bubble.tsx` with markdown rendering
   - Create `thinking-block.tsx`
   - Create `message-composer.tsx` (input + send)
   - Create `chat-header.tsx` (session info)
   - Create `processing-indicator.tsx`
   - Wire into `routes/_app/session.$id.tsx`

8. **Session System: Tool Cards** — depends on step 7
   - Create `tool-call-card.tsx` (collapsible card pattern from harnss)
   - Create `tool-renderers/expanded-tool-content.tsx` (router)
   - Create individual renderers: `bash-content.tsx`, `read-content.tsx`, `write-content.tsx`, `edit-content.tsx`, `search-content.tsx`, `generic-content.tsx`
   - Create `tool-labels.ts` for tool name → icon/label mapping

9. **Session System: Permissions** — depends on steps 7, 8
   - Create `permission-prompt.tsx` component
   - Wire pending permission state from Zustand store
   - Add permission indicator to `session-sidebar-item.tsx` (pulsing amber dot)
   - Implement approve/reject POST to daemon

10. **Session History + Navigation** — depends on steps 6, 7
    - Create `use-session-history.ts` to fetch and transform session events to UIMessage[]
    - Implement session switch: save current messages, load history for target session
    - Handle SSE reconnection on session switch (connect to active session's stream)

### Technical Dependencies

- **NPM packages to add**: `@ai-sdk/react`, `ai`, `@tanstack/react-virtual`, `react-markdown`, `remark-gfm`, `react-syntax-highlighter`, `@types/react-syntax-highlighter`
- **Daemon running**: Required for integration testing. Use `agh daemon start` in dev.
- **Backend change required**: `POST /api/sessions/:id/approve` must be implemented (currently 501 stub) — see task_07
- **All other endpoints** already exist in `internal/httpapi/`

## Monitoring and Observability

### Browser Console Logging

- SSE events logged at `debug` level (filterable via browser devtools)
- API errors logged at `warn` level with request context
- Connection state changes logged at `info` level

### TanStack Query Devtools

- Already configured in `web/vite.config.ts` via `@tanstack/devtools-vite`
- Shows all query states, cache, refetch timing in development

### Performance Metrics

- Track streaming frame rate (rAF callback timing) — warn if flush takes > 16ms
- Track virtualized row count vs DOM node count (should be ~20 DOM nodes regardless of message count)

## Technical Considerations

### Key Decisions

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| AI SDK for SSE transport | Daemon outputs Vercel-compatible format; proven reconnection | Coupled to AI SDK stream format |
| SimpleStreamingBuffer + rAF | 60fps rendering regardless of event rate; proven in harnss | Slight complexity vs direct setState |
| Zustand for UI state | Selector-based subscriptions prevent re-render cascade during streaming | Two state systems to understand |
| TanStack Virtual for chat | Handles 1000+ messages with ~20 DOM nodes | Requires height estimation + measurement |
| 3 systems (agent, session, daemon) | Clear domain boundaries; session system is largest and most complex | Slightly more files than monolithic approach |

### Known Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| AI SDK `useChat` may not expose all custom event types (permission, data-agh-event) | Medium | High | Use stream part callbacks or `fetch` option to intercept raw stream. Fallback: custom SSE parser for prompt endpoint only |
| Permission approve endpoint may not exist yet in daemon | Medium | High | Verify `POST /api/sessions/:id/approve` exists. If not, add it as a prerequisite task |
| `react-markdown` + shiki bundle size impact | Low | Medium | Use dynamic import for shiki; consider lighter alternatives like `marked` + manual code highlighting |
| Session history → UIMessage reconstruction may lose streaming fidelity | Low | Low | Events store full content — reconstruction is text-only (no streaming animation), which is acceptable for history |

## Architecture Decision Records

- [ADR-001: Harness Web Lite](adrs/adr-001.md) — SPA web focada no loop de conversa, copiando patterns do harnss, sem features Electron-specific
- [ADR-002: AI SDK + Harnss Hybrid Streaming](adrs/adr-002.md) — useChat como transport SSE + SimpleStreamingBuffer + rAF flush para rendering 60fps
- [ADR-003: Zustand + TanStack Query State Management](adrs/adr-003.md) — Zustand para UI/streaming state, TanStack Query para server state cache
