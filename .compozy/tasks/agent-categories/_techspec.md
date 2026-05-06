# TechSpec — AGENT.md Category Path With Tree Sidebar And Command Agent Picker

## Executive Summary

Adds an optional, display-only `category_path: ["Marketing", "Sales"]` array to `AGENT.md` frontmatter so agents can be organized hierarchically without affecting runtime behavior, ACP execution, scheduling, autonomy, or permissions. The field flows verbatim through the existing parse → validate → resource codec → daemon sync → contract → CLI/HTTP/UDS → web UI pipeline as a flat string array on every agent payload; the web client builds the tree/group presentation purely client-side. The web UI swaps the flat sidebar agent list for a `@headless-tree/react`-backed `AgentCategoryTree` built on `packages/ui/src/components/reui/tree.tsx`, and replaces every native agent `<select>` with two new shared components (`AgentCommandSelect`, `AgentCommandMultiSelect`) built on `packages/ui/src/components/command.tsx`.

The primary trade-off: we introduce **one** new optional metadata field across every agent surface (Go struct, parsed YAML, resource codec, daemon project, bundle activation payload, OpenAPI contract, generated TypeScript, CLI rows, web payload) instead of any backend tree endpoint, denormalized "categories" join table, schema migration, or `config.toml` key. This costs surface coverage breadth and forces a single PR to ship every consumer in lockstep, but it preserves the runtime moat (no behavior branches on category), keeps ACP/scheduler/permission code untouched, and lets the web absorb 100% of the presentation logic. Greenfield-alpha posture is enforced: the canonical form is the array `category_path` only — no `categories` alias, no slash-string fallback (`"Marketing/Sales"`), no compatibility shim, no migration code.

## System Architecture

### Component Overview

The change crosses six existing components plus three new web components. None are added to runtime execution paths.

- **`internal/config` (parser & validator)** — owns `AgentDef`, `parsedAgentDef`, `ParseAgentDef`, `AgentDef.Validate`, `EditAgentDefFile`, `CloneAgentDef`, `validateAgentResourceSpec`. Becomes the single source of truth for the canonical form, normalization rules, and validation errors. All other components consume the already-normalized field.
- **`internal/workspace`** — `cloneAgentDefs` in `clone.go` currently rebuilds `AgentDef` field-by-field by hand and already drops `Skills` (a latent bug). Replaced with `aghconfig.CloneAgentDef`, fixing the existing skills-clone gap and preventing `category_path` drift in the same change.
- **`internal/api/contract` + `internal/api/core`** — `AgentPayload`, `BundleAgentPayload`, `AgentPayloadFromDef`, `AgentPayloadFromDiagnostic`, and the bundle activation projector gain a single `category_path,omitempty []string` field. `AgentPayloadFromDiagnostic` keeps `category_path` nil because diagnostics represent malformed files where the field cannot be trusted.
- **`internal/daemon` native tools surface** — `workspace_describe` and any other agent-shaped projection consume `AgentPayload` already; they get `category_path` for free once the contract is updated. Verified end-to-end via the existing transport-parity tests.
- **`internal/cli` (`agh agent list`, `agh agent info`, workspace agent views)** — JSON output exposes `category_path` (no aliases). Human and toon output add a single compact `Category` column/key rendered as `Marketing / Sales` (a single-space-delimited path) for root-level agents the cell is empty.
- **`@agh/ui` kit (`packages/ui`)** — exports `Tree`, `TreeItem`, `TreeItemLabel`, `TreeDragLine` from `packages/ui/src/index.ts` so `agh-web` consumes them via the public surface. `command.tsx` exports are already public.
- **`agh-web` (new web modules)** —
  - `AgentCategoryTree`: replaces the flat `AgentList` map inside `AppSidebar`.
  - `AgentCommandSelect` (single) and `AgentCommandMultiSelect` (multi): replace native agent `<select>` controls in the session-create dialog, the settings skills agent scope picker, and the network create-channel dialog.
  - A private shared `AgentCommandList` renders `Command`/`CommandInput`/`CommandList`/`CommandGroup`/`CommandItem`/`CommandEmpty` so the two consumer components share keyboard, grouping, and empty-state behavior.
  - A new `agent-category` library inside the agent system owns category utilities (build tree nodes, sort, ID derivation, label formatting).

### Data Flow

```
AGENT.md (frontmatter)
  → ParseAgentDef → AgentDef.CategoryPath (normalized)
  → AgentDef.Validate (rejects invalid segments)
  → workspace resolver / resource codec / daemon project
  → contract.AgentPayload.CategoryPath  (flat string[])
     ├── HTTP /api/agents and /api/workspaces/:id  (web client builds tree)
     ├── UDS  agh agent list / info / workspace describe
     └── Native tool agh__workspace_describe
```

The web tree/group is purely presentational: same flat payload, two distinct UI projections (sidebar tree + command-picker grouped list).

## Implementation Design

### Core Interfaces

#### `internal/config/agent.go` — canonical field and normalization

```go
type AgentDef struct {
    // ... existing fields ...
    CategoryPath []string `yaml:"category_path,omitempty" toml:"category_path,omitempty" json:"category_path,omitempty"`
    // ... existing fields ...
}

type parsedAgentDef struct {
    // ... existing fields ...
    CategoryPath []string `yaml:"category_path,omitempty" toml:"category_path,omitempty"`
    // ... existing fields ...
}

// ParseAgentDef:
agent.CategoryPath = normalizeAgentCategoryPath(parsed.CategoryPath)
```

```go
// normalizeAgentCategoryPath trims each segment and returns nil for an empty result.
// It does NOT lowercase, reorder, or deduplicate; casing and order are author intent.
func normalizeAgentCategoryPath(in []string) []string {
    if len(in) == 0 {
        return nil
    }
    out := make([]string, 0, len(in))
    for _, raw := range in {
        out = append(out, strings.TrimSpace(raw))
    }
    return out
}
```

```go
// validateAgentCategoryPath rejects segments the file system / UI cannot safely render.
// Called from AgentDef.Validate after normalization.
func validateAgentCategoryPath(path []string) error {
    for i, seg := range path {
        switch {
        case seg == "":
            return fmt.Errorf("agent.category_path[%d]: blank segment", i)
        case seg == "." || seg == "..":
            return fmt.Errorf("agent.category_path[%d]: %q is not a valid segment", i, seg)
        case strings.ContainsAny(seg, `/\`):
            return fmt.Errorf("agent.category_path[%d]: %q must not contain '/' or '\\'", i, seg)
        }
    }
    return nil
}
```

`AgentDef.Validate` calls `validateAgentCategoryPath(a.CategoryPath)` after the existing tool/permission checks. `validateAgentResourceSpec` and `EditAgentDefFile` route through `normalizeAgentCategoryPath` then `Validate`, so the same rules apply uniformly.

#### `internal/config/agent_clone.go` — single clone authority

```go
func CloneAgentDef(agent AgentDef) AgentDef {
    return AgentDef{
        // ... existing fields ...
        Skills:       normalizeAgentSkillsConfig(agent.Skills),
        CategoryPath: append([]string(nil), agent.CategoryPath...),
        // ... existing fields ...
    }
}
```

#### `internal/workspace/clone.go` — delete the hand-rolled clone

Replace `cloneAgentDefs` body with a single call:

```go
func cloneAgentDefs(src []aghconfig.AgentDef) []aghconfig.AgentDef {
    if len(src) == 0 {
        return nil
    }
    cloned := make([]aghconfig.AgentDef, 0, len(src))
    for _, agent := range src {
        cloned = append(cloned, aghconfig.CloneAgentDef(agent))
    }
    return cloned
}
```

This deletes the hand-rolled field-by-field copy that already silently dropped `Skills`. Treat it as a delete target (see Key Decisions).

#### `internal/api/contract/contract.go` and `bundles.go`

```go
type AgentPayload struct {
    // ... existing fields ...
    CategoryPath []string `json:"category_path,omitempty"`
    // ... existing fields ...
}

type BundleAgentPayload struct {
    // ... existing fields ...
    CategoryPath []string `json:"category_path,omitempty"`
    // ... existing fields ...
}
```

#### `internal/api/core/conversions.go`

```go
func AgentPayloadFromDef(agent aghconfig.AgentDef) contract.AgentPayload {
    // ... existing field copies ...
    return contract.AgentPayload{
        // ... existing fields ...
        CategoryPath: append([]string(nil), agent.CategoryPath...),
        // ... existing fields ...
    }
}
```

`AgentPayloadFromDiagnostic` does NOT set `CategoryPath` — diagnostic placeholder rows leave the field nil because the source AGENT.md is malformed and the parsed value cannot be trusted.

#### Web — agent category utility (new `web/src/systems/agents/lib/agent-category.ts`)

```ts
export type AgentCategoryNode =
  | { kind: "folder"; id: string; label: string; segments: string[]; children: AgentCategoryNode[] }
  | { kind: "leaf"; id: string; label: string; agent: AgentPayload };

// buildAgentCategoryTree: groups agents by AgentPayload.category_path.
// Folders are sorted before leaves; siblings are sorted case-insensitively by visible label.
// Folder IDs derive from the joined path ("category:Marketing/Sales"); leaf IDs derive from agent name ("agent:coder").
// Agents with no category_path become root-level leaves (no synthetic "Uncategorized" folder).
export function buildAgentCategoryTree(agents: AgentPayload[]): AgentCategoryNode[];

// formatCategoryLabel(["Marketing", "Sales"]) === "Marketing / Sales"
export function formatCategoryLabel(path: string[] | null | undefined): string;
```

#### Web — `AgentCategoryTree` (new `web/src/components/agent-category-tree.tsx`)

Wraps `useTree` from `@headless-tree/react` with `syncDataLoaderFeature` + `selectionFeature`. Renders folder nodes with `TreeItem`/`TreeItemLabel`, and leaf nodes via the `TreeItem` render hook so each leaf is a TanStack `Link` to `/agents/$name`. Preserves existing test IDs (`agent-row-${agent.name}`, `agent-active-${agent.name}`, `agent-status-dot-${agent.name}`) and adds new deterministic test IDs for folders (`agent-category-${joinedPath}`).

Default expansion: ancestors of the active agent expand on first render; otherwise top-level categories are expanded. Tree expansion state is local UI state only (no persistence, no config key).

#### Web — `AgentCommandSelect` / `AgentCommandMultiSelect`

Both compose a private `AgentCommandList` that renders `Command`, `CommandInput`, `CommandList`, `CommandGroup` (heading = formatted category label, root-level agents under `"Agents"`), `CommandItem`, `CommandEmpty`, and `CommandShortcut`. They differ only in selection semantics:

```ts
interface AgentCommandSelectProps {
  agents: AgentPayload[];
  value: string | null;
  onChange: (next: string | null) => void;
  triggerTestId?: string; // reuses existing testIds (e.g. "session-agent-select")
  disabled?: boolean;
  placeholder?: string;
}

interface AgentCommandMultiSelectProps {
  agents: AgentPayload[];
  value: string[];
  onToggle: (next: string[]) => void;
  triggerTestId?: string; // reuses "settings-agent-select" etc.
}
```

Single closes its popover on selection and shows the selected agent's name + provider + formatted category label inside the trigger. Multi keeps the popover open, marks each item with `data-checked={selected}`, and surfaces a selected count + per-item provider/category metadata.

### Data Models

| Surface | Type | Field | Shape |
|---|---|---|---|
| `internal/config.AgentDef` | Go struct | `CategoryPath` | `[]string` (yaml/toml/json `category_path,omitempty`) |
| `internal/config.parsedAgentDef` | Go struct | `CategoryPath` | `[]string` |
| `contract.AgentPayload` | API contract | `CategoryPath` | `[]string` (`json:"category_path,omitempty"`) |
| `contract.BundleAgentPayload` | API contract | `CategoryPath` | `[]string` (`json:"category_path,omitempty"`) |
| Generated `web/src/generated/agh-openapi.d.ts` | TS | `category_path?: string[]` | array, optional |
| Web `AgentCategoryNode` | Discriminated union | `kind` | `"folder" \| "leaf"` |

No new database columns, indexes, tables, or migrations are introduced. AGH SQLite (`agh.db`, `events.db`, catalog DBs) is untouched.

### API Endpoints

No new endpoints. The following existing endpoints gain `category_path?: string[]` on their agent payloads:

| Method | Path | Source | Notes |
|---|---|---|---|
| GET | `/api/agents` | `core.HandlerListAgents` | `AgentResponse.agents[].category_path` |
| GET | `/api/agents/:name` | `core.HandlerGetAgent` | `AgentResponse.agent.category_path` |
| GET | `/api/workspaces/:id` | workspace detail handler | Each `agents[].category_path` |
| GET | `/api/bundles/.../activations` | bundle activation projector | Each `agents[].category_path` |
| (UDS) | `agent.list`, `agent.info`, `workspace.describe` | UDS handlers | Same payload over UDS |
| (Native tool) | `agh__workspace_describe` | `daemon/native_tools.go` | Inherits the contract |

OpenAPI regeneration via `make codegen` propagates the field into `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`. `make codegen-check` MUST pass.

## Integration Points

No external services. The change extends two third-party UI primitives already in the repo (`@headless-tree/core`, `cmdk`) by adding the missing peer:

- **`@headless-tree/react@^1.6.3`** — added to `agh-web` via `bun add @headless-tree/react@^1.6.3 --filter agh-web` to match the existing `@headless-tree/core@^1.6.3`. No version drift between core and react peers is allowed.
- **No new icon, motion, or animation packages.** `motion` and `lucide-react` already exist in `agh-web`; the tree and command components rely on simple CSS transitions only.

## Impact Analysis

### Code Surfaces

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|----------------------|-----------------|
| `internal/config/agent.go` | modified | New `CategoryPath` field in `AgentDef` + `parsedAgentDef`, normalization helper, validation rule. Risk: medium — every parse path runs through this. | Add normalization + validation + tests for valid/invalid segments and root-level nil. |
| `internal/config/agent_edit.go` | modified | `EditAgentDefFile` must round-trip `CategoryPath` through `parsedAgentDef` so on-disk frontmatter survives skill enable/disable mutations. Risk: medium — write path silently drops fields when forgotten. | Mirror parsed → agent → parsed copy for `CategoryPath`; add a test that toggles `Skills.Disabled` and confirms `category_path` is preserved on disk. |
| `internal/config/agent_clone.go` | modified | `CloneAgentDef` is the single deep-copy authority for any consumer that needs to mutate. | Add `CategoryPath` to clone with defensive copy. |
| `internal/config/agent_resource.go` | modified | `validateAgentResourceSpec` normalizes and re-validates spec before storage. | Apply same normalization + validation. |
| `internal/workspace/clone.go` | **delete + replace** | Hand-rolled `cloneAgentDefs` already drops `Skills`; rewriting it would also need to track every new `AgentDef` field forever. | **Delete** the field-by-field clone body and call `aghconfig.CloneAgentDef`. (Listed as a delete target.) |
| `internal/api/contract/{contract,bundles}.go` | modified | Adds `CategoryPath` to `AgentPayload` and `BundleAgentPayload`. | Mark `omitempty`; never inline JSON aliases. |
| `internal/api/core/conversions.go` | modified | `AgentPayloadFromDef` copies the field; `AgentPayloadFromDiagnostic` keeps it nil. | Defensive copy; document the diagnostic exclusion in a comment. |
| `internal/daemon/*` (resource sync, bundle activation) | modified | Bundle materialization/projection and resource sync inherit the field. | Verified via transport-parity + bundle activation tests. |
| `internal/cli/agent_commands.go` | modified | Human, toon, and JSON output for `agent list`, `agent info`, and workspace agent views show category. | Add a Category column to human/toon output; JSON exposes the array verbatim. |
| `internal/testutil/e2e/config_seed.go` | modified | Adds optional `CategoryPath` to `AgentSeed` for fixtures. | Backward-compatible default = nil; any e2e fixture can now seed a categorized agent. |
| `web/src/generated/agh-openapi.d.ts` | regenerated | OpenAPI codegen output. | `make codegen` then `make codegen-check`. |
| `web/e2e/fixtures/runtime-seed.ts` | modified | `BrowserMockAgentSeed` gets optional `category_path: string[]`. | Mock builder writes `category_path` into the served payload. |
| `packages/ui/src/index.ts` | modified | Re-exports `Tree`, `TreeItem`, `TreeItemLabel`, `TreeDragLine`. | Public surface for `agh-web`. |
| `agh-web` `app-sidebar.tsx` | refactored | Replace `AgentList` with `AgentCategoryTree`. Risk: high (test ID & route preservation). | Preserve existing test IDs and active route handling. Delete `AgentList`. |
| `agh-web` session-create dialog | refactored | Replace native `<select>` with `AgentCommandSelect`. | Preserve `session-agent-select` test ID on the new trigger. |
| `agh-web` settings skills agent scope | refactored | Replace native `<select>` with `AgentCommandSelect`. | Preserve `settings-agent-select` test ID. |
| `agh-web` network create-channel dialog | refactored | Replace custom button list with `AgentCommandMultiSelect`. | Preserve `network-agent-option-${agent.name}` on each item. |

### Extensibility, Agent-Manageability, and Config Lifecycle

- **Extensibility surfaces (per CLAUDE.md "extensible by the runtime").** No new extension hook or registry is required. The field is part of the `AgentDef` resource shape so it automatically flows through:
  - Resource codec (`NewAgentResourceCodec`) — extensions that author agent resources via the resource API may set the field; validation runs the same `validateAgentCategoryPath` rule.
  - Bundle materialization/projection — `BundleAgentPayload.category_path` makes the field part of the bundle activation contract, so external bundles can declare categories without touching AGH source.
  - Skill/agent registry — unchanged. Categories never affect skill resolution, precedence, or agent lookup.
  - Bridge SDK / extension manifest — unchanged. Categories are display metadata, not capability.
- **Agent-manageability (per CLAUDE.md "managed by agents").** Agents must be able to discover and reason about categories without the web UI:
  - CLI `agh agent list -o json` and `agh agent info -o json` expose `category_path` verbatim.
  - HTTP/UDS parity: same `AgentPayload.category_path` over both transports.
  - Native tool `agh__workspace_describe` returns the field through the existing `AgentPayload` projection so onboard agents can introspect categories.
  - No agent-only mutation API is added — categories are authored by humans editing `AGENT.md`. `agh agent edit` (via `EditAgentDefFile`) round-trips the field on disk, so any agent that already has write access to AGENT.md can update it.
- **Config lifecycle (per CLAUDE.md "config.toml keys/defaults/docs").** **No `config.toml` key is added.** Justification: `category_path` is per-agent metadata stored in the agent's own `AGENT.md`, not a runtime/global behavior toggle. There is nothing to enable, disable, default, or version. Adding a config key would invent a global flag for an opt-in field that is already per-agent. Sidebar tree expansion is intentionally local UI state; no persisted preference.

### Web/Docs Impact

- **Web (`web/`):**
  - `src/components/app-sidebar.tsx` — `AgentList` deleted, `AgentCategoryTree` added.
  - `src/components/app-sidebar.test.tsx` — rewritten to assert tree behavior, active row, active session dot, default expansion, keyboard navigation, and preserved test IDs.
  - `src/components/stories/app-sidebar.stories.tsx` — adds a story with categorized agents so Storybook visibly demonstrates the tree.
  - `src/systems/agents/lib/agent-category.ts` (new) + `agent-category.test.ts` (new).
  - `src/systems/agents/components/agent-command-select.tsx` (new) + tests.
  - `src/systems/agents/components/agent-command-multi-select.tsx` (new) + tests.
  - `src/systems/agents/components/agent-command-list.tsx` (new, private) + colocated tests of the shared list.
  - `src/systems/sessions/...session-create-dialog.tsx` — switches to `AgentCommandSelect`.
  - `src/routes/.../settings*` — settings skills agent scope switches to `AgentCommandSelect`.
  - `src/systems/network/components/network-create-channel-dialog.tsx` — switches to `AgentCommandMultiSelect`; preserves `network-agent-option-${agent.name}` test IDs.
  - All four touched dialogs/views require updated tests that drive the picker via `userEvent.keyboard` rather than `selectOptions` (native `<select>` is gone).
- **Docs / `packages/site`:**
  - `packages/site/content/docs/agents/...` — the AGENT.md frontmatter reference page gains a `category_path` row with example, validation rules, and the explicit "display only, no runtime impact" caveat.
  - `packages/site/content/docs/cli/agent.mdx` (or equivalent generated doc) — regenerate via `make cli-docs` so the human help shows the new column.
  - If `cd packages/site && bun run source:generate` reports drift after adding the doc page, regenerate.
  - No marketing surface change is required; this is operator-facing.

### Delete Targets

The following existing code/behavior MUST be removed in this change (greenfield-alpha posture):

1. The hand-rolled `cloneAgentDefs` body in `internal/workspace/clone.go` (currently silently drops `Skills`). Replaced by a delegation to `aghconfig.CloneAgentDef`.
2. The flat `AgentList` map in `web/src/components/app-sidebar.tsx`. Replaced by `AgentCategoryTree`.
3. The native `<select>` elements (and their `selectOptions` test interactions) used for agent selection in:
   - session-create dialog,
   - settings skills agent scope picker,
   - network create-channel dialog.
4. Any inline agent-grouping helpers in network channel dialog. Categories are now derived from `AgentPayload.category_path`.

Delete targets explicitly NOT in scope (rejected aliases that must not be introduced or revived):

- No `categories` alias (singular/plural) on AgentDef, parsedAgentDef, or any payload.
- No slash-string fallback (`"Marketing/Sales"`) accepted by parsing or validation.
- No `Uncategorized` synthetic folder in the web tree.
- No `category_path` config.toml key.
- No backend tree/group endpoint or denormalized category table.

## Testing Approach

### Unit Tests

#### `internal/config` (Go)

- `TestParseAgentDef_ShouldParseCategoryPath` — valid `category_path: [Marketing, Sales]` → `AgentDef.CategoryPath` equals `["Marketing", "Sales"]`, casing and order preserved.
- `TestParseAgentDef_ShouldReturnNilWhenCategoryPathMissing` — agents without the key parse with `CategoryPath == nil` (root-level).
- `TestParseAgentDef_ShouldReturnNilWhenCategoryPathEmptyArray` — `category_path: []` normalizes to nil (no synthetic folder).
- `TestParseAgentDef_ShouldTrimWhitespaceSegments` — `["  Marketing  ", "Sales"]` normalizes to `["Marketing", "Sales"]`.
- `TestParseAgentDef_ShouldRejectBlankSegment` — `["Marketing", ""]` fails validation with a message naming `agent.category_path[1]`.
- `TestParseAgentDef_ShouldRejectWhitespaceOnlySegment` — `["   "]` fails validation with the blank-segment message.
- `TestParseAgentDef_ShouldRejectDotSegment` — `["."]` fails with a message naming the invalid segment.
- `TestParseAgentDef_ShouldRejectDotDotSegment` — `[".."]` fails with the invalid-segment message.
- `TestParseAgentDef_ShouldRejectForwardSlashInSegment` — `["Marketing/Sales"]` fails (no slash-string fallback).
- `TestParseAgentDef_ShouldRejectBackslashInSegment` — `["Marketing\\Sales"]` fails.
- `TestParseAgentDef_ShouldRejectNonArrayValue` — `category_path: "Marketing"` fails with a strict-yaml decode error (no scalar fallback).
- `TestParseAgentDef_ShouldRejectCategoriesAliasKey` — `categories: [Marketing]` fails with `ErrInvalidAgentFrontmatterKey` (or equivalent strict yaml unknown-key error). Confirms zero alias support.
- `TestEditAgentDefFile_ShouldPreserveCategoryPathOnSkillToggle` — load fixture with `category_path`, mutate `Skills.Disabled`, write, re-read; assert `category_path` is preserved verbatim in YAML.
- `TestCloneAgentDef_ShouldDeepCopyCategoryPath` — mutating the source slice after clone must not affect the clone.
- `TestValidateAgentResourceSpec_ShouldNormalizeAndValidateCategoryPath` — feeds a spec with whitespace + invalid segments; first round normalizes, second rejects with `errors.Is(err, resources.ErrValidation)`.
- `TestNormalizeAgentCategoryPath_ShouldReturnNilForEmptyInput` — explicit unit test for the helper.

Each test uses `t.Run("Should ...")` subtests with `t.Parallel()` per `agh-test-conventions`.

#### `internal/workspace`

- `TestCloneAgentDefs_ShouldPreserveCategoryPath` — workspace clone round-trips the field.
- `TestCloneAgentDefs_ShouldPreserveSkills` — regression test for the pre-existing skills-clone gap, now fixed by delegating to `aghconfig.CloneAgentDef`.

#### `internal/api/core` + `internal/api/contract`

- `TestAgentPayloadFromDef_ShouldCopyCategoryPath` — extends the existing `TestAgentPayloadFromDef`.
- `TestAgentPayloadFromDef_ShouldDefensivelyCopyCategoryPath` — mutating the source after conversion must not leak into the payload.
- `TestAgentPayloadFromDiagnostic_ShouldOmitCategoryPath` — diagnostic placeholder rows leave the field nil.
- `TestAgentPayload_JSONShape_ShouldOmitNilCategoryPath` — encode/decode confirms `omitempty` + array shape.
- `TestBundleAgentPayload_ShouldRoundTripCategoryPath` — bundle activation payload preserves the field through encode/decode.

#### Bundle materialization / daemon sync (`internal/bundles`, `internal/daemon`)

- `TestBundleProjector_ShouldProjectCategoryPath` — bundle materialization/projection includes the field on each `BundleAgentPayload`.
- `TestBundleActivationPayload_ShouldExposeCategoryPathOnAgents` — verifies the activation-conversion seam.
- `TestDaemonResourceSync_ShouldStoreCategoryPath` — round-trips through resource validate + storage.

#### CLI (`internal/cli`)

- `TestAgentList_ShouldRenderCategoryColumn_Human` — human output contains `Marketing / Sales` for a categorized agent and an empty cell for a root-level agent.
- `TestAgentList_ShouldRenderCategoryColumn_Toon` — toon output includes a `category` key with the same formatting.
- `TestAgentList_ShouldExposeCategoryPath_JSON` — `-o json` output includes `category_path` as an array exactly as parsed.
- `TestAgentInfo_ShouldRenderCategoryPath_AllFormats` — same triple coverage for `agent info`.
- `TestWorkspaceAgents_ShouldRenderCategoryColumn` — workspace agent table view includes the column.

#### Native/agent-manageable surfaces

- `TestNativeWorkspaceDescribe_ShouldExposeCategoryPath` — `agh__workspace_describe` returns `category_path` on each agent.
- `TestToolsTransportParity_WorkspaceDescribe_ShouldIncludeCategoryPath` — extends the existing transport-parity test.

#### Web utilities + components (Vitest + Testing Library)

- **`agent-category.test.ts`:**
  - `Should build a flat list when no agent has category_path`.
  - `Should group agents by single-segment category_path`.
  - `Should build nested folders for multi-segment paths`.
  - `Should sort folders before leaves`.
  - `Should sort siblings case-insensitively by visible label`.
  - `Should derive deterministic folder IDs from joined segments`.
  - `Should derive deterministic leaf IDs from agent names`.
  - `Should render root-level leaves alongside top-level folders (no Uncategorized)`.
  - `Should treat undefined and empty-array category_path as root-level`.
- **`AgentCategoryTree.test.tsx`:**
  - `Should render category folders and agent leaves`.
  - `Should render the active route with data-active=true`.
  - `Should render the active-session dot for active agents`.
  - `Should preserve agent-row-${name}, agent-active-${name}, agent-status-dot-${name} test IDs`.
  - `Should expand ancestors of the active agent on initial render`.
  - `Should expand top-level categories on initial render when no agent is active`.
  - `Should support keyboard navigation between siblings, into folders, and onto leaves` (Arrow keys + Enter via `userEvent`).
  - `Should render the loading state with agents-loading test ID`.
  - `Should render the empty state with agents-empty test ID`.
  - `Should render the error state with agents-empty test ID` (current behavior).
- **`AgentCommandSelect.test.tsx`:**
  - `Should filter results via keyboard search` (`userEvent.type`).
  - `Should group results by formatted category label`.
  - `Should render root-level agents under "Agents" group`.
  - `Should render an empty state when search yields no results`.
  - `Should display selected agent name, provider, and category label in the trigger`.
  - `Should call onChange with the agent name when an item is selected`.
  - `Should close the popover on selection`.
  - `Should preserve the existing trigger test IDs (session-agent-select, settings-agent-select)`.
- **`AgentCommandMultiSelect.test.tsx`:**
  - `Should render data-checked on selected items`.
  - `Should toggle items via onToggle`.
  - `Should remain open after a selection`.
  - `Should display the selected count`.
  - `Should render provider/category metadata per item`.
  - `Should preserve network-agent-option-${name} test IDs on items`.
- Updated **session-create-dialog.test.tsx**, **settings skills agent-scope test**, and **network-create-channel-dialog.test.tsx** must drive the picker via keyboard (no `selectOptions`).

#### Fixtures, stories, mocks

- `internal/testutil/e2e.AgentSeed` gets an optional `CategoryPath []string` and is used in at least one new e2e fixture exercising a categorized agent.
- `web/e2e/fixtures/runtime-seed.ts` `BrowserMockAgentSeed` gains `category_path?: string[]`; the mock builder writes it into the served `AgentPayload`.
- `app-sidebar.stories.tsx` adds a `Categorized` story with a multi-level `category_path` so the tree is visible in Storybook.
- `agent-command-select.stories.tsx` (new) demonstrates grouped, empty, and selected states.

### Integration Tests

- **Resource codec round-trip** (`internal/config/agent_resource_test.go` integration tag): write a categorized agent into the resource store, read it back via the resource API, assert structural equality and that validation errors surface as `errors.Join(resources.ErrValidation, ...)`.
- **Daemon validate/encode** (existing daemon test harness): a categorized agent flows through validation + encoding without diagnostics; an invalid category_path raises a structured error captured in diagnostics.
- **Bundle activation end-to-end** (existing bundle activation integration test): a bundle that ships a categorized agent yields the field on the activation payload over both HTTP and UDS.
- **`make test-e2e-runtime`** Go harness: a workspace seeded with `AgentDefs: []AgentSeed{{Name: "coder", CategoryPath: []string{"Engineering", "Tools"}}}` exposes the field on `GET /api/workspaces/:id` and on `agh__workspace_describe`.
- **`make test-e2e-web`** Playwright lane: with categorized fixtures, the sidebar shows the tree, the session-create dialog picker groups agents, and clicking an item navigates to `/agents/:name`.

### Negative Cases (consolidated)

The following inputs MUST fail at validation with a stable error message:

| Input | Surface | Required failure |
|---|---|---|
| `category_path: [""]` | parse + validate | "blank segment" |
| `category_path: ["   "]` | parse + validate | "blank segment" |
| `category_path: ["."]` | parse + validate | invalid-segment |
| `category_path: [".."]` | parse + validate | invalid-segment |
| `category_path: ["a/b"]` | parse + validate | must not contain '/' or '\\' |
| `category_path: ["a\\b"]` | parse + validate | must not contain '/' or '\\' |
| `category_path: "Marketing"` (scalar) | strict-yaml decode | non-array value |
| `categories: [Marketing]` (alias) | strict-yaml decode | unknown key |
| `category_path: [Marketing, Sales]` then disk edit retains | `EditAgentDefFile` | preserved verbatim |

## Development Sequencing

### Build Order

1. **`internal/config`**: add field to `AgentDef` + `parsedAgentDef`, normalization helper, validation rule. Wire into `ParseAgentDef`, `AgentDef.Validate`, `EditAgentDefFile`, `validateAgentResourceSpec`. Update `CloneAgentDef`. Land all unit tests (parse, validate, edit-roundtrip, clone, resource).
2. **`internal/workspace`**: replace `cloneAgentDefs` body with `aghconfig.CloneAgentDef`. Add tests for category + skills preservation.
3. **`internal/api/contract` + `internal/api/core`**: add field to `AgentPayload`, `BundleAgentPayload`. Update `AgentPayloadFromDef`. Add tests.
4. **`internal/daemon` + `internal/bundles`**: ensure resource sync, bundle materialization/projection, and bundle activation payload propagate the field. Add/extend integration tests.
5. **`make codegen`** then `make codegen-check` to regenerate `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`.
6. **`internal/cli`**: add Category column/key to `agent list`, `agent info`, workspace agent views (human, toon, JSON). Add CLI tests across all three formats.
7. **`internal/testutil/e2e.AgentSeed`**: add optional `CategoryPath` and update at least one e2e seed.
8. **`packages/ui`**: export `Tree`, `TreeItem`, `TreeItemLabel`, `TreeDragLine` from `packages/ui/src/index.ts`.
9. **`agh-web` dependency**: `bun add @headless-tree/react@^1.6.3 --filter agh-web`.
10. **`agh-web` agent category lib**: `agent-category.ts` + tests.
11. **`agh-web` shared command components**: `AgentCommandList`, `AgentCommandSelect`, `AgentCommandMultiSelect` + tests + stories.
12. **`agh-web` `AgentCategoryTree`** + tests + story.
13. **`agh-web` consumers**: rewire `AppSidebar`, session-create dialog, settings skills agent scope, and network create-channel dialog. Update each test to drive the picker via keyboard. Update mocks/fixtures.
14. **Docs**: add `category_path` row to AGENT.md reference docs in `packages/site`. Run `make cli-docs` to regenerate CLI reference. Run `cd packages/site && bun run source:generate` if drift is reported.
15. **Final gate**: `make verify` (codegen-check → bun-lint → bun-typecheck → bun-test → web-build → fmt → lint → test → build → boundaries).

### Technical Dependencies

- `@headless-tree/react@^1.6.3` must be installed before any `AgentCategoryTree` work compiles.
- `make codegen` must run before any web code that imports `AgentPayload.category_path` from `agh-openapi.d.ts`.
- `packages/ui` exports must land before `agh-web` imports `Tree` family from `@agh/ui`.
- Backend contract changes must land in the same PR as the consumer updates per CLAUDE.md "no partial-surface completions".

## Monitoring and Observability

This change adds metadata only and creates no new lifecycle events, hooks, or canonical event types. Specifically:

- **No new event/log fields.** `category_path` is not added to canonical events; it would couple display metadata to operational telemetry without a use case.
- **No new metrics.** Agent counts/grouping in dashboards remain transport-agnostic.
- **No new alerts.** No threshold or signal depends on category structure.
- **Existing diagnostics still cover the failure modes.** Invalid `category_path` produces a parse/validate error already routed through the existing `AgentDiagnosticPayload` path (`agent_diagnostics`); the malformed agent is surfaced through the existing list endpoint with `category_path: nil`.

If sidebar tree expansion or command picker latency becomes a concern at scale, web client telemetry already covers component render timing; no AGH-side metric is introduced.

## Technical Considerations

### Key Decisions

- **Decision:** Canonical field is the array `category_path: [..]`. **Rationale:** array preserves segment boundaries unambiguously; matches the structural shape we render. **Trade-offs:** more verbose than a slash string. **Alternatives rejected:**
  - Slash-string `"Marketing/Sales"` — rejected because it conflates separator with content (an agent author cannot include `/` in a category name) and forces the parser to invent escape rules.
  - `categories: ["Marketing", "Sales"]` (plural alias) — rejected because plural reads as a multi-tag set; `category_path` reads as a single ordered hierarchy. Greenfield-alpha allows only one canonical name; no alias.
  - Multi-tag (multiple paths per agent) — out of scope; one path or none.
- **Decision:** Display-only metadata. **Rationale:** runtime moat is in execution, not organization. Branching ACP/scheduling/permissions on category_path would introduce a hidden coupling that future refactors would have to honor. **Trade-offs:** cannot use category for permission scoping or workspace partitioning later without revisiting the contract.
- **Decision:** Backend payload stays flat; web builds the tree. **Rationale:** keeps the OpenAPI contract simple, lets HTTP/UDS/CLI consumers stay agnostic, and lets the web evolve grouping logic without API changes. **Trade-offs:** every web consumer must call the same `buildAgentCategoryTree` helper to stay consistent.
- **Decision:** No `config.toml` key. **Rationale:** there is nothing to enable/disable globally; the feature is per-agent opt-in by editing AGENT.md. **Trade-offs:** no admin override to hide the column; not needed.
- **Decision:** Replace the hand-rolled `cloneAgentDefs` with `aghconfig.CloneAgentDef`. **Rationale:** the hand-rolled clone already silently dropped `Skills`; every new field would re-introduce the same class of bug. **Trade-offs:** the workspace package now depends more heavily on the config clone authority — acceptable per the repo's "single source of truth" composition discipline.
- **Decision:** Use `@headless-tree/react` (matching `@headless-tree/core`) for the sidebar tree. **Rationale:** keyboard navigation, focus management, expand/collapse, and selection are non-trivial; a battle-tested primitive avoids reinventing them. **Trade-offs:** one new transitive web dependency.
- **Decision:** Build single + multi pickers on `cmdk` via `command.tsx` rather than custom `<select>`. **Rationale:** native `<select>` cannot show category groupings, provider chips, or keyboard search — and our existing custom multi-select is ad-hoc. **Trade-offs:** more component code than a native select, but unifies three call sites onto one shared primitive.

### Known Risks

- **Risk:** Forgetting to thread `CategoryPath` through one of the conversion seams (resource codec, bundle materialization, bundle activation payload) silently drops the field for some consumers. **Likelihood:** medium. **Mitigation:** the test matrix covers each seam by name; impact analysis enumerates them; a dedicated `BundleAgentPayload` round-trip test catches the bundle-side regression.
- **Risk:** `EditAgentDefFile` rewrites only the fields explicitly mirrored back to `parsed`. Any new field that is not added to the post-mutate copy is silently lost on the next on-disk write. **Likelihood:** high if we're not deliberate. **Mitigation:** explicit `EditAgentDefFile` round-trip test that toggles `Skills.Disabled` and asserts `category_path` survives; build-order step 1 lands this before consumers depend on the round-trip.
- **Risk:** Replacing native `<select>` elements breaks every test that uses `userEvent.selectOptions`. **Likelihood:** certain. **Mitigation:** treat the test rewrite as part of each picker swap; CI will fail loudly until each call site is updated.
- **Risk:** `AgentCategoryTree` default-expansion rules are surprising to users — e.g., expand-all on empty active agent vs collapse-all. **Likelihood:** low. **Mitigation:** stories cover both states; tests assert the expansion rule explicitly.
- **Risk:** The web tree expects deterministic IDs for `useTree`'s data loader; collisions between an agent named like a category path could produce two siblings with the same ID. **Likelihood:** low. **Mitigation:** ID prefixes (`category:` vs `agent:`) prevent collisions by construction; tested in `agent-category.test.ts`.
- **Risk:** Strict YAML on an unknown alias (`categories:`) could regress to a permissive decode if `yaml.Strict()` flags change. **Likelihood:** low. **Mitigation:** the negative test `TestParseAgentDef_ShouldRejectCategoriesAliasKey` locks the contract.
- **Risk:** Documentation drift between `packages/site` AGENT.md reference and the actual validation rules. **Likelihood:** medium. **Mitigation:** include the validation rules verbatim in the doc page; CI runs `make codegen-check` and the docs site build under `make verify`.

## Architecture Decision Records

No standalone ADR files are added for this feature. The decisions above are local to a small surface and reversible (the field is opt-in metadata). If a future change promotes `category_path` to a behavior-bearing concept (permissions, partitioning, workspace scoping), open an ADR at that time.
