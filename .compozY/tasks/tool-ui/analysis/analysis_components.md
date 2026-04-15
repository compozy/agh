# UI Component Analysis: Tool Call Rendering — t3code Reference vs AGH

## 1. Timeline Component Architecture

### t3code: `MessagesTimeline.tsx` (1010 lines)

Three-layer architecture:

1. `MessagesTimeline` — list owner, pure orchestrator
2. `TimelineRowCtx` — shared context bypassing LegendList's memo boundaries
3. `TimelineRowContent` — dispatches to sub-components per row kind

**Row model** (`MessagesTimelineRow`):

```typescript
type MessagesTimelineRow =
  | { kind: "work"; id: string; groupedEntries: WorkLogEntry[] }
  | {
      kind: "message";
      id: string;
      message: ChatMessage;
      durationStart: string;
      showCompletionDivider: boolean;
      showAssistantCopyButton: boolean;
      assistantTurnDiffSummary?: TurnDiffSummary;
      revertTurnCount?: number;
    }
  | { kind: "proposed-plan"; id: string; proposedPlan: ProposedPlan }
  | { kind: "working"; id: string; createdAt: string | null };
```

### AGH: `chat-view.tsx`

Two-layer approach:

1. `ChatView` — owns virtualizer + scroll (TanStack Virtual)
2. `ChatMessageRow` — dispatches per row kind

**Row model** (`RowDescriptor`):

```typescript
type RowDescriptor =
  | { kind: "message"; msg: UIMessage }
  | { kind: "tool_group"; tools: UIMessage[] }
  | { kind: "processing" };
```

### Gaps in our model

- No `proposed-plan` kind
- No `showCompletionDivider` (no response divider concept)
- No `durationStart` / timing data on messages
- No `revertTurnCount` (checkpoint revert not modeled)
- No `assistantTurnDiffSummary` (no changed-files section)

---

## 2. Structural Sharing / Performance

### t3code: `computeStableMessagesTimelineRows`

```typescript
function isRowUnchanged(a: MessagesTimelineRow, b: MessagesTimelineRow): boolean {
  if (a.kind !== b.kind || a.id !== b.id) return false;
  switch (a.kind) {
    case "message":
      return (
        a.message === bm.message &&
        a.durationStart === bm.durationStart &&
        a.showCompletionDivider === bm.showCompletionDivider &&
        a.showAssistantCopyButton === bm.showAssistantCopyButton &&
        a.assistantTurnDiffSummary === bm.assistantTurnDiffSummary &&
        a.revertTurnCount === bm.revertTurnCount
      );
    // ...
  }
}
```

Prevents virtualizer from re-rendering unchanged rows during streaming.

### AGH: No structural sharing

Every `buildRows()` call creates fresh objects. Custom comparator on `ChatMessageRow`:

```typescript
(prev, next) => prev.row === next.row && prev.agentName === next.agentName;
```

This always fails because `row` is a new reference each time.

**Action**: Implement `computeStableRows` with per-field shallow comparison.

---

## 3. Work Group Section (t3code) vs Tool Group (AGH)

### t3code: `WorkGroupSection`

```tsx
<div className="rounded-xl border border-border/45 bg-card/25 px-2 py-1.5">
  {showHeader && (
    <div className="mb-1.5 flex items-center justify-between gap-2 px-0.5">
      <p className="text-[9px] uppercase tracking-[0.16em] text-muted-foreground/55">
        {groupLabel} ({groupedEntries.length})
      </p>
      {hasOverflow && <button>{isExpanded ? "Show less" : `Show ${hiddenCount} more`}</button>}
    </div>
  )}
  <div className="space-y-0.5">
    {visibleEntries.map(e => (
      <SimpleWorkEntryRow key={e.id} workEntry={e} />
    ))}
  </div>
</div>
```

- Groups ALL consecutive tool calls into one bordered card
- Shows last 6 entries when overflowed (slice from tail)
- Label adapts: "Tool calls" vs "Work log"
- `MAX_VISIBLE_WORK_LOG_ENTRIES = 6`

### AGH: tool_group rendering

```tsx
<div className="space-y-1 px-4 py-1" data-testid="tool-group">
  {cards.map(tool => (
    <ToolCallCard key={tool.id} message={tool} />
  ))}
</div>
```

No grouping container, no overflow, no count, no expand/collapse for the group.

**Action**: Wrap tool groups in bordered container with overflow logic.

---

## 4. SimpleWorkEntryRow (t3code) vs ToolCallCard (AGH)

### t3code: Compact one-liner

```tsx
<div className="rounded-lg px-1 py-1">
  <div className="flex items-center gap-2">
    <span className="flex size-5 shrink-0 items-center justify-center">
      <EntryIcon className="size-3" />
    </span>
    <p className="truncate text-xs leading-5">
      <span className="text-foreground/80">{heading}</span>
      <span className="text-muted-foreground/55"> - {preview}</span>
    </p>
  </div>
  {hasChangedFiles && (
    <div className="mt-1 flex flex-wrap gap-1 pl-6">
      {changedFiles.slice(0, 4).map(f => (
        <span className="rounded-md border bg-background/75 px-1.5 py-0.5 font-mono text-[10px]">
          {f}
        </span>
      ))}
      {count > 4 && <span>+{count - 4}</span>}
    </div>
  )}
</div>
```

### AGH: Expandable card

```tsx
<button className="group flex w-full items-center gap-2.5 rounded-lg border px-3 py-2
  border-[color:var(--color-divider)] bg-[color:var(--color-surface)]">
  <Icon className="size-3.5" />
  <span className="font-medium">{label}</span>
  <span className="truncate">{summary}</span>
  <div className="ml-auto flex items-center gap-2">
    {statusBadge}
    <ChevronRight className={cn(..., expanded && "rotate-90")} />
  </div>
</button>
{expanded && <ExpandedToolContent message={message} />}
```

### Key Differences

| Aspect         | t3code                            | AGH                                         |
| -------------- | --------------------------------- | ------------------------------------------- |
| Layout         | Flat one-liner, no border per row | Bordered card per tool                      |
| Expansion      | None (compact only)               | Per-card expand/collapse                    |
| Status         | Text color via `workToneClass`    | Explicit status badges (Running/Done/Error) |
| Preview        | Tooltip for raw command on hover  | 60-80 char truncation                       |
| Changed files  | Inline pills below row            | Only in expanded EditContent                |
| Vertical space | Very compact                      | Higher (padding + border per card)          |

---

## 5. Icon Resolution

### t3code: Priority chain

```typescript
function workEntryIcon(workEntry): LucideIcon {
  // 1. requestKind
  if (requestKind === "command") return TerminalIcon;
  if (requestKind === "file-read") return EyeIcon;
  if (requestKind === "file-change") return SquarePenIcon;
  // 2. itemType or content
  if (itemType === "command_execution" || has command) return TerminalIcon;
  if (itemType === "file_change" || has changedFiles) return SquarePenIcon;
  if (itemType === "web_search") return GlobeIcon;
  if (itemType === "image_view") return EyeIcon;
  // 3. special types
  if (itemType === "mcp_tool_call") return WrenchIcon;
  if (itemType === "dynamic_tool_call" || "collab_agent_tool_call") return HammerIcon;
  // 4. tone fallback
  return workToneIcon(tone).icon;
}
```

### AGH: Direct name lookup

```typescript
const TOOL_ICONS: Record<string, LucideIcon> = {
  Bash: Terminal,
  Read: FileText,
  Write: FileEdit,
  Edit: FileEdit,
  Grep: Search,
  Glob: FolderSearch /* ... 15 total */,
};
export function getToolIcon(toolName: string): LucideIcon {
  return TOOL_ICONS[toolName] ?? Wrench;
}
```

**Action**: Add semantic fallbacks for unknown/MCP tools.

---

## 6. Tone System

### t3code

```typescript
function workToneClass(tone): string {
  if (tone === "error") return "text-rose-300/50";
  if (tone === "tool") return "text-muted-foreground/70";
  if (tone === "thinking") return "text-muted-foreground/50";
  return "text-muted-foreground/40"; // info
}
```

### AGH: Status badges instead

```tsx
// Running: orange bg + text
// Error: red bg + text
// Done: green bg + text
```

Our badges are more scannable for in-flight tools. Keep badges but add tone-based text color as secondary signal.

---

## 7. Markdown Rendering

### t3code: Shiki + Suspense + LRU Cache

- Shiki via `@pierre/diffs`
- LRU cache: 500 entries / 50MB, keyed by `fnv1a32(code):length:language:theme`
- Cache skipped during streaming
- `Suspense` boundary per code block with fallback to `<pre>`
- Copy button on every code block
- File links → open in editor

### AGH: Prism + no caching

- PrismAsyncLight with 11 hardcoded languages
- No caching, no streaming awareness
- No Suspense boundary
- No copy button on code blocks
- All links → new tab

**Action**: Replace Prism with Shiki. Add LRU cache. Add copy button. Add streaming-aware cache bypass.

---

## 8. User Message Bubble

### t3code

```tsx
<div className="group max-w-[80%] rounded-2xl rounded-br-sm border border-border bg-secondary px-4 py-3">
  {/* Image grid (2-col) */}
  {/* Plain text (whitespace-pre-wrap) */}
  {/* Footer: copy + revert + timestamp (hover-reveal) */}
</div>
```

### AGH

```tsx
<div className="max-w-[85%] rounded-xl px-5 py-4 bg-[color:var(--color-surface-elevated)]">
  <MessageMarkdown content={message.content} />
</div>
```

Gaps: No speech-bubble corner, no border, no image support, no copy button, no timestamp, no revert.

---

## 9. Assistant Message

### t3code

- Completion divider between turns: `Response • 2.3s`
- Changed files section after each message
- Live streaming timer (`LiveMessageMeta`)
- Hover-reveal copy button
- Timestamp in `text-[10px] text-muted-foreground/30`

### AGH

- Agent label row (green dot + name)
- ThinkingBlock (collapsible)
- Markdown content
- No completion divider, no changed files, no copy button, no elapsed time

---

## 10. Working Indicator

### t3code

```tsx
<span className="inline-flex items-center gap-[3px]">
  <span className="h-1 w-1 rounded-full bg-muted-foreground/30 animate-pulse" />
  <span className="h-1 w-1 rounded-full bg-muted-foreground/30 animate-pulse [animation-delay:200ms]" />
  <span className="h-1 w-1 rounded-full bg-muted-foreground/30 animate-pulse [animation-delay:400ms]" />
</span>
<span>Working for <WorkingTimer /></span>
```

Self-ticking timer: `useState(Date.now)` + `setInterval(1000)`.

### AGH

```tsx
<Loader2 className="size-3.5 animate-spin" />
<span>Thinking...</span>
```

Static text, no elapsed timer.

**Action**: Replace with staggered dots + elapsed timer.

---

## 11. Approval Components

### t3code: Split design

- `ComposerPendingApprovalPanel`: inline text "PENDING APPROVAL · Command approval · 1/3"
- `ComposerPendingApprovalActions`: 4 buttons (Cancel turn, Decline, Always allow, Approve once)

### AGH: Card design

- `PermissionPrompt`: amber Card with tool input JSON, 4 buttons (Allow Once, Allow Always, Reject Once, Reject Always)

Our card is more informative (shows JSON input). T3code supports "Cancel turn" which we don't.

---

## 12. Components We Have That t3code Doesn't

- **`ThinkingBlock`** — collapsible thinking traces (keep)
- **`ExpandedToolContent`** — per-tool rich detail views (keep, this is a strength)
- **`data-testid` attributes** throughout (keep, critical for testing)

---

## 13. Adoption Priorities

### Critical (High Impact)

1. **Structural sharing for rows** — `computeStableRows` pattern
2. **Shiki + LRU cache** — replace Prism, add streaming awareness
3. **Copy button on code blocks** — CheckIcon/CopyIcon toggle
4. **Work group container** — bordered card with "Show N more" overflow
5. **Staggered dot animation + elapsed timer** — replace spinning loader

### High Impact

6. **User bubble shape** — `rounded-2xl rounded-br-sm` + border
7. **Hover-reveal actions** — copy button on user/assistant messages
8. **Tone-based text color** — supplement status badges
9. **Split approval components** — Panel + Actions separation
10. **Command tooltip** — show raw command on hover

### Lower Priority

11. **Completion divider** between turns
12. **Changed files section** after assistant messages
13. **DiffStatLabel** component
14. **VscodeEntryIcon** for file types

---

## 14. Color Token Translation

| t3code                     | AGH                                          |
| -------------------------- | -------------------------------------------- |
| `text-muted-foreground/30` | `text-[color:var(--color-text-tertiary)]/30` |
| `text-muted-foreground/70` | `text-[color:var(--color-text-tertiary)]`    |
| `text-foreground/80`       | `text-[color:var(--color-text-primary)]`     |
| `bg-secondary`             | `bg-[color:var(--color-surface-elevated)]`   |
| `bg-card/25`               | `bg-[color:var(--color-surface)]/25`         |
| `border-border/45`         | `border-[color:var(--color-divider)]/45`     |
| `text-success`             | `text-[color:var(--color-success)]`          |
| `text-destructive`         | `text-[color:var(--color-danger)]`           |
| `text-rose-300/50`         | `text-[color:var(--color-danger)]/50`        |
