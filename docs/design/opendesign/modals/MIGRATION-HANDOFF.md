# Modal markup migration — execution handoff

Scope: migrate the markup of the 16 modal HTMLs in `modals/` to the new production-parity anatomy already shipped in `modal-system.css` and rendered in `modal-design-system.html`. This document is the complete work order: every change, exact snippets, per-file matrix, validation protocol, and acceptance criteria. Execute it without re-deriving design decisions.

## 1. Context — what is already done vs. what this migration does

Already done (do NOT redo, do NOT modify):
- `modal-system.css` was rewritten against the production geometry extracted from `packages/ui/src` and `web/src/systems` (inputs 36px on `--color-elevated`, buttons 22/26/30px at radius 8 weight 500, runtime trigger 34px with `--color-line-strong` border, PillGroup 24px segments in a borderless 2px track, neutral RadioCard selection, accent selection only inside the runtime catalog popup, 7-bar intensity meter component `.im`).
- `modal-design-system.html` is the rendered canonical reference. Open it before starting; every pattern below is visible there.
- `MODAL-STANDARD.md` records the resolved contracts (header pattern, control geometry, runtime accent-selection exception).

What remains — this migration: the 16 modal HTML files still carry the previous markup generation. They inherit the new CSS automatically, but their markup lacks the new anatomy: no icon-well headers, unicode glyphs instead of SVGs, glyph-string meters, flat popover options. Your job is markup-only migration, file by file.

## 2. Authority chain (on conflict, higher wins)

1. `packages/ui/src/tokens.css` + production components in `packages/ui/src/components` and `web/src/systems` (runtime truth).
2. `modals/MODAL-STANDARD.md` (resolved contracts).
3. `modals/modal-design-system.html` (rendered reference — copy its patterns verbatim).
4. This handoff (sequencing and per-file specifics).

## 3. Hard constraints — violating any of these fails the migration

- **Markup-only.** Do not change any CSS value in `modal-system.css`. Do not rename classes. The only permitted JS change is the one-line chip glyph fix in §4.7.
- **Preserve every JS hook**: `aria-controls`, `data-component-trigger`, `data-runtime-segment`, `data-pressed-group`, `data-value`, `data-component-value`, `data-component-summary`, `data-selection-noun`, `data-chip-target`, `data-mode`, `data-adv`, `data-pillgroup`, `data-choice-group`, `data-choice`, `data-conditional`, `data-reveal*`, `data-close`, `data-od-id`, popover element IDs, and `role` attributes. `modal-system.js` queries all of these; a dropped attribute silently kills behavior.
- **No inline `style=""` attributes, no per-page `<script>` blocks** (forbidden drift per MODAL-STANDARD).
- **Colors only via `var(--color-*)`** — but this migration should not need any color at all; SVGs use `stroke="currentColor"`.
- **No invented data.** When enriching popover options (§4.8), only restructure text already present in the file. Never add context-window sizes, effort counts, member counts, or paths that the file does not already contain.
- **Do not "fix" intentional CSS behaviors**: neutral RadioCard selection (glaze + rim, no accent), accent-tint selected rows only inside `role="dialog"` popovers, `--color-subtle` placeholders. These are production-derived; leave them.
- Artifacts in English. Keep existing copy untouched except where a snippet below says otherwise.

## 4. The changes

All SVGs use this wrapper (stroke icons, 24 viewBox):

```html
<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">…</svg>
```

Icon path library — use these exact paths, do not substitute your own:

| id | inner paths |
|---|---|
| `bot` | `<rect x="4" y="8" width="16" height="12" rx="2"/><path d="M12 8V5"/><circle cx="12" cy="3.5" r="1"/><path d="M9 13v2M15 13v2"/>` |
| `play` | `<polygon points="6 4 20 12 6 20 6 4"/>` |
| `zap` | `<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>` |
| `book` | `<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>` |
| `server` | `<rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/>` |
| `hash` | `<line x1="4" y1="9" x2="20" y2="9"/><line x1="4" y1="15" x2="20" y2="15"/><line x1="10" y1="3" x2="8" y2="21"/><line x1="16" y1="3" x2="14" y2="21"/>` |
| `shield` | `<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>` |
| `key` | `<circle cx="7.5" cy="15.5" r="5.5"/><path d="M21 2l-9.6 9.6"/><path d="M15.5 7.5l3 3"/>` |
| `layers` | `<path d="M3 7l9-4 9 4-9 4-9-4z"/><path d="M3 7v10l9 4 9-4V7"/>` |
| `check` | `<polyline points="20 6 9 17 4 12"/>` (use `stroke-width="2.4"`) |
| `x` | `<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>` |
| `chevron-down` | `<polyline points="6 9 12 15 18 9"/>` |
| `chevrons-up-down` | `<polyline points="7 9 12 4 17 9"/><polyline points="7 15 12 20 17 15"/>` |
| `search` | `<circle cx="11" cy="11" r="7"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>` |
| `refresh` | `<polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>` |
| `star` | `<polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>` |
| `home` | `<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/>` |
| `plus` | `<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>` |

### 4.1 Header: icon well + eyebrow (14 files)

Every non-sheet modal header currently looks like:

```html
<header class="dialog__head" data-od-id="…-header">
  <div class="head-copy">
    <h1 data-od-id="…-title">…</h1>
```

Transform to (insert the icon well as the first child of the header, and the eyebrow as the first child of `.head-copy`):

```html
<header class="dialog__head" data-od-id="…-header">
  <div class="head-icon" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">{ICON}</svg></div>
  <div class="head-copy">
    <div class="eyebrow">{EYEBROW}</div>
    <h1 data-od-id="…-title">…</h1>
```

Per-file icon + eyebrow (CSS already renders the well at 36px with the accent tint and the eyebrow in `--color-accent-strong`; do not add classes beyond `head-icon`):

| file | icon | eyebrow |
|---|---|---|
| `create-agent.html` | bot | `Operate · Agent` |
| `start-session.html` | play | `Operate · Session` |
| `create-network-channel.html` | hash | `Network · Channel` |
| `edit-network-channel.html` | hash | `Network · Channel` |
| `create-bridge.html` | zap | `Catalog · Bridge` |
| `edit-bridge.html` | zap | `Catalog · Bridge` |
| `create-knowledge.html` | book | `Catalog · Knowledge` |
| `edit-knowledge.html` | book | `Catalog · Knowledge` |
| `create-mcp-server.html` | server | `System · MCP server` |
| `edit-mcp-server.html` | server | `System · MCP server` |
| `create-sandbox-profile.html` | shield | `System · Sandbox` |
| `edit-sandbox-profile.html` | shield | `System · Sandbox` |
| `create-vault-secret.html` | key | `System · Vault` |
| `add-workspace.html` | layers | `Workspace` |

Skip `create-provider-sheet.html` and `edit-provider-sheet.html` — they already carry `head-icon head-icon--neutral` + eyebrow; leave their headers untouched.

Related one-line fix in `modal-design-system.html`: the anatomy hero's eyebrow reads `Catalog · MCP server`; change it to `System · MCP server` so the taxonomy matches this table (MCP lives under System in the product sidebar).

### 4.2 CommandSelect chevrons (7 files)

Replace every

```html
<span class="command-select__chevron" aria-hidden="true">⌄</span>
```

with

```html
<span class="command-select__chevron" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="7 9 12 4 17 9"/><polyline points="7 15 12 20 17 15"/></svg></span>
```

Files: `create-agent.html`, `create-bridge.html`, `create-mcp-server.html`, `create-network-channel.html`, `edit-network-channel.html`, `start-session.html`, `add-workspace.html`.

### 4.3 Runtime trigger chevron cap (2 files)

In `create-agent.html` and `start-session.html`, the last runtime-trigger segment ends with a bare `⌄`:

```html
… data-component-trigger="runtime-selector">⌄</button>
```

Replace the `⌄` character with the `chevron-down` SVG (standard wrapper). Do not touch the button's attributes.

### 4.4 Runtime trigger meter (1 file) + reasoning meters (2 files)

`create-agent.html` has one glyph meter:

```html
<span class="runtime-trigger__meter" aria-hidden="true">▮▮▯▯▯▯</span><strong>Medium</strong>
```

Replace the span with the real meter component at the level matching the **label**, not the glyph count (the glyphs are wrong — 2 filled for Medium):

```html
<span class="im" data-level="4" aria-hidden="true"><i></i><i></i><i></i><i></i><i></i><i></i><i></i></span><strong>Medium</strong>
```

Level map (7 bars total, `data-level` = number filled): `none`=1, `minimal`=2, `low`=3, `medium`=4, `high`=5, `xhigh`=6, `max`=7. `default` never gets a meter — it gets the hollow ring (below).

Reasoning pressed groups (`create-agent.html` and `start-session.html` runtime popovers, `data-pressed-group`): each effort pill currently holds only its label. Insert the meter before the label using the same level map, e.g.:

```html
<button class="pill" type="button" aria-pressed="false" data-value="low"><span class="im" data-level="3" aria-hidden="true"><i></i><i></i><i></i><i></i><i></i><i></i><i></i></span>Low</button>
```

And the Default pill gains the ring:

```html
<button class="pill reasoning-default" type="button" aria-pressed="true" data-value="default"><span class="reasoning-ring" aria-hidden="true"></span>Default</button>
```

Keep `aria-pressed` states and `data-value`s exactly as found. The meter spans must contain exactly seven empty `<i></i>` elements — the fill is driven by `data-level` in CSS. Note: `modal-system.js` copies `button.textContent` into the trigger's reasoning label; the `<i>` elements contribute no text, so this stays safe — do not add text inside `.im`.

### 4.5 Popover search icon (every file with a `.component-popover__search`)

Each popover search row currently starts directly with `<input`. Insert the `search` SVG (standard wrapper, `aria-hidden="true"`) between the opening `<div class="component-popover__search">` and the `<input`. Apply in every popover of every file (grep for `component-popover__search`).

In the two runtime popovers (`create-agent.html`, `start-session.html`), also append the refresh button after the input, matching the reference:

```html
<button class="icon-button" type="button" aria-label="Refresh model catalog"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg></button>
```

### 4.6 Star glyphs (2 files)

- `favorite-toggle` buttons (`create-agent.html`, `start-session.html`): replace `<span aria-hidden="true">★</span>` with the `star` SVG in the standard wrapper (keep the sibling `<span data-favorite-copy>` untouched). CSS fills the star via `fill: currentColor` when pressed — the SVG must keep `fill="none"` in markup.
- `create-agent.html` provider rail: the radio `aria-label="Favorites"` contains a bare `★`; replace the character with the `star` SVG.

### 4.7 Chip remove glyphs (2 files + 1 JS line)

Static chips in `create-agent.html` and `create-network-channel.html` contain `<button type="button" aria-label="Remove …">×</button>`. Replace the `×` with the `x` SVG (standard wrapper; CSS sizes it to 10px).

Dynamic chips: `modal-system.js` line ~120 sets `remove.textContent = '×'`. Replace that single line with an SVG injection so dynamic chips match:

```js
remove.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
```

This is the only JS change permitted.

### 4.8 Popover option enrichment (structure existing text only)

Target anatomy (see any popover in `modal-design-system.html`):

```html
<button class="component-popover__option" type="button" role="option" aria-selected="…" data-value="…">
  <span class="opt-ic mono" aria-hidden="true">A</span>
  <span class="opt-copy"><strong>NAME</strong><span><b>PRIMARY META</b> · secondary meta</span></span>
  <span class="opt-check" aria-hidden="true"><svg …stroke-width="2.4"…>{check}</svg></span>
</button>
```

Rules:
- Split the existing option text on ` · `: first token → `<strong>`, second token → `<b>` inside the meta span, remaining tokens → plain meta text. Options with a single token get no meta line (omit the inner `<span>` entirely).
- `opt-ic` content: agents → `bot` SVG (drop the `mono` class when using an SVG); workspaces → mono first letter of the name; channels → `hash` SVG; models → mono first letter of the provider token; `Home workspace` → `home` SVG; `Agent default` → `bot` SVG; `None` options → no icon well at all (leave the option plain: just the text and an `opt-check`).
- Every option gains the trailing `opt-check` span. `data-value` and `aria-selected` stay byte-identical.
- Apply to all `.component-popover__option` buttons in: `create-agent.html`, `start-session.html`, `create-network-channel.html`, `edit-network-channel.html`, `add-workspace.html`, `create-mcp-server.html`, `create-bridge.html`.
- **Do not add data that is not already in the option text** (no ctx sizes, no effort counts, no paths).

### 4.9 Per-file change matrix

| file | 4.1 header | 4.2 chevron | 4.3 cap | 4.4 meters | 4.5 search | 4.6 star | 4.7 chips | 4.8 options |
|---|---|---|---|---|---|---|---|---|
| create-agent | ✓ | ✓ | ✓ | ✓ trigger + reasoning | ✓ + refresh | ✓ ×2 | ✓ | ✓ |
| start-session | ✓ | ✓ | ✓ | ✓ reasoning | ✓ + refresh | ✓ | — | ✓ |
| create-network-channel | ✓ | ✓ | — | — | ✓ | — | ✓ | ✓ |
| edit-network-channel | ✓ | ✓ | — | — | ✓ | — | — | ✓ |
| add-workspace | ✓ | ✓ | — | — | ✓ | — | — | ✓ |
| create-mcp-server | ✓ | ✓ | — | — | ✓ | — | — | ✓ |
| edit-mcp-server | ✓ | — | — | — | — | — | — | — |
| create-bridge | ✓ | ✓ | — | — | ✓ | — | — | ✓ |
| edit-bridge | ✓ | — | — | — | — | — | — | — |
| create-knowledge / edit-knowledge | ✓ | — | — | — | — | — | — | — |
| create-sandbox-profile / edit-sandbox-profile | ✓ | — | — | — | — | — | — | — |
| create-vault-secret | ✓ | — | — | — | — | — | — | — |
| create-provider-sheet / edit-provider-sheet | skip | — | — | — | — | — | — | — |
| index.html | no changes | | | | | | | |
| modal-design-system.html | only the §4.1 eyebrow taxonomy fix | | | | | | | |

## 5. Validation protocol (all steps required)

1. **Glyph sweep is clean**: from `modals/`, `grep -l '⌄\|★\|▮\|>×<' *.html` returns nothing (the `×` check specifically as `>×<` to avoid false positives).
2. **Header sweep**: `grep -c 'head-icon' *.html` → exactly 1 per modal surface (16 total across the 14 migrated files + 2 sheets), 0 in `index.html`.
3. **Structural checker**: run `node verify.mjs` (or `bun verify.mjs`) inside `modals/` — it validates tag balance, unique IDs, `aria-controls` references, and component-root markers across all surfaces. Zero failures. Read its output; do not assume.
4. **Behavior smoke** (per file with popovers): open the file, confirm (a) popovers open/close from their triggers, (b) selecting an option updates the trigger value, (c) multi-select count + chips stay in sync, (d) reasoning pill press updates the trigger label, (e) Escape closes the innermost layer and restores focus. All of this is `modal-system.js` behavior that only breaks if a hook attribute was lost.
5. **Visual captures**: render `create-agent.html`, `start-session.html`, and one edit modal (`edit-mcp-server.html`) at 1440×2000, plus `create-agent.html` at 390px wide. Save under `modals/evidence/visual/final/`. Compare headers, triggers, meters, and popovers against `modal-design-system.html` — structural mismatches are failures.
6. **Records**: update `modals/critique.json` (`pending_evidence` → move the per-modal migration item to `resolved_findings`, list new capture paths) and `agents/critique.json` scope line.

## 6. Acceptance criteria

- [ ] 14 modals carry the icon-well + eyebrow header per the §4.1 table; sheets untouched.
- [ ] Zero unicode UI glyphs (`⌄ ★ ▮ ×`) remain in any modal HTML; all replaced with the exact SVG library paths.
- [ ] All meters are 7-bar `.im` components with `data-level` per the label map; Default uses `reasoning-ring`.
- [ ] Every popover search row has the search icon; runtime popovers also have the refresh button.
- [ ] All popover options follow the `opt-ic` / `opt-copy` / `opt-check` anatomy with no invented data.
- [ ] No CSS changes; the single permitted JS change is the chip glyph line.
- [ ] All JS hooks intact (validated by §5.3 + §5.4).
- [ ] `verify.mjs` passes; glyph/header greps pass; captures saved; critique records updated.

## 7. Out of scope — do not touch

- `modal-system.css` (any value), `modal-system.js` beyond §4.7, `modal-design-system.html` beyond the §4.1 eyebrow fix.
- The reference trio `create-task-redesign.html` / `create-trigger-redesign.html` / `create-job-redesign.html` (`modals/`) — historical references, never edited.
- Field inventories, copy, section structure, Simple/Advanced wiring, and `data-od-id`s of the 16 modals.
- Any change to `packages/ui` or `web/src` — this is a static-artifact migration only.
