# AGH Modal Standard

Living authority for the modal library under `docs/design/opendesign/modals/`.

**Authority chain (higher wins):**

1. `packages/ui/src/tokens.css`, `DESIGN.md`, `packages/ui/src/index.ts`, and production Web selectors/`Dialog` shells in `web/src/systems`
2. `docs/design/opendesign/design-system/` — especially `patterns.html` § Modals and `GUIDE.md`
3. This file + the 16 runnable surfaces in this folder (`index.html`, `modal-system.css` / `modal-system.js`, `verify.mjs`)

Historical Tier-1 artboards (`create-{task,trigger,job}-redesign.html`) stay as research references; production Task/Job/Trigger editors are the live Tier-1.

## Frame

- Register: Product.
- Scene: an operator configures a runtime entity on a dark desktop control surface, scanning dense but calm forms while retaining a complete mobile path.
- Dials: visual variance 2, motion intensity 2, information density 8.
- Static artifacts may transcribe daemon-backed controls and existing catalog integrations. They must not invent runtime behavior.

## Hosts and shell

| Host | Width | Use |
| --- | ---: | --- |
| Compact dialog | `--width-modal-sm` / 560px | Short, focused task |
| Standard dialog | `--width-modal-md` / 720px | Default configuration flow |
| Wide dialog | `--width-modal-lg` / 880px | Dense policy/catalog composition |
| Extra-wide host | `--width-modal-xl` / 1180px | Workspace-scale task that cannot preserve comprehension at 880px |
| Wizard host | `--height-modal-wizard` / 960px max | Ordered, validated steps for one irreversible task |
| Provider sheet | 576px | Persistent provider settings |
| ConfirmDialog | `--width-modal-sm` / 560px | Destructive or irreversible confirmation |

- Use `--color-overlay-scrim`, `--overlay-blur`, `--shadow-overlay`, and `--radius-lg` exactly.
- Header pattern (production `task-editor-modal.tsx`): a 36px icon well (`--radius-icon-well`, `--color-accent-tint`, inset `--color-accent-dim` ring, 16px glyph) beside an accent-strong `Eyebrow`, a 15px/500 `DialogTitle` (`--tracking-tight`), and a 13px muted description. Entity editor modals and provider sheets use it; ConfirmDialog uses a neutral well or a plain title. The icon well is the only accent-tinted surface in the shell. The body is the only scroll owner. The footer stays stable and contains one primary action.
- Control geometry (production components): buttons 22/26/30px at `--radius-md` and 12px/500 labels; inputs, selects, and CommandSelect triggers 36px on `--color-elevated`; RuntimeSelector trigger 34px with a `--color-line-strong` border; PillGroup segments 24px in a 2px track; switch 18×32 with a `--color-fg-strong` thumb.
- Simple shows the common path. Advanced is the only disclosure tier and never hides required configuration.
- At 760px and below, dialogs become full-width bottom surfaces; fields and selection grids stack and interactive targets reach 44px.
- Use a route or sheet instead of a modal for browse-heavy, long-lived, multi-entity, or cross-navigation work.

## Component grammar

- `RuntimeSelector`: provider, model, and reasoning are one segmented control. Provider filters remain a radiogroup (active item carries a 2px accent bar). Reasoning is a `role="group"` of `aria-pressed` buttons: Default is separate from `none`, `minimal`, `low`, `medium`, `high`, `xhigh` (displayed as Extra high), and `max`; effort intensity renders as the production 7-bar meter (2.5px bars, 3→12px, `--color-accent-strong` fill), never as glyph characters. Inside the popup the selected model row uses `--color-accent-tint` with an accent-strong check, and the pressed effort uses `--color-accent-tint-strong` — this runtime-catalog accent selection is production behavior and does not extend to RadioCard. The selected model's favorite toggle is a separate button keyed by `provider:model` (pressed state is `--color-warning`); never nest it in an option. Do not rebuild the provider/model/reasoning trio.
- `AgentCommandSelect` / `AgentCommandMultiSelect`: searchable agent catalogs with provider and category metadata. Native agent `<select>` is forbidden.
- `ScopeSelector`: PillGroup plus `WorkspaceCommandSelect` when workspace scope is active.
- `WorkspaceCommandSelect`: searchable workspace identity, including home behavior where supported.
- `DirectoryBrowser`: filesystem root picker transcribing onboarding `directory-browser.tsx` — home/up toolbar with mono path and "Use this folder", hover-reveal row pick, reading/empty states, and a selected-root summary row feeding an immutable root field. Never replace it with a plain path input for root selection.
- `CommandSelect`: searchable catalog composition for tools, toolsets, sandboxes, channels, coordinators, and other existing catalogs.
- `NativeSelect`: fixed enums with at most seven stable choices.
- `RadioCard`: two to four consequence-bearing choices. Selected state uses `--color-row-selected`, `--color-line-strong`, and `--shadow-highlight`, never an accent border or fill.
- `SettingsFieldRow`: provider-sheet label, description, hint, error, and control association.
- `SecretField`: plaintext enters once; edit surfaces expose presence and rotation only.
- `ImmutableIdentity`: readable summary for fields the update contract cannot mutate. Do not use disabled inputs as a data display.
- `FormSection`, `SettingsFieldRow`, and `Alert` markers belong on each rendered component root (`.sec`, `.settings-row`, and `.notice`/`.note` respectively), never on a wrapper that merely contains instances.

## Interaction and accessibility

- Dialogs are named, modal, focus-trapped, Escape-dismissible, and restore focus to their trigger.
- Every field has a visible programmatic label. Errors are associated through `aria-describedby` and focus moves to the first invalid field.
- Command selectors use a button trigger with `aria-controls`, a searchable popup, and listbox semantics. RuntimeSelector uses a dialog with a combobox, provider-filter radiogroup, model listbox, separate favorite toggle, and reasoning pressed group.
- Escape dismisses the innermost popup before the dialog. ArrowDown from search enters the first listbox option. Provider-filter radios support arrow keys plus Home/End.
- `PillGroup` follows the production `role="group"` + `aria-pressed` contract. Do not add roving-arrow behavior that production does not implement.
- All interactions expose a distinct 2px focus-visible indicator. Text contrast is at least 4.5:1 and non-text/focus indicators at least 3:1 against rendered surfaces.
- Motion uses `--duration-*` and `--ease-*`; reduced motion removes transforms and collapses durations.

## State and copy

- Catalog states: loading, refreshing, stale, empty, no match, error, and current. Preserve drafts and disable only dependent controls.
- Form states: default, focus, disabled, validation error, saving, save error, and saved. Save errors retain every entered value.
- Secrets: absent, present, editing, invalid, saving, and rotated.
- Actions use verb + object. Errors state what happened and how to recover. No welcome copy, hype, exclamation marks, or unsupported claims.

## Stable artifact vocabulary

Use `data-od-component` with: `dialog`, `sheet`, `runtime-selector`, `agent-select`, `agent-multi-select`, `scope-selector`, `workspace-select`, `command-select`, `native-select`, `radio-card`, `settings-field-row`, `secret-field`, `immutable-identity`, `alert`, `form-section`, and `dialog-footer`.

## Forbidden drift

- Parallel token aliases (`--surface`, `--fg`, `--line`, `--accent`, and similar).
- Inline styles or per-page scripts.
- Gradients, accent side rails, decorative shadows, accent-selected cards, and numbered editorial sections.
- Native agent selectors, `.agent-field`, or separate provider/model/reasoning controls.
- RuntimeSelector inside provider settings; provider sheets configure a provider rather than choose a runtime.
