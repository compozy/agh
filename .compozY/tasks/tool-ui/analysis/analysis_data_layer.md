# Data Layer Analysis: Tool Call Rendering — t3code Reference vs AGH

## 1. t3code Canonical Schema (`orchestration.ts`)

### Core Entity Hierarchy

```
OrchestrationReadModel
  └── threads: OrchestrationThread[]
        ├── messages: OrchestrationMessage[]
        ├── activities: OrchestrationThreadActivity[]   ← key for tool rendering
        ├── checkpoints: OrchestrationCheckpointSummary[]
        ├── proposedPlans: OrchestrationProposedPlan[]
        └── session: OrchestrationSession | null
              └── latestTurn: OrchestrationLatestTurn | null
```

### Key Type Definitions

```typescript
// The atomic unit for tool-call activity
interface OrchestrationThreadActivity {
  id: EventId;
  tone: "info" | "tool" | "approval" | "error"; // display category
  kind: string; // e.g. "tool.updated", "tool.completed", "task.progress"
  summary: string; // human-readable label
  payload: unknown; // richly structured opaque blob
  turnId: TurnId | null;
  sequence?: number; // ordering hint from server
  createdAt: IsoDateTime;
}

// Session lifecycle
interface OrchestrationSession {
  status: "idle" | "starting" | "running" | "ready" | "interrupted" | "stopped" | "error";
  runtimeMode: "approval-required" | "auto-accept-edits" | "full-access";
  activeTurnId: TurnId | null;
}

// Turn lifecycle
interface OrchestrationLatestTurn {
  turnId: TurnId;
  state: "running" | "interrupted" | "completed" | "error";
  requestedAt: IsoDateTime;
  startedAt: IsoDateTime | null;
  completedAt: IsoDateTime | null;
  assistantMessageId: MessageId | null;
}
```

### Event Stream Architecture

Two streams:

1. **Shell stream** (`subscribeShell`): Lightweight thread shells with computed booleans like `hasPendingApprovals`
2. **Thread detail stream** (`subscribeThread`): Full `OrchestrationThreadDetailSnapshot` or incremental `OrchestrationEvent` (21 event types)

Key internal command for tool rendering:

```typescript
const ThreadActivityAppendCommand = {
  type: "thread.activity.append",
  activity: OrchestrationThreadActivity,
};
```

---

## 2. t3code Provider Runtime Types (`providerRuntime.ts`)

### Canonical Item Types (tool taxonomy)

```typescript
const TOOL_LIFECYCLE_ITEM_TYPES = [
  "command_execution",
  "file_change",
  "mcp_tool_call",
  "dynamic_tool_call",
  "collab_agent_tool_call",
  "web_search",
  "image_view",
] as const;

type CanonicalRequestType =
  | "command_execution_approval"
  | "file_read_approval"
  | "file_change_approval"
  | "apply_patch_approval"
  | "exec_command_approval"
  | "tool_user_input"
  | "dynamic_tool_call"
  | "auth_tokens_refresh"
  | "unknown";
```

### Item Lifecycle Payload

```typescript
interface ItemLifecyclePayload {
  itemType: CanonicalItemType;
  status?: "inProgress" | "completed" | "failed" | "declined";
  title?: string;
  detail?: string;
  data?: unknown;
}
```

### Tool Rendering Event Chain

```
item.started   { itemType, status: "inProgress", title }
  ↓
content.delta  { streamKind: "command_output" | "file_change_output", delta }
  ↓
tool.progress  { toolUseId, toolName, summary, elapsedSeconds }
  ↓
item.updated   { itemType, status, title, detail, data }
  ↓
item.completed { itemType, status: "completed"|"failed", title, detail, data }
```

---

## 3. t3code Business Logic (`session-logic.ts`)

### Output Types

```typescript
interface WorkLogEntry {
  id: string;
  createdAt: string;
  label: string;
  detail?: string;
  command?: string;
  rawCommand?: string;
  changedFiles?: ReadonlyArray<string>;
  tone: "thinking" | "tool" | "info" | "error";
  toolTitle?: string;
  itemType?: ToolLifecycleItemType;
  requestKind?: "command" | "file-read" | "file-change";
}

type TimelineEntry =
  | { kind: "message"; message: ChatMessage }
  | { kind: "proposed-plan"; proposedPlan: ProposedPlan }
  | { kind: "work"; entry: WorkLogEntry };
```

### `deriveWorkLogEntries` Pipeline

1. **Sort** by `sequence → createdAt → lifecycleRank → id`
2. **Filter to current turn** (`activity.turnId === latestTurnId`)
3. **Filter noise**: `tool.started`, `task.started`, `context-window.updated`, checkpoints, plan boundaries
4. **Map** via `toDerivedWorkLogEntry`
5. **Collapse** consecutive entries with same `collapseKey`
6. **Strip** internal fields

### Collapse Logic

```typescript
function shouldCollapseToolLifecycleEntries(previous, next): boolean {
  // Both must be tool lifecycle events (tool.updated or tool.completed)
  // Don't collapse if previous is already completed
  // Must share same collapseKey
  return previous.collapseKey === next.collapseKey;
}

function deriveToolLifecycleCollapseKey(entry): string | undefined {
  const normalizedLabel = label.replace(/\s+(?:complete|completed)\s*$/i, "").trim();
  return [itemType, normalizedLabel, detail.trim()].join("\u001f");
}
```

### Command Extraction

```typescript
// extractToolCommand tries multiple candidate paths:
// payload.data.item.command → payload.data.item.input.command →
// payload.data.item.result.command → payload.data.command → detail text

// Then unwraps shell wrappers:
// bash -c "..." → inner command
// sh -lc "..." → inner command
// pwsh -Command "..." → inner command
// cmd /c "..." → inner command
```

### Changed Files Extraction

Recursively searches `payload.data` for: `path`, `filePath`, `relativePath`, `filename`, `newPath`, `oldPath`. Recurses into `item`, `result`, `input`, `data`, `changes`, `files`, `edits`, `patch`, `patches`, `operations`. Deduplicates and caps at 12.

### Approval State Machine

```typescript
function derivePendingApprovals(activities): PendingApproval[];
// Maintains Map<requestId, PendingApproval>
// "approval.requested" → add to map
// "approval.resolved" → remove from map
// Handles stale request cleanup

// requestKind mapping:
// command_execution_approval | exec_command_approval → "command"
// file_read_approval → "file-read"
// file_change_approval | apply_patch_approval → "file-change"
```

---

## 4. AGH Current Data Types

### `UIMessage` (web/src/systems/session/types.ts)

```typescript
interface UIMessage {
  id: string;
  role: "user" | "assistant" | "tool_call" | "tool_result" | "system";
  content: string;
  toolName?: string;
  toolInput?: Record<string, unknown>;
  toolResult?: ToolUseResult;
  toolError?: boolean;
  thinking?: string;
  thinkingComplete?: boolean;
  isStreaming?: boolean;
  timestamp: number; // unix ms (t3code uses ISO string)
}

interface ToolUseResult {
  stdout?: string;
  stderr?: string;
  filePath?: string;
  content?: string;
  structuredPatch?: unknown[];
  error?: string;
  rawOutput?: unknown;
}
```

### Backend Contract (`internal/api/contract/contract.go`)

```go
type SessionEventPayload struct {
    ID, SessionID string
    Sequence      int64
    TurnID, Type  string
    Content       json.RawMessage // opaque blob
    Timestamp     time.Time
}
```

### Transcript (`internal/transcript/transcript.go`)

```go
type Message struct {
    ID               string
    Role             Role // "user" | "assistant" | "tool_call" | "tool_result"
    Content, Thinking string
    ToolName         string
    ToolInput        json.RawMessage
    ToolResult       *ToolResult
    ToolError        bool
    Timestamp        time.Time
}

type ToolResult struct {
    Stdout, Stderr, FilePath, Content, Error string
    StructuredPatch, RawOutput json.RawMessage
}
```

---

## 5. Gap Analysis

### Missing in AGH

| Concept                  | t3code                                          | AGH                                            |
| ------------------------ | ----------------------------------------------- | ---------------------------------------------- |
| Activity taxonomy        | typed `kind`, `tone`, `summary`, `payload`      | free-form `type` string                        |
| Render tone              | `"thinking" \| "tool" \| "info" \| "error"`     | none                                           |
| Lifecycle stages         | `tool.started → tool.updated → tool.completed`  | single `type` field                            |
| `WorkLogEntry`           | separate rendered unit for tool activity        | tools are `UIMessage` with `role: "tool_call"` |
| Collapse/dedup           | consecutive tool events merged by `collapseKey` | none                                           |
| Command normalization    | shell wrapper unwrapping                        | none                                           |
| Changed files extraction | recursive payload traversal, capped at 12       | none                                           |
| Multi-approval tracking  | `Map<requestId, PendingApproval>`               | single `pendingPermission` slot                |
| `CanonicalItemType`      | 7-item tool taxonomy                            | none                                           |
| Timeline merging         | messages + work entries + plans interleaved     | flat `UIMessage[]`                             |

### AGH Has That t3code Doesn't

- **Token usage as first-class data** — `TokenUsagePayload` richly exposed at API boundary
- **Multi-format transcript assembler** — handles canonical/legacy/loose event formats
- **Separate `tool_call` / `tool_result` messages** — enables rich per-tool expanded renderers

### Data Flow Comparison

**t3code:**

```
Provider → ProviderRuntimeEventV2 → OrchestrationThreadActivity
  → deriveWorkLogEntries() (filter, sort, map, collapse)
  → WorkLogEntry[]
  → deriveTimelineEntries() (merge with messages and plans)
  → TimelineEntry[]
```

**AGH:**

```
Provider (ACP) → AgentEvent → SessionEvent (stored)
  → transcript.Assemble() OR mapAgentEventToUIMessage()
  → UIMessage[] (flat, tool_call/tool_result as roles)
  → rendered directly
```

### Sorting Differences

- t3code: 4-level sort: `sequence → createdAt → lifecycleRank → id`
- AGH: `Sequence → Timestamp → ID` (no lifecycle rank concept)

---

## 6. Key Implementation Gaps for Tool UI Improvement

1. **`WorkLogEntry` type** — separate from `UIMessage`, with `tone`, `itemType`, `requestKind`, `command`, `rawCommand`, `changedFiles`, `toolTitle`
2. **Activity taxonomy** — add structured `kind`/`tone` to events or derive client-side
3. **`CanonicalItemType` classification** — map tool event types to 7-item taxonomy
4. **`deriveWorkLogEntries`** — sort, filter, map, collapse pipeline
5. **Collapse logic** — merge `tool.updated → tool.completed` pairs by collapseKey
6. **Command normalization** — shell wrapper detection and unwrapping
7. **`extractChangedFiles`** — recursive payload traversal for file paths
8. **Stateful approval tracker** — replace single-slot with `Map<requestId, PendingApproval>`
9. **`deriveTimelineEntries`** — merge sorted messages + work entries
10. **Turn lifecycle tracking** — for divider/completion UI
