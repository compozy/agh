# TechSpec: Web UI Redesign

## Executive Summary

Redesign the AGH web frontend to match the Paper design system — replacing the current OKLCH/Geist/Bricolage visual layer with DESIGN.md's hex/Inter/JetBrains Mono flat model, building a custom two-zone sidebar (workspace icon rail + panel), updating the session chat view, and adding Skills (Installed + Marketplace) and Knowledge pages with full backend API support.

**Key architectural decisions:**
- Full replacement of design tokens (no migration layer) — ADR-001
- Custom sidebar replacing shadcn Sidebar — ADR-002
- Full systems architecture (adapters + hooks + components) for Skills and Knowledge from day one — ADR-003
- Foundation-first build order with parallelizable page work — ADR-004

**Primary trade-off:** The full-replace approach requires touching every existing component in one pass, but eliminates technical debt and ensures visual consistency from the start. This is acceptable because the app is greenfield alpha with zero production users.

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  web/src/                                                       │
│  ├── styles.css              ← DESIGN.md tokens (hex, flat)     │
│  ├── components/                                                │
│  │   ├── app-sidebar.tsx     ← Custom icon rail + panel         │
│  │   ├── app-header.tsx      ← Breadcrumb / session header      │
│  │   ├── ui/                 ← shadcn (themed to DESIGN.md)     │
│  │   └── shared/             ← Reusable DS components           │
│  ├── routes/                                                    │
│  │   ├── _app.tsx            ← Layout: sidebar + content        │
│  │   └── _app/                                                  │
│  │       ├── index.tsx       ← Empty state                      │
│  │       ├── session.$id.tsx ← Chat view (redesigned)           │
│  │       ├── skills.tsx      ← Skills page (3-panel)            │
│  │       └── knowledge.tsx   ← Knowledge page (3-panel)         │
│  └── systems/                                                   │
│      ├── agent/              ← Existing (unchanged)             │
│      ├── daemon/             ← Existing (unchanged)             │
│      ├── session/            ← Existing (components updated)    │
│      ├── workspace/          ← Existing (icon rail additions)   │
│      ├── skill/              ← NEW: Skills data layer           │
│      └── knowledge/          ← NEW: Knowledge data layer        │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  internal/api/                                                  │
│  ├── contract/contract.go    ← New DTOs for Skills              │
│  ├── core/handlers.go        ← New SkillsRegistry field + handlers │
│  └── httpapi/server.go       ← New /api/skills routes           │
└─────────────────────────────────────────────────────────────────┘
```

**Data flow:**
- Frontend systems follow unidirectional flow: `adapters → lib → hooks → components`
- Backend handlers follow: `Gin router → BaseHandlers → Domain service → Contract payload`
- Session chat: SSE streaming via `useSessionChat` (unchanged)
- Skills + Knowledge: TanStack Query with `refetchInterval` polling

## Implementation Design

### Core Interfaces

**Skills HTTP handler (Go, backend):**

```go
// SkillPayload is the HTTP response type for a skill.
type SkillPayload struct {
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Version     string            `json:"version,omitempty"`
    Source      string            `json:"source"`
    Enabled     bool              `json:"enabled"`
    Dir         string            `json:"dir"`
    Content     string            `json:"content,omitempty"`
    Metadata    map[string]any    `json:"metadata,omitempty"`
    Provenance  *ProvenancePayload `json:"provenance,omitempty"`
}
```

**Skill system adapter (TypeScript, frontend):**

```typescript
// systems/skill/adapters/skill-api.ts
export type SkillPayload = {
  name: string;
  description: string;
  version?: string;
  source: "bundled" | "marketplace" | "user" | "workspace" | "additional";
  enabled: boolean;
  dir: string;
  content?: string;
  metadata?: Record<string, unknown>;
  provenance?: { slug: string; registry: string; version: string };
};

export async function listSkills(
  workspaceId: string,
  signal?: AbortSignal,
): Promise<SkillPayload[]>;
export async function getSkill(
  name: string, workspaceId: string, signal?: AbortSignal,
): Promise<SkillPayload>;
```

**Knowledge system adapter (TypeScript, frontend):**

```typescript
// systems/knowledge/adapters/knowledge-api.ts
export type MemoryHeader = {
  filename: string;
  mod_time: string;
  name: string;
  description?: string;
  type: "user" | "feedback" | "project" | "reference";
  agent_name?: string;
};

export async function listMemories(
  scope?: string, workspace?: string, signal?: AbortSignal,
): Promise<MemoryHeader[]>;
export async function readMemory(
  scope: string, filename: string, signal?: AbortSignal,
): Promise<string>;
export async function deleteMemory(
  scope: string, filename: string,
): Promise<void>;
```

### Data Models

**Backend — New contract types (`internal/api/contract/contract.go`):**

| Type | Fields | Usage |
|------|--------|-------|
| `SkillPayload` | name, description, version, source, enabled, dir, content, metadata, provenance | GET /api/skills, GET /api/skills/:name |
| `ProvenancePayload` | slug, registry, version, installed_at | Nested in SkillPayload |
| `SkillActionResponse` | ok (bool) | POST enable/disable responses |

**Frontend — System types:**

| System | Types File | Key Types |
|--------|-----------|-----------|
| `skill` | `systems/skill/types.ts` | `SkillPayload`, `SkillSource`, `SkillFilter` |
| `knowledge` | `systems/knowledge/types.ts` | `MemoryHeader`, `MemoryScope`, `MemoryType`, `KnowledgeFilter` |

### API Endpoints

**New Skills endpoints (backend):**

| Method | Path | Description | Request | Response |
|--------|------|-------------|---------|----------|
| GET | `/api/skills?workspace=:id` | List all skills for workspace | query: workspace (required) | `{"skills": SkillPayload[]}` |
| GET | `/api/skills/:name?workspace=:id` | Get skill detail | query: workspace (required) | `{"skill": SkillPayload}` |
| POST | `/api/skills/:name/enable?workspace=:id` | Enable skill | query: workspace (required) | `{"ok": true}` |
| POST | `/api/skills/:name/disable?workspace=:id` | Disable skill | query: workspace (required) | `{"ok": true}` |

**Existing Memory endpoints (already implemented, used by Knowledge page):**

| Method | Path | Description | Request | Response |
|--------|------|-------------|---------|----------|
| GET | `/api/memory?scope=:scope&workspace=:ws` | List memory headers | query: scope, workspace | `MemoryHeader[]` |
| GET | `/api/memory/:scope/:filename` | Read memory content | path: scope, filename | `{"content": string}` |
| DELETE | `/api/memory/:scope/:filename` | Delete memory | path: scope, filename | `{"ok": true}` |
| POST | `/api/memory` | Write memory | body: content, scope, workspace | `{"ok": true}` |
| POST | `/api/memory/consolidate` | Trigger dream consolidation | body: workspace | `{"ok": true}` |

### Frontend Routes

| Route | File | Description |
|-------|------|-------------|
| `/_app/` | `routes/_app/index.tsx` | Empty state ("Select a session to begin") |
| `/_app/session/$id` | `routes/_app/session.$id.tsx` | Session chat view |
| `/_app/skills` | `routes/_app/skills.tsx` | Skills page (Installed + Marketplace tabs) |
| `/_app/knowledge` | `routes/_app/knowledge.tsx` | Knowledge page (All/Global/Workspace tabs) |

## Design Token Replacement

### styles.css — Complete Replacement

**Remove:**
- `@fontsource-variable/geist`, `@fontsource/bricolage-grotesque/*` imports
- All `--ds-*` OKLCH custom properties
- All `.ds-texture-*`, `.ds-panel*` CSS classes with `::before`/`::after` pseudo-elements
- `--ds-shadow-*`, `--ds-canvas-glow`, `--ds-vignette`, `--ds-texture-line`
- `color-mix()` expressions

**Add:**
- `@fontsource-variable/inter` (400, 500, 600, 700), `@fontsource/jetbrains-mono` (500, 600) imports
- DESIGN.md hex-based custom properties:

```css
:root {
  /* Backgrounds */
  --color-canvas: #121212;
  --color-surface: #1C1C1E;
  --color-surface-elevated: #2C2C2E;
  --color-divider: #3A3A3C;
  --color-hover: #333336;
  --color-disabled: #48484A;

  /* Text */
  --color-text-primary: #E5E5E7;
  --color-text-secondary: #8E8E93;
  --color-text-tertiary: #636366;
  --color-text-label: #98989D;

  /* Accent & Semantic */
  --color-accent: #E8572A;
  --color-accent-hover: #D14E25;
  --color-success: #30D158;
  --color-danger: #FF453A;
  --color-warning: #FFD60A;
  --color-info: #BF5AF2;
}
```

**shadcn theme mapping:**

| shadcn variable | DESIGN.md token |
|----------------|----------------|
| `--background` | `#121212` (canvas) |
| `--foreground` | `#E5E5E7` (text-primary) |
| `--card` | `#1C1C1E` (surface) |
| `--popover` | `#2C2C2E` (elevated) |
| `--primary` | `#E8572A` (accent) |
| `--primary-foreground` | `#FFFFFF` |
| `--secondary` | `transparent` |
| `--secondary-foreground` | `#E5E5E7` |
| `--muted` | `#2C2C2E` (elevated) |
| `--muted-foreground` | `#8E8E93` (text-secondary) |
| `--destructive` | `#FF453A` (danger) |
| `--border` | `#3A3A3C` (divider) |
| `--input` | `#3A3A3C` (divider) |
| `--ring` | `#E8572A` (accent) |
| `--radius` | `0.5rem` (8px default) |
| `--sidebar` | `#1C1C1E` (surface) |
| `--sidebar-border` | `#3A3A3C` (divider) |
| `--sidebar-primary` | `#E8572A` (accent) |

**Font theme:**

```css
@theme inline {
  --font-sans: "Inter Variable", -apple-system, BlinkMacSystemFont, sans-serif;
  --font-mono: "JetBrains Mono", "Courier New", monospace;
}
```

Remove `--font-display` entirely. No display/heading font per DESIGN.md.

**Flat depth model:**
- Remove all `box-shadow` declarations
- Remove `--ds-shadow-panel` and `--ds-shadow-focus`
- Focus ring: `1.5px solid #E8572A` (border, not shadow)

## Custom Sidebar Component

### Structure

```
┌──────┬─────────────────────────┐
│ Rail │  Panel (~220px)         │
│40px  │                         │
│      │  ┌─ Header ──────────┐  │
│ [A]  │  │ "Polybot" 🔍 ➕   │  │
│      │  └───────────────────┘  │
│ [P]  │  ┌─ AGENTS ─────────┐  │
│      │  │ Coder        3 > │  │
│ [D]  │  │ Writer       1 > │  │
│      │  │ Researcher   2 > │  │
│ [N]  │  │ General      0 > │  │
│      │  └───────────────────┘  │
│      │  ┌─ WORKSPACE ──────┐  │
│      │  │ 📖 Knowledge     │  │
│      │  │ 🔧 Skills        │  │
│      │  └───────────────────┘  │
│      │                         │
│      │  ┌─ SYSTEM ─────────┐  │
│      │  │ ● Connected v0.1 │  │
│ [+]  │  │ ⚙ Settings       │  │
│      │  └───────────────────┘  │
└──────┴─────────────────────────┘
```

### Props & State

- Sidebar collapse state: Zustand store (`useSidebarStore`)
- Active workspace: derived from workspace list + selection
- Icon rail always visible; panel collapsible to 0px width

### Agent List with Expandable Sessions

Each agent in the sidebar is expandable. Clicking the chevron reveals sessions for that agent. Clicking the agent name + "+" creates a new session. This matches the current `AgentSidebarGroup` pattern but styled per Paper:
- Agent avatar: 24px circle with letter, colored by agent type
- Session count + chevron right-aligned
- Active session: 3px left accent bar `#E8572A`

### Navigation Items

Sidebar adds navigation items below the agent list:
- Knowledge → navigates to `/_app/knowledge`
- Skills → navigates to `/_app/skills`
- These use the active indicator pattern: 3px left accent bar when route matches

## Page Designs

### Skills Page (`/_app/skills`)

**Three-panel layout:** sidebar (shared) + skill list panel + skill detail panel.

**Header bar:**
- Icon + "Skills" title + count badge
- Tab pills: INSTALLED (default) | MARKETPLACE
- Search input in list panel

**Installed tab:**
- Grouped skill list: BUNDLED, WORKSPACE, MARKETPLACE sections
- Each item: status dot + name + version right-aligned
- Selected item: bg `#2C2C2E` with 3px left accent bar
- Detail panel: name, version, source badge (BUNDLED), enabled status, description, content preview card, Disable/View in CLI buttons

**Marketplace tab:**
- Full-width search input ("Search skills on ClawHub...")
- Category filter chips: ALL, TESTING, DATABASE, DEPLOY, AI, DEVOPS, SECURITY
- Sort control: "Sort by: Downloads ↓"
- Marketplace rows: name, @author, version, tags (bordered pills), download count, INSTALL/INSTALLED button

### Knowledge Page (`/_app/knowledge`)

**Three-panel layout:** sidebar (shared) + knowledge list panel + knowledge detail panel.

**Header bar:**
- Icon + "Knowledge" title + count badge
- Tab pills: ALL (default) | GLOBAL | WORKSPACE
- Dream status indicator: "● Dream: 3h ago" right-aligned
- Search input in list panel

**List panel:**
- Grouped by scope: GLOBAL, WORKSPACE with counts
- Each item: title, description, date right-aligned, type+scope badges below
- Selected item: bg `#2C2C2E` with 3px left accent bar

**Detail panel:**
- Title, version, status (● Active), file path
- DESCRIPTION section with text
- CONTENT section: preview card with title, description, bullet points, "View full content →" link
- Action buttons: Delete, View in CLI
- METADATA section: striped key-value table (type, scope, agent, modified)

### Session Chat View (Updated Styling)

The existing session chat view components get updated styling to match Paper:
- User message: right-aligned bubble, bg `#2C2C2E`, radius 12px
- Agent message: left-aligned, no bubble, agent label with status dot + JetBrains Mono name
- Tool call card: bg `#1C1C1E`, border `#3A3A3C`, terminal icon, status badge (DONE/RUNNING/ERROR)
- Chat input: bg `#1C1C1E`, radius 12px, border `#3A3A3C`, focused border `#E8572A`
- Send button: 36px circle, bg `#E8572A`, white send icon

### Empty State (Index Page)

- Centered terminal icon (48px, `#636366`)
- "Select a session to begin" — Inter 15px Medium, `#8E8E93`
- "or create a new one from the sidebar" — Inter 13px Regular, `#636366`

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `web/src/styles.css` | Modified (breaking) | Complete token replacement. High risk — all components affected. | Replace all tokens, verify every component renders. |
| `web/src/components/app-sidebar.tsx` | Modified (rewrite) | Replace shadcn Sidebar with custom two-zone component. High risk — navigation depends on it. | Build icon rail + panel from scratch. |
| `web/src/components/app-header.tsx` | Modified | Update styling to match new tokens. Low risk. | Update classes and colors. |
| `web/src/routes/_app.tsx` | Modified | Remove SidebarProvider/SidebarInset, use custom layout. Medium risk. | Update layout wrapper. |
| `web/src/routes/_app/index.tsx` | Modified | Update empty state styling. Low risk. | Update to match Paper empty state. |
| `web/src/routes/_app/session.$id.tsx` | Modified | Update chat component styling. Medium risk. | Restyle chat components per Paper. |
| `web/src/routes/_app/skills.tsx` | New | Skills page with three-panel layout. | Create route + wire to skill system. |
| `web/src/routes/_app/knowledge.tsx` | New | Knowledge page with three-panel layout. | Create route + wire to knowledge system. |
| `web/src/systems/skill/` | New | Full skill data layer. | Create adapters, hooks, components, types. |
| `web/src/systems/knowledge/` | New | Full knowledge data layer. | Create adapters, hooks, components, types. |
| `internal/api/contract/contract.go` | Modified | Add SkillPayload, ProvenancePayload types. Low risk. | Add new DTOs. |
| `internal/api/core/handlers.go` | Modified | Add SkillsRegistry field, skill handler methods. Medium risk. | Add ListSkills, GetSkill, EnableSkill, DisableSkill handlers. |
| `internal/api/core/conversions.go` | Modified | Add SkillPayloadFromSkill conversion. Low risk. | Add conversion helper. |
| `internal/api/httpapi/server.go` | Modified | Register new /api/skills routes. Low risk. | Add route group. |
| `web/package.json` | Modified | Swap font packages. Low risk. | Remove geist/bricolage, add inter/jetbrains-mono. |

## Testing Approach

### Unit Tests (Frontend)

- **Design tokens**: Snapshot test that `styles.css` contains no OKLCH, no shadow, no gradient
- **Sidebar**: Test icon rail renders workspace circles, panel renders agent list, collapse state toggles
- **Skill system**: Test `listSkills`, `getSkill` adapters return typed data. Test query hooks with MSW-mocked endpoints.
- **Knowledge system**: Test `listMemories`, `readMemory`, `deleteMemory` adapters. Test query hooks.
- **Chat components**: Test user message renders right-aligned, agent message left-aligned, tool call card shows status badge

### Unit Tests (Backend)

- **Skill handlers**: Table-driven tests for ListSkills, GetSkill, EnableSkill, DisableSkill with mock registry
- **Contract types**: Test JSON serialization of SkillPayload with all fields, with optional fields omitted
- **Conversion helpers**: Test SkillPayloadFromSkill produces correct JSON field mapping

### Integration Tests (Backend)

- **Skills API**: Real HTTP requests against test server with real skills registry (loaded from `t.TempDir()`)
- **Memory API**: Already covered by existing tests — verify Knowledge page scenarios work

### Visual Verification

- Compare each implemented page screenshot against Paper artboard
- Verify color values match DESIGN.md hex tokens exactly
- Verify typography: Inter for content, JetBrains Mono for meta labels (uppercase, tracked)

## Development Sequencing

### Build Order

1. **Design tokens + fonts** — Replace `styles.css`, swap font packages, update shadcn theme mapping. Update all existing component styles to use new tokens. **No dependencies.** Deliverable: `make web-lint && make web-typecheck` passes, existing pages render with new visual language.

2. **Custom sidebar + app shell layout** — Build two-zone sidebar (icon rail + panel), update `_app.tsx` layout. Add Knowledge and Skills nav items linking to empty routes. **Depends on step 1.** Deliverable: sidebar matches Paper, navigation works.

3. **Session chat view update** — Restyle chat header, messages, tool calls, composer to match Paper. **Depends on steps 1-2.** Deliverable: session page matches Paper artboard "AGH Sidebar — Sessions in Header".

4. **Skills backend endpoints** — Add SkillPayload to contract, add handlers to BaseHandlers, register routes. **Depends on step 1 (contract types only).** Deliverable: `GET /api/skills`, `GET /api/skills/:name`, `POST enable/disable` working.

5. **Skills frontend system + page** — Create `systems/skill/`, build three-panel Skills page (Installed + Marketplace). **Depends on steps 2, 4.** Deliverable: Skills page matches Paper artboards.

6. **Knowledge frontend system + page** — Create `systems/knowledge/`, build three-panel Knowledge page. Uses existing memory endpoints. **Depends on step 2.** Deliverable: Knowledge page matches Paper artboard.

Steps 3, 4, 5, and 6 can run in parallel after step 2 completes (with step 5 also requiring step 4).

### Technical Dependencies

- **Fonts**: `bun add @fontsource-variable/inter @fontsource/jetbrains-mono` and `bun remove @fontsource-variable/geist @fontsource/bricolage-grotesque`
- **Backend**: Skills registry must be injected into `BaseHandlers` (currently available via daemon composition root)
- **Memory endpoints**: Already exist — no backend work needed for Knowledge page

## Monitoring and Observability

- **Skills endpoints**: Log at `slog.Info` level for list/get, `slog.Warn` for enable/disable failures
- **Frontend errors**: Existing TanStack Query error handling with `sonner` toasts
- **Metrics**: Track skill list latency via existing observe system; no new metrics infrastructure needed

## Technical Considerations

### Key Decisions

1. **Full token replacement over incremental migration** — Greenfield alpha, no users to disrupt. Clean break is cheaper than maintaining two systems. Trade-off: all components need updating in one batch.

2. **Custom sidebar over shadcn extension** — Paper's icon rail + panel layout is fundamentally different from shadcn Sidebar's single-panel model. Custom code is simpler than fighting the library.

3. **TanStack Query + polling for Skills/Knowledge** — These are CRUD pages, not real-time streams. SSE would add complexity without user benefit. Trade-off: data can be up to 60s stale (acceptable for skills/knowledge).

4. **Full systems from day one** — Avoids mock data divergence and double-work. Trade-off: requires backend and frontend to advance in lockstep.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Token replacement breaks existing session page | Medium | Visual diff testing before/after for every component |
| Custom sidebar accessibility gaps | Medium | Use semantic HTML (`nav`, `aside`), ARIA labels, keyboard navigation testing |
| Skills registry API surface gaps | Low | Review `Registry` methods before writing handlers; public API is well-defined |
| Font loading performance regression | Low | Use `font-display: swap`, preload critical weights (Inter 400, 500) |

## Architecture Decision Records

- [ADR-001: Full Replace of Design Token System](adrs/adr-001.md) — Delete OKLCH/Geist/Bricolage tokens, rebuild with DESIGN.md hex/Inter/JetBrains Mono flat model
- [ADR-002: Custom Sidebar with Workspace Icon Rail](adrs/adr-002.md) — Replace shadcn Sidebar with custom two-zone component (40px icon rail + 220px collapsible panel)
- [ADR-003: Full Systems Architecture for Skills and Knowledge](adrs/adr-003.md) — Build complete data layers (backend endpoints + frontend systems) from day one, no mock data
- [ADR-004: Foundation-First Build Order](adrs/adr-004.md) — Tokens → Sidebar → Pages (session, skills, knowledge) with steps 3-6 parallelizable
