# PR #9: feat: web ui v2

- **URL**: https://github.com/compozy/agh/pull/9
- **Author**: @pedronauck
- **State**: merged
- **Created**: 2026-04-09T11:29:18Z
- **Merged**: 2026-04-09T17:43:04Z

## Summary by CodeRabbit

- **New Features**
  - Skills management API and UI: list skills, view details & content, enable/disable, and marketplace install indicators
  - Knowledge system: browse, search, view, delete, and consolidate memories

- **UI Updates**
  - Redesigned collapsible app sidebar and simplified header
  - New Skills and Knowledge two‑pane pages and routes
  - Chat: agent labels, timestamps, and refined message styling
  - Global design token and font refresh (updated colors, spacing, and fonts)

## Walkthrough

Adds a new skills subsystem: API DTOs, handlers, and server wiring; a SkillsRegistry interface and registry runtime changes (load content, SetEnabled); integrates skills into daemon and servers; and implements full frontend features (skills + knowledge UIs, hooks, adapters), styling/token migration, and many tests.

## Changes

| Cohort / File(s)                                                                                                                                                                                                                                                                                                                                                                                                           | Summary                                                                                                                                                                                                                             |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **API contract & conversions** <br> `internal/api/contract/contract.go`, `internal/api/core/conversions.go`                                                                                                                                                                                                                                                                                                                | Added skill DTOs (`SkillPayload`, `ProvenancePayload`, `SkillContentResponse`, `SkillActionResponse`) and conversion helpers mapping `*skills.Skill` → contract payloads.                                                           |
| **API errors & handlers** <br> `internal/api/core/errors.go`, `internal/api/core/handlers.go`, `internal/api/core/interfaces.go`, `internal/api/core/skills.go`, `internal/api/core/skills_test.go`                                                                                                                                                                                                                        | New sentinel errors and StatusForSkillError; injected `SkillsRegistry` into BaseHandlers; added `SkillsRegistry` interface and handlers `ListSkills`, `GetSkill`, `GetSkillContent`, `EnableSkill`, `DisableSkill` plus unit tests. |
| **Server & routing wiring** <br> `internal/api/httpapi/server.go`, `internal/api/udsapi/server.go`, `internal/api/udsapi/routes.go`, `internal/api/httpapi/handlers_test.go`, `internal/api/udsapi/handlers_test.go`, `internal/api/udsapi/helpers_test.go`, `internal/api/udsapi/server_test.go`                                                                                                                          | Added `WithSkillsRegistry` option and server fields; registered `/api/skills` endpoints on HTTP and UDS; updated route/handler tests and server test setup.                                                                         |
| **Registry internals** <br> `internal/skills/registry.go`, `internal/skills/registry_test.go`, `internal/skills/loader.go`, `internal/skills/loader_test.go`, `internal/skills/types.go`, `internal/skills/bundled/bundled_test.go`                                                                                                                                                                                        | Registry now supports workspace-specific disabled overlays, `SetEnabled`, `LoadContent`, `SkillSourceName`, and refactored parsing/read-content separation; tests added/updated for SetEnabled and LoadContent.                     |
| **Daemon integration & boot** <br> `internal/daemon/boot.go`, `internal/daemon/daemon.go`                                                                                                                                                                                                                                                                                                                                  | Threaded `SkillsRegistry` through RuntimeDeps and boot wiring into HTTP/UDS server construction.                                                                                                                                    |
| **API test utilities** <br> `internal/api/testutil/apitest.go`, `internal/api/udsapi/helpers_test.go`, `internal/api/udsapi/server_test.go`                                                                                                                                                                                                                                                                                | Added `StubSkillsRegistry` test stub and adjusted test helpers/servers to supply a skills registry in tests.                                                                                                                        |
| **Frontend: skill system (types, adapters, hooks, components, route, tests)** <br> `web/src/systems/skill/types.ts`, `web/src/systems/skill/adapters/skill-api.ts`, `web/src/systems/skill/lib/*`, `web/src/systems/skill/hooks/*`, `web/src/systems/skill/components/*`, `web/src/systems/skill/index.ts`, `web/src/routes/_app/skills.tsx`, `web/src/routes/_app/-skills.test.tsx`                                       | Full frontend skill feature: Zod schemas, SkillApiError adapter, query keys/options, hooks (list/detail/content/enable/disable), UI panels (list/detail, marketplace), route and comprehensive tests.                               |
| **Frontend: knowledge system (types, adapters, hooks, components, route, tests)** <br> `web/src/systems/knowledge/types.ts`, `web/src/systems/knowledge/adapters/knowledge-api.ts`, `web/src/systems/knowledge/lib/*`, `web/src/systems/knowledge/hooks/*`, `web/src/systems/knowledge/components/*`, `web/src/systems/knowledge/index.ts`, `web/src/routes/_app/knowledge.tsx`, `web/src/routes/_app/-knowledge.test.tsx` | Added knowledge/memory system: Zod schemas, KnowledgeApiError adapter, query keys/options, hooks, list/detail UI panels, route and tests.                                                                                           |
| **Frontend: layout, sidebar & state** <br> `web/src/components/app-sidebar.tsx`, `web/src/components/app-sidebar.test.tsx`, `web/src/stores/sidebar-store.ts`, `web/src/stores/sidebar-store.test.ts`, `web/src/routes/_app.tsx`, `web/src/routes/-_app.test.tsx`                                                                                                                                                          | Refactored sidebar to controlled component with Zustand store, replaced SidebarProvider layout with explicit layout wiring, and updated tests.                                                                                      |
| **Frontend: design tokens & styles migration** <br> `web/src/styles.css`, `web/src/styles.test.ts`, `web/src/components/design-system/*`, `web/src/components/design-system/stories/*`                                                                                                                                                                                                                                     | Replaced `--ds-*` tokens with `--color-*`, swapped fonts to Inter/JetBrains Mono, adjusted radii and removed legacy component layer; added style enforcement tests and updated many design-system components.                       |
| **Frontend: chat/session UI updates** <br> `web/src/systems/session/components/*`                                                                                                                                                                                                                                                                                                                                          | Propagated `agentName` through ChatView/MessageBubble, added agent label/timestamp, replaced composer Button with native button, updated tool-call card status badges and related tests.                                            |
| **Frontend: misc generated routes & adjustments** <br> `web/src/routeTree.gen.ts`, `web/src/routes/_app/index.tsx`, `web/src/routes/_app/*`                                                                                                                                                                                                                                                                                | Added generated routes for /skills and /knowledge, simplified index empty state, and added multiple route/page tests.                                                                                                               |
| **Misc & CI/test updates** <br> `.gitignore`, many test files across backend/frontend                                                                                                                                                                                                                                                                                                                                      | Added `.firecrawl` ignore and numerous new/updated tests across systems to cover new features and migration.                                                                                                                        |

## Sequence Diagram

mermaid
sequenceDiagram
participant Client
participant HTTP_Server as HTTP Server
participant Handler as BaseHandlers
participant Workspace as WorkspaceResolver
participant Registry as SkillsRegistry
Client->>HTTP_Server: POST /api/skills/:name/enable?workspace=...
HTTP_Server->>Handler: dispatch -> EnableSkill(c)
Handler->>Workspace: resolve workspace (if provided)
alt workspace-scoped resolution
Workspace-->>Handler: ResolvedWorkspace
Handler->>Registry: ForWorkspace(ctx, resolved)
Registry-->>Handler: []*Skill (workspace list)
else global resolution
Handler->>Registry: Get(name)
Registry-->>Handler: *Skill / not found
end
alt skill not found
Handler-->>HTTP_Server: 404 Not Found
else skill found & already desired state
Handler-->>HTTP_Server: 200 { ok: true }
else skill found & state change required
Handler->>Registry: SetEnabled(name, resolved?, true)
Registry-->>Handler: nil / error
Handler->>Handler: Logger.Info("enabled")
Handler-->>HTTP_Server: 200 { ok: true }
end
