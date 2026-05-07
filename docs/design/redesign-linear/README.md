# Redesign Linear — exploratory variations

Three Linear-flavored re-skins of the AGH operator surface, applied to the **Tasks** and **Jobs** lists. None of them touch `packages/ui/`, `web/`, or `DESIGN.md`. They translate Linear's editorial calm into the existing AGH warm-dark palette and ask: _what would AGH look like if our visual grammar leaned more product-marketing-clean and less control-panel-tech?_

Open `index.html` for the gallery. Open any variation directly to inspect.

## Why

Today's `web/` is operator-first and dense, faithful to `DESIGN.md`. That fidelity is good — but the visual chrome (mono uppercase eyebrows everywhere, 15%-tinted status pills, solid-accent active fills, decorated wire-protocol chips) reads as "developer tool" before it reads as "AGH". This exploration previews the same content with a calmer, more editorial register, anchored on:

- Linear's **surface ladder** (hierarchy by lightness, not borders + shadows).
- Linear's **single-voice typography** (Inter from masthead to body, mono only for raw identifiers).
- Linear's **scarce accent** discipline. Lavender lives on Linear's brand mark, primary CTA, and focus — and that's it. We do the same with `#E8572A`.
- Linear's **iconographic status taxonomy** (open ring, half-fill, full-disc, X) instead of colored dots.
- Linear's **3-bar priority signal** instead of the words "high priority".
- Linear's **assignee avatars on every row** — agents get squircles, humans get circles, system events get a workspace glyph.
- Linear's **text-led filters** (V3) and **pill-toggle segmented controls** (V1, V2), replacing tint-led active states.
- Linear's **rich detail inspector** — properties strip → sub-issues with progress → activity feed with avatars → comment composer.

What stays immovable from `DESIGN.md`:

- Warm gray ramp (`#0E0E0F → #141312 → #1E1C1B → #2E2C2B`).
- Accent `#E8572A` and the four semantic tones (`#30D158`, `#FF453A`, `#FFD60A`, `#BF5AF2`).
- Inter Variable + JetBrains Mono. **No** Playfair Display, **no** NuixyberNext — these are operator surfaces, not marketing or wordmark contexts.
- Flat depth model. Box-shadow appears only as sub-pixel inset highlights to communicate panel lift; never as drop shadow or glow.
- Hairline `#3C3A39` divider.

## The three variations

### V1 — `linear-calm`

Direct port. Operator density preserved (`px-4 py-3` rows). Surface ladder of 3 levels (canvas-deep → canvas → surface) carries hierarchy. The shifts are:

- **Section labels** become Inter 500 12px (no mono uppercase). Reads as "calm header" not "metadata eyebrow".
- **Filter pills** become Linear pill-toggles: track on `--color-surface-panel`, active segment on `--color-surface-elevated`. **No accent fill.**
- **List headline sticks** while scrolling — the active scope is always visible.
- **Active list rows** swap surface bg + a 2px inset accent rail flush with the panel edge.
- **Status indicators** are iconographic (ring family). No tinted pill backgrounds. Caption is Mono 10px uppercase only when the label is essential.
- **Priority** is a 3-bar signal aligned to a fixed 14px slot.
- **Avatars** on every row + every activity event.
- **Inspector** opens with a `PropertyPill` strip (Status / Priority / Owner-with-avatar / Parent / Approval / Attempts / Children) replacing the legacy 110px-label KV grid.
- **Sub-issues** render with a thin progress bar and per-row status icon + assignee avatar.
- **Activity feed** rebuilt as `avatar | message | detail | mono-time`.
- **Empty state** is a real card: 32px outline icon, secondary headline, tertiary helper, and a kbd legend (`↑↓ navigate · ⏎ open · ⌘K jump`).

### V2 — `linear-panel`

Linear's "product card" idiom. The entire workspace sits inside a lifted surface-1 frame (16px radius, hairline border, 24px gutter on canvas). The frame's lift is real: a top-inner sub-pixel white highlight, a soft radial vignette behind it, and an outer 1px hairline ground it as the focal element. Inside, sidebar / list / detail are independent surface-2 sub-panels (10px radius each) with their own header gradient overlay so each panel reads as a card with structure.

- **Outer frame** has `inset 0 1px 0 rgba(255,255,255,0.045)` plus a 1px outer hairline.
- **Sub-panel headers** carry a `linear-gradient(180deg, rgba(255,255,255,0.022), transparent)` overlay so the head visibly lifts above the body.
- **Page H1 stays small** (22px). The hero moment is the frame, not the masthead.
- **Sidebar section labels** use the DESIGN.md `SidebarSectionLabel` spec verbatim (mono 10px 600, 0.14em tracking, uppercase, label color).
- **Active nav row** swaps to canvas-deep bg with a 1px inset highlight.
- **Selected list rows** tint accent at 4% + a 2px rail. Justified by the extra elevation of the surface-2 panel underneath.
- **Inspector ships** the full set: properties strip → sub-issues with progress (`completed / total`) → activity feed → comment composer with `⌘⏎` hint → actions cluster.

### V3 — `linear-editorial`

Maximum reduction. Hierarchy becomes 100% typographic — surface ladder + type weight + tracking carry every distinction.

- **Masthead is heroic**: Inter 700 `clamp(44px, 5.4vw, 76px)`, tracking `-0.05em`, line-height 0.94. The page title sets the page tone, not its filters.
- **Eyebrow is mono uppercase folio** (`Workspace · agh-runtime · Updated 2m ago`).
- **Sub-lead** is a situation report, not marketing — `12 open across 3 workspaces, 1 blocked, 1 failed in the last hour.`
- **Sidebar has no icons in nav rows.** `Tasks 12` is enough. Active row carries a 2px accent tick on the gutter outside the row, not a fill.
- **Filter bar is text-led**: `All · Mine · Watched` separated by middots. Active filter has a 4px accent dot prefix and bold weight. **No underline.**
- **Tabular row grid** — `24px / 1fr / 132px / 88px` — so status and time columns line up across every row.
- **Errors lift to pull-quotes**. Failed-row errors render as `<figure class="le-quote le-quote--danger">` with a 2px danger-color border-left, mono 12px body, and a `attempt 3/3 · tsk-XXX` source line. Blocked reasons render as a warning quote.
- **Properties become a hairline grid** (2-column, 1px gap on divider, mono uppercase labels, sans values).
- **Activity feed** is `avatar | message + detail | mono-time` with a hairline divider between events at 50% divider opacity.
- **Empty state is an editorial colophon**, left-aligned: mono eyebrow `THE RUNTIME`, then a 22px Inter 500 sentence with mid-italic emphasis (`Every contract is a task. Every task is replayable.`), then a mono uppercase footer line (`agh-runtime · v0.41.0 · uptime 3h 22m`).

## Diff against `DESIGN.md` today

| Aspect | Today | V1 calm | V2 panel | V3 editorial |
|---|---|---|---|---|
| Section labels | Mono 11px UPPERCASE 0.06em | Inter 500 12px | Mono 10px UPPERCASE 0.14em | Mono 10px UPPERCASE 0.16em |
| Status indicator | 8px dot + 15% tint pill | Ring icon family (no tint) | Ring icon family (no tint) | Ring icon family (right-aligned) |
| Priority | Text "High priority" pill | 3-bar glyph (14px slot) | 3-bar glyph (14px slot) | 3-bar glyph + text inline |
| Filter pills active | Solid accent fill | Surface-elevated | Surface-elevated | Underline-free; accent dot prefix |
| Active list row | 3px accent rail + surface bg | 2px inset rail + surface bg | 2px rail + 4% accent wash | Surface wash + Inter 500 weight |
| Page masthead | Inter 700 20px | Inter 600 22px | Inter 600 22px | Inter 700 44–76px (-0.05em) |
| Density | Operator dense | Same as today | +25% breathing room | Editorial sparse |
| Sidebar nav | Icon + label + count + rail | Same as today (no rail) | Same as today (no rail) | **Label + count only**, accent tick |
| Avatars | Absent in `web/` | On every row + event | On every row + event | On every row + event |
| Detail panel | KV grid | Properties strip + activity | Properties + sub-issues + activity + composer | Hairline property grid + colophon empty |
| Errors | Inline danger text | Border-left 2px danger box | Border-left 2px danger box | Pull-quote `<figure>` |
| Variant pin | n/a | Bottom-right ghost pill | Bottom-right ghost pill | Bottom-right ghost pill |

## Running locally

The project is plain HTML + React via CDN + Babel standalone — no build step.

```sh
cd docs/design/redesign-linear
python3 server.py -p 8002
```

Then open <http://127.0.0.1:8002/> for the gallery, or jump directly to:

- <http://127.0.0.1:8002/v1-linear-calm/>
- <http://127.0.0.1:8002/v2-linear-panel/>
- <http://127.0.0.1:8002/v3-linear-editorial/>

Each variation has a quiet pin at the bottom-right that switches between V1/V2/V3. Inside any variation, the sidebar `Tasks` / `Jobs` rows act as the in-page view switcher.

## File structure

```
docs/design/redesign-linear/
├── README.md                       This file
├── index.html                      Gallery: 3 cards linking to V1/V2/V3
├── server.py                       Python no-cache static server
├── shared/
│   ├── fonts.css                   Inter Variable + JetBrains Mono
│   ├── tokens-base.css             AGH base tokens (mirrored from packages/ui)
│   ├── data.jsx                    Mock TASKS + JOBS + ACTIVITY + helpers
│   └── icons.jsx                   Lucide-equivalent SVGs + StatusIcon + PriorityIcon
├── v1-linear-calm/
│   ├── index.html
│   ├── styles.css
│   └── app.jsx
├── v2-linear-panel/
│   ├── index.html
│   ├── styles.css
│   └── app.jsx
└── v3-linear-editorial/
    ├── index.html
    ├── styles.css
    └── app.jsx
```

## Fidelity checklist

- All colors come from `--color-*` tokens declared in `shared/tokens-base.css`. Hex literals appear only inside that file plus `#ffffff` on the primary CTA per DESIGN.md, and fixed accent ink (`#17110F`) inside the workspace tile when it sits on accent.
- Only Inter Variable + JetBrains Mono are loaded. Playfair and NuixyberNext appear nowhere.
- Zero `box-shadow` outside (a) sub-pixel inset highlights communicating panel lift, (b) the 1.5px accent ring on the active workspace tile, (c) the 2px inset accent rail on selected rows.
- Zero gradients on content. The two gradient usages (V2 frame top-inner highlight and sub-panel header overlay) are atmospheric depth, not decoration; the V2 canvas vignette is a single `radial-gradient` at 2.2% opacity.
- Status coverage in `data.jsx`: `running`, `in_progress`, `ready`, `failed`, `blocked`, `pending`, `draft`, `completed`, `canceled` — full 9-state coverage.
- Job coverage: `enabled`/`disabled` × `dynamic`/`manual` × `workspace`/`global`.
- Per-task activity feed seeded for the 5 most-likely-selected tasks; default fallback for the rest.
- Sub-issues seeded for tasks with `childCount > 0`; render with progress bar.
- Status taxonomy reads as semantic — `running=accent` is the only orange, `in_progress=info` (purple), `blocked=warning`, `failed=danger`, `completed=success`, `ready/pending/draft/canceled=neutral`.
- Pulse animation appears on `running` only (one row at a time). Not on `in_progress`.
- Avatars are deterministic: same owner.label always renders the same hue. Agents render as squircles (5–6px radius), humans as full circles, system events as a workspace-glyph squircle.
- Variant pin is a quiet bottom-right segmented control on `surface` ground. No backdrop blur. No accent fill.
