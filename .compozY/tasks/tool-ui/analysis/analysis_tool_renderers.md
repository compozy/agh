# Tool Renderer & Icon System Analysis: t3code Reference vs AGH

## 1. t3code Approach: Compact Work Log Entries

### 1.1 Data Model: `WorkLogEntry`

```typescript
interface WorkLogEntry {
  id: string;
  label: string;
  tone: "thinking" | "tool" | "info" | "error";
  detail?: string;
  command?: string;
  rawCommand?: string;
  changedFiles?: ReadonlyArray<string>;
  toolTitle?: string;
  itemType?: ToolLifecycleItemType;
  requestKind?: "command" | "file-read" | "file-change";
}
```

### 1.2 Row Rendering: `SimpleWorkEntryRow`

Visual anatomy:

```
[icon]  Heading text - preview/command text
        ┌─────────┐ ┌──────────┐
        │ file.ts │ │ file2.ts │   ← optional changed file pills (max 4)
        └─────────┘ └──────────┘
```

- Single-line per tool, no border per row
- Preview text shows `command → detail → first changed file` (priority order)
- Tooltip shows `rawCommand` when it differs from display command
- Changed file pills: `rounded-md border bg-background/75 px-1.5 py-0.5 font-mono text-[10px]`
- Max 4 pills + `+N` overflow

### 1.3 Icon Resolution: Priority Chain

```typescript
function workEntryIcon(workEntry): LucideIcon {
  // Priority 1: requestKind (semantic)
  "command" → TerminalIcon
  "file-read" → EyeIcon
  "file-change" → SquarePenIcon

  // Priority 2: itemType or content signals
  "command_execution" || has command → TerminalIcon
  "file_change" || has changedFiles → SquarePenIcon
  "web_search" → GlobeIcon
  "image_view" → EyeIcon

  // Priority 3: special tool types
  "mcp_tool_call" → WrenchIcon
  "dynamic_tool_call" || "collab_agent_tool_call" → HammerIcon

  // Priority 4: tone fallback
  "error" → CircleAlertIcon
  "thinking" → BotIcon
  "info" → CheckIcon
  "tool" → ZapIcon
}
```

### 1.4 Tone Color System

```typescript
function workToneClass(tone): string {
  "error"    → "text-rose-300/50"
  "tool"     → "text-muted-foreground/70"
  "thinking" → "text-muted-foreground/50"
  "info"     → "text-muted-foreground/40"
}
```

### 1.5 Label Processing

```typescript
// Strip trailing "complete"/"completed"
normalizeCompactToolLabel(value) → value.replace(/\s+(?:complete|completed)\s*$/i, "").trim()

// Heading: toolTitle preferred over label
toolWorkEntryHeading(e) → capitalizePhrase(normalizeCompactToolLabel(e.toolTitle || e.label))

// Preview: command → detail → first changed file
workEntryPreview(e) → e.command || e.detail || firstChangedFile
```

### 1.6 Diff Panel (separate component)

- Uses `@pierre/diffs` library
- Turn-scoped navigation (turn chip strip)
- Stacked (unified) and split view modes
- Word wrap toggle
- Virtualized for large diffs
- CSS variable theming:
  ```css
  --diffs-bg-addition-override: color-mix(in srgb, var(--background) 92%, var(--success));
  --diffs-bg-deletion-override: color-mix(in srgb, var(--background) 92%, var(--destructive));
  ```

### 1.7 Plan Card

- Inline in timeline as `ProposedPlanCard`
- Collapse threshold: `>900 chars` or `>20 lines`
- Gradient fade: `bg-linear-to-t from-card/95`
- Actions: Copy, Download MD, Save to workspace
- `PlanSidebar`: 340px, step statuses (completed/inProgress/pending)

---

## 2. AGH Approach: Expandable Card-Based Renderers

### 2.1 Tool Card: `ToolCallCard`

Expandable card with header + collapsible content:

```tsx
<button
  className="group flex w-full items-center gap-2.5 rounded-lg border px-3 py-2
  border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
>
  <Icon className="size-3.5" />
  <span className="font-medium">{label}</span>
  <span className="truncate">{summary}</span>
  <div className="ml-auto">
    {statusBadge}
    <ChevronRight />
  </div>
</button>;
{
  expanded && <ExpandedToolContent />;
}
```

- localStorage persistence per tool call ID
- Auto-expand on result arrival (2s then collapse)
- Edit/Write default to expanded
- Three status badges: Running (orange), Done (green), Error (red)

### 2.2 Icon System: Name Lookup

```typescript
const TOOL_ICONS: Record<string, LucideIcon> = {
  Bash: Terminal,
  Read: FileText,
  Write: FileEdit,
  Edit: FileEdit,
  Grep: Search,
  Glob: FolderSearch,
  WebSearch: Globe,
  WebFetch: Globe,
  Task: Bot,
  Agent: Bot,
  Think: Lightbulb,
  TodoWrite: ListChecks,
  NotebookEdit: NotebookPen,
  EnterPlanMode: Lightbulb,
  ExitPlanMode: Map,
  AskUserQuestion: MessageCircleQuestion,
  ToolSearch: PackageSearch,
  Skill: Sparkles,
};
// fallback: Wrench
```

### 2.3 Label System: Three-Tense

```typescript
interface ToolLabels {
  active: string; // "Running command..."
  past: string; // "Ran command"
  failure: string; // "run command" → "Failed to run command"
}
```

17 tools explicitly mapped. Fallback: `"Running {toolName}..."` / `"Used {toolName}"`.

### 2.4 Compact Summary

```typescript
function getToolCompactSummary(toolName, toolInput): string | undefined {
  Bash → toolInput.command (80 chars)
  Read/Write/Edit → toolInput.file_path (60 chars)
  Grep/Glob → toolInput.pattern (60 chars)
  WebSearch → toolInput.query (60 chars)
  WebFetch → toolInput.url (60 chars)
  default → undefined
}
```

---

## 3. Tool-Specific Renderers (AGH)

### `BashContent`

```
$ command
┌──────────────────────────────┐
│ stderr (red bg, red text)    │  ← bg-red-500/5 text-red-400/80
└──────────────────────────────┘
┌──────────────────────────────┐
│ stdout (neutral)             │  ← max-h-48, 200 line truncation
└──────────────────────────────┘
[Show full output (N lines)]      ← expandable
```

- `$ command` in monospace
- Separate stderr (red) / stdout blocks
- 200-line truncation with expand button
- No syntax highlighting

### `ReadContent`

```
path/to/file.ts    42 lines
```

- Minimal: filename + line count
- Falls back to GenericContent if no file_path

### `WriteContent`

```
path/to/file.ts
┌──────────────────────────────┐
│ content preview              │  ← max-h-48, 2000 char hard truncation
└──────────────────────────────┘
```

- File path + content preview
- Hard truncation at 2000 chars (no expand option)

### `EditContent`

```
path/to/file.ts
┌──────────────────────────────┐
│ old string (red block)       │  ← bg-red-500/5 text-red-400/70
└──────────────────────────────┘
┌──────────────────────────────┐
│ new string (green block)     │  ← bg-green-500/5 text-green-400/70
└──────────────────────────────┘
```

- Raw old/new strings (no actual diff rendering)
- 1500 char hard truncation each (no expand)
- No line numbers, no token-level diff

### `SearchContent` (Grep/Glob)

```
pattern   scope: glob/path
├── FileText  path/to/result1.ts
├── FileText  path/to/result2.ts
└── FileText  path/to/result3.ts
+N more
```

- Pattern + scope
- Max 20 results
- `shortenPath()`: last 3 path segments

### `GenericContent`

- JSON input as pretty-printed
- Result: error > stdout > content (priority)
- `max-h-32` / `max-h-48` caps

---

## 4. Side-by-Side Comparison

### Visual Density

| Aspect         | t3code                         | AGH                                |
| -------------- | ------------------------------ | ---------------------------------- |
| Layout         | All tools grouped in one card  | Each tool = own card               |
| Default state  | One line per tool, compact     | Button row per tool                |
| Expansion      | Group-level (show N more)      | Individual card expand             |
| Vertical space | Very compact                   | Higher (padding + border per card) |
| Overflow       | Last 6 entries + "Show N more" | All cards always visible           |

### Icon Logic

| Aspect         | t3code                                                   | AGH                                 |
| -------------- | -------------------------------------------------------- | ----------------------------------- |
| Strategy       | Priority chain (requestKind → itemType → content → tone) | Direct name lookup                  |
| Unknown tools  | Semantic fallback via content signals                    | Wrench fallback                     |
| MCP tools      | Explicit WrenchIcon                                      | Wrench (fallback)                   |
| Dynamic tools  | HammerIcon                                               | Not handled                         |
| Error override | CircleAlertIcon overrides tool icon                      | AlertCircle replaces icon in header |

### Label System

| Aspect     | t3code                                     | AGH                            |
| ---------- | ------------------------------------------ | ------------------------------ |
| Source     | Server-streamed activity label + toolTitle | Static string table            |
| Tenses     | Single (derived from activity state)       | Three: active/past/failure     |
| Preview    | command → detail → first changed file      | getToolCompactSummary per tool |
| Truncation | CSS truncate                               | 60-80 char hard limit          |
| Tooltip    | Raw command on hover                       | None                           |

### Diff Rendering

| Aspect         | t3code                                     | AGH                                |
| -------------- | ------------------------------------------ | ---------------------------------- |
| Approach       | Dedicated side panel with @pierre/diffs    | Raw old/new strings in EditContent |
| Quality        | Full syntax-highlighted unified/split diff | Color-coded pre blocks             |
| Virtualization | Yes                                        | No                                 |
| View modes     | Stacked + Split + Word wrap                | None                               |
| Scope          | Turn-level or conversation-level           | Per edit operation                 |

### Changed Files

| Aspect      | t3code                                           | AGH                                 |
| ----------- | ------------------------------------------------ | ----------------------------------- |
| Display     | Inline pills + AssistantChangedFilesSection tree | Only in expanded EditContent header |
| Max per row | 4 pills + "+N"                                   | 1 (the file being edited)           |
| Tree view   | ChangedFilesTree with folder expand/collapse     | None                                |
| Diff stats  | +additions / -deletions                          | None                                |

---

## 5. What t3code Does Better (Adoption Recommendations)

### HIGH PRIORITY

1. **Tone/semantic layer** — Add `tone` to UIMessage. CSS class resolution for visual hierarchy.
2. **Group-based layout** — Wrap consecutive tool calls in bordered container with overflow.
3. **Tooltip on raw command** — Show full raw command on hover for Bash tools.
4. **Changed file pills** — Show up to 4 file pills on Edit/Write entries.
5. **"Show last N" overflow** — When truncating, show most recent entries (slice from tail).

### MEDIUM PRIORITY

6. **Proper diff rendering** — Replace raw old/new blocks with actual unified diff (consider @pierre/diffs or simpler library).
7. **Priority-based icon resolution** — Add semantic fallbacks for unknown/MCP/dynamic tools.
8. **Expand button on Write/Edit** — Match BashContent's expand pattern (currently hard-truncated with no option to expand).
9. **Label normalization** — Strip "completed"/"complete" suffix from server-provided labels.

### LOW PRIORITY

10. **Plan card** — ProposedPlanCard inline in timeline with collapse/gradient fade.
11. **Plan sidebar** — Step statuses with completed/inProgress/pending indicators.
12. **DiffStatLabel** — Reusable `+N / -N` component.
13. **VscodeEntryIcon** — File-type icons with onError fallback.

---

## 6. What AGH Does Better (Keep)

1. **Per-tool expanded renderers** — Rich detail views t3code doesn't have
2. **localStorage persistence** — Expand/collapse state survives reload
3. **Auto-expand + auto-collapse** — 2s auto-expand on result arrival
4. **Three-tense labels** — Natural language for running/completed/failed states
5. **Separate stderr/stdout** — BashContent distinguishes error output clearly
6. **Test coverage** — Comprehensive Vitest unit tests for all renderers

---

## 7. File Reference

### t3code

- `.resources/t3code/apps/web/src/components/chat/MessagesTimeline.tsx` — `SimpleWorkEntryRow`, `WorkGroupSection`, `workEntryIcon`, `workToneClass`, `workEntryPreview`
- `.resources/t3code/apps/web/src/components/chat/MessagesTimeline.logic.ts` — `normalizeCompactToolLabel`, `MAX_VISIBLE_WORK_LOG_ENTRIES`
- `.resources/t3code/apps/web/src/components/DiffPanel.tsx` — diff panel
- `.resources/t3code/apps/web/src/components/chat/ProposedPlanCard.tsx` — plan card
- `.resources/t3code/apps/web/src/components/PlanSidebar.tsx` — plan sidebar
- `.resources/t3code/apps/web/src/index.css` — chat-markdown, diff styles

### AGH

- `web/src/systems/session/components/tool-call-card.tsx` — main card
- `web/src/systems/session/lib/tool-labels.ts` — icons, labels, summaries
- `web/src/systems/session/components/tool-renderers/expanded-tool-content.tsx` — router
- `web/src/systems/session/components/tool-renderers/bash-content.tsx`
- `web/src/systems/session/components/tool-renderers/edit-content.tsx`
- `web/src/systems/session/components/tool-renderers/read-content.tsx`
- `web/src/systems/session/components/tool-renderers/write-content.tsx`
- `web/src/systems/session/components/tool-renderers/search-content.tsx`
- `web/src/systems/session/components/tool-renderers/generic-content.tsx`
