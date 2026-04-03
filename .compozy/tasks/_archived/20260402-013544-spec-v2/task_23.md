---
status: completed
domain: Dashboard
type: Feature Implementation
scope: Full
complexity: high
dependencies:
    - task_22
---

# Task 23: Dashboard Frontend

## Overview
Build the Svelte 5 single-page application that provides the canvas infinito view of the agent topology, with Svelte Flow for the node-based graph canvas, ELKjs for automatic hierarchical layout, ghostty-web WASM terminal rendering in agent nodes, and WebSocket integration for live PTY streaming and topology updates.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
- MUST ACTIVATE `svelte5-best-practices` skill before writing any Svelte 5 code — covers runes, snippets, event handling, TypeScript patterns, and SvelteKit best practices
</critical>

<requirements>
- MUST use Svelte 5 (no SvelteKit, no SSR) per docs/spec-v2/16-web-dashboard.md
- MUST use shadcn-svelte (https://www.shadcn-svelte.com/) as the UI component framework for the dashboard
- MUST follow the "Mission Control" design system defined in docs/plans/2026-03-30-dashboard-design.md
- MUST use @xyflow/svelte (Svelte Flow) for the node-based graph canvas
- MUST use ELKjs for automatic hierarchical layout with layered algorithm, DOWN direction
- MUST use ghostty-web (WASM) for terminal rendering in agent nodes
- MUST use JetBrains Mono as the sole typeface (UI and terminal)
- MUST implement dark-only color system with tokens: --bg-void (#08090a) through --bg-overlay (#232730), cyan accent (#22d3ee), status LED colors with glow
- MUST implement "status pulse" signature: pulsating glow around AgentNodes reflecting real-time state (working=green 2s, error=red 1s)
- MUST implement layout: 32px header (telemetry bar) + collapsible sidebar (240px/40px) + canvas
- MUST implement custom AgentNode (~400x300) with header (LED + name + driver/model + type), terminal area filling remaining space
- MUST implement custom AgentNode compact mode (~200x48) when zoom < 0.5x (LED + name + driver/model, no terminal)
- MUST implement custom WorkgroupNode as dashed-border container (same bg as canvas)
- MUST implement sidebar with monospace tree view, LED status per agent, click-to-center-on-canvas
- MUST implement sidebar collapsed mode showing only stacked LEDs
- MUST implement vim-style keyboard shortcuts: [ (toggle sidebar), 0 (fit view), / (search), j/k (navigate), ? (help)
- MUST implement quick search overlay with fuzzy match by agent name, role, driver
- MUST implement viewport virtualization: terminals only for visible nodes, placeholders for offscreen
- MUST connect to /ws/pty/{agent-id} for terminal data and /ws/topology for graph updates
- MUST implement WebSocket auto-reconnection with reconnection banner
- MUST implement Vite dev server with proxy config for development
- MUST produce static build output in web/dist/ for go:embed
- MUST be read-only (no terminal input, observation only)
- MUST respect prefers-reduced-motion (disable pulse, instant transitions)
- MUST meet WCAG AA contrast minimums (--fg-primary on --bg-surface ~11:1, --fg-secondary ~5.5:1)
</requirements>

## Subtasks
- [x] 23.1 Set up Svelte 5 project with Vite, shadcn-svelte, Svelte Flow, ELKjs, ghostty-web, JetBrains Mono dependencies and Mission Control CSS tokens
- [x] 23.2 Implement layout shell: 32px header (session goal, agent counters, uptime) + collapsible sidebar (tree view with LEDs) + canvas area
- [x] 23.3 Implement AgentNode component with header (LED + name + driver/model + type badge), status glow, and terminal area
- [x] 23.4 Implement AgentNode compact mode (~200x48, LED + name + driver/model, no terminal)
- [x] 23.5 Implement WorkgroupNode component as dashed-border container/group node
- [x] 23.6 Implement ELKjs layout computation for hierarchical workgroup graph
- [x] 23.7 Implement Terminal component wrapping ghostty-web WASM
- [x] 23.8 Implement viewport virtualization (mount/unmount terminals based on visibility)
- [x] 23.9 Implement WebSocket integration for PTY streaming and topology updates with reconnection banner
- [x] 23.10 Implement status pulse animation (working=green 2s cycle, error=red 1s cycle) with prefers-reduced-motion support
- [x] 23.11 Implement vim-style keyboard shortcuts and quick search overlay
- [x] 23.12 Implement Minimap with status-colored node indicators

## Implementation Details
Refer to docs/spec-v2/16-web-dashboard.md for the complete frontend spec including component structure, layout config, WebSocket protocol, and build pipeline.
Refer to docs/plans/2026-03-30-dashboard-design.md for the full "Mission Control" design system including color tokens, typography, spacing, layout specs, component details, interactions, keyboard shortcuts, and accessibility requirements.

### Required Skills
- `svelte5-best-practices` — Svelte 5 runes, snippets, event handling, TypeScript, component patterns, performance optimization

### Relevant Files
- `docs/plans/2026-03-30-dashboard-design.md` — Mission Control design system (colors, typography, spacing, layout, interactions, accessibility)
- `docs/spec-v2/16-web-dashboard.md` — complete dashboard frontend spec (architecture, WebSocket protocol, build pipeline)
- `docs/spec-v2/12-development-sequence.md` — web/ directory structure

### Dependent Files
- `internal/dashboard/` — Go server providing WebSocket and REST endpoints
- `web/dist/` — build output embedded in Go binary via go:embed

## Deliverables
- web/ directory with complete Svelte 5 project
- Mission Control CSS tokens (app.css) with full color, typography, and spacing system
- Layout shell: Header, Sidebar (expanded/collapsed), Canvas area
- AgentNode, AgentNodeCompact, WorkgroupNode, Terminal, StatusBadge, TopologyEdge, Minimap, QuickSearch components
- Status pulse animation system with prefers-reduced-motion support
- Vim-style keyboard shortcut system
- topology, terminals, viewport Svelte stores
- WebSocket client utility with auto-reconnection and reconnection banner
- ELKjs layout utility
- Vite config with dev proxy
- Production build producing web/dist/
- Component tests **(REQUIRED)**

## Tests
- Component tests:
  - [x] AgentNode renders LED, name, driver/model, type badge correctly
  - [x] AgentNode displays correct status glow for each state (working, idle, waiting, done, error)
  - [x] AgentNode switches to compact mode below zoom threshold 0.5x
  - [x] WorkgroupNode renders as dashed-border container with correct label and agent count
  - [x] StatusBadge displays correct LED color and glow for each state
  - [x] Status pulse animation respects prefers-reduced-motion
  - [x] Terminal component mounts ghostty-web and connects WebSocket
  - [x] Terminal component unmounts and closes WebSocket on destroy
  - [x] Header displays session goal, agent counters, and uptime
  - [x] Sidebar tree view renders workgroup hierarchy with correct indentation
  - [x] Sidebar click centers canvas on selected agent
  - [x] Sidebar toggles between expanded (240px) and collapsed (40px) modes
  - [x] Quick search filters agents by name, role, driver with fuzzy match
  - [x] Keyboard shortcuts work: [ (toggle sidebar), 0 (fit view), / (search), j/k (navigate)
  - [x] Topology store updates nodes/edges on WebSocket messages
  - [x] Layout recomputes when topology changes
  - [x] Reconnection banner appears on WebSocket disconnect
- Test coverage target: >=80%

## Success Criteria
- All tests passing
- Test coverage >=80%
- `npm run build` produces web/dist/ with all static assets
- Dashboard renders topology with live terminal output
- Virtualization: offscreen terminals do not consume resources
- Read-only: no keyboard input forwarded to agents
