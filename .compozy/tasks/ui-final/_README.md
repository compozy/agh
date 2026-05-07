# UI/UX Final Audit — release prep

Per-route deep UI/UX audits ahead of the AGH alpha release. One subagent per top-level module, one report file per route. Every claim in every report MUST cite evidence (file:line, screenshot, or live probe).

## Probe targets

- **Web SPA (vite dev):** `http://localhost:5173`
- **Storybook:** `http://localhost:6006`
- **Daemon (Go runtime, API only):** `http://localhost:2123` — note: web is empty data; rely on Storybook for populated states.

## Skills used by every subagent

- `/agent-browser` — live DOM/screenshot inspection of both the SPA and Storybook.
- `/impeccable` (specifically `critique` and the shared design laws) — deep UI/UX audit, AI-slop test, Nielsen scoring, persona red flags, prioritised findings.
- `agh-design`, `design-taste-frontend`, `minimalist-ui` — register-aware design grammar (already authoritative via `DESIGN.md` + `COPY.md`).

## Layout

```
.compozy/tasks/ui-final/
├── _README.md                # this file
├── _TEMPLATE.md              # mandatory report template
├── 01_dashboard/
├── 02_agents/
├── 03_network/
├── 04_tasks/
├── 05_jobs/
├── 06_triggers/
├── 07_knowledge/
├── 08_skills/
├── 09_bridges/
├── 10_sandbox/
└── 11_settings/
```

Each module folder holds:
- `00_module_overview.md` — cross-route synthesis (consistency, shared components, IA gaps).
- `<NN>_analysis_<route>.md` — one file per route, following `_TEMPLATE.md` verbatim.
- `_evidence/<route>/<file>.png|.json` — screenshots, DOM snapshots, console logs, network HARs.

## Authoring rules

1. Use `_TEMPLATE.md` verbatim. Do not delete sections — mark them `n/a — <reason>` if genuinely irrelevant.
2. Every finding cites evidence. No "general impression" findings.
3. Storybook is authoritative for populated states (daemon has no data).
4. Truthful UI > plausible UI: if the daemon doesn't support a control or metric, the UI must not pretend it does. Flag any drift.
5. `DESIGN.md` and `COPY.md` are authoritative. Do not invent tokens, fonts, or copy.
6. No em dashes in reports (`—` or `--`).
7. Be honest. Most routes will score 20–32 / 40 on Nielsen. A score of 38+ requires evidence.
8. P0 findings are *ship-blockers* — use sparingly and back them up.
