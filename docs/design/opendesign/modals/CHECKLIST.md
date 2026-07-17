# Modal Migration Checklist

## Runtime truth

- [ ] Every control maps to a daemon field or an existing catalog endpoint.
- [ ] Immutable update fields render as `ImmutableIdentity`, not disabled inputs.
- [ ] Provider settings never contain RuntimeSelector.
- [ ] Secrets are write-only and expose only presence/reference after save.

## Reuse gate

- [ ] Provider + model + reasoning uses RuntimeSelector.
- [ ] Agent selection uses AgentCommandSelect or AgentCommandMultiSelect.
- [ ] Scope uses ScopeSelector and WorkspaceCommandSelect.
- [ ] Existing catalogs use CommandSelect composition.
- [ ] Fixed enums use NativeSelect; consequence-bearing choices use RadioCard.
- [ ] Provider sheets use SettingsFieldRow.

## Shell and responsive behavior

- [ ] Host is 560, 720, 880, 1180px, a 576px provider sheet, or the 960px-tall wizard envelope for the documented task class.
- [ ] Dialog has one accessible name, one body scroll owner, and one primary action.
- [ ] Simple/Advanced has at most one disclosure tier.
- [ ] 360px stacks grids, preserves copy, and provides 44px targets.
- [ ] 200% zoom retains all controls without body-level horizontal scrolling.

## States and accessibility

- [ ] Default, hover, active, focus-visible, disabled, loading, empty, error, selected, saving, and save-error states are designed where applicable.
- [ ] Focus enters the modal, remains trapped, closes with Escape, and returns to the opener.
- [ ] Combobox/listbox/radiogroup semantics match behavior; reasoning uses `role="group"` + `aria-pressed`, with Default separate from seven efforts.
- [ ] Favorite is a separate `aria-pressed` button keyed by `provider:model`, never nested inside a model option.
- [ ] Escape closes the innermost popup; ArrowDown enters the first result; provider radio rails support Home/End.
- [ ] Every `.sec`, `.settings-row`, and `.notice`/`.note` carries the marker for the real FormSection, SettingsFieldRow, or Alert instance.
- [ ] Labels and errors are programmatically associated.
- [ ] Rendered contrast meets WCAG 2.2 AA and color never carries meaning alone.
- [ ] Reduced motion preserves state clarity without transforms.

## Drift scan

- [ ] No inline `style` attributes or page-local scripts.
- [ ] No parallel token aliases, raw production palette, gradients, accent side rails, or decorative shadows.
- [ ] No native agent selectors, `.agent-field`, separate provider/model/reasoning fields, or numbered editorial sections.
- [ ] Every surface declares stable `data-od-component` markers owned by `verify.mjs`.

## Evidence

- [x] `rtk node verify.mjs` passes.
- [x] `rtk node --check modal-system.js` passes.
- [ ] All 16 surfaces have 360, 768, and 1440px evidence; dense dialogs and sheets add 1920px.
- [ ] Runtime, Agent, Agent Multi, Scope, Workspace, and Command selector mocks have matched production-story pairs at 1100×700.
- [ ] Every visual-contract bundle contains source identity, reference, implementation, side-by-side, diff, comparison JSON, and a passing review.
- [ ] Owned processes are stopped and teardown evidence is clean.

## Markup migration (MIGRATION-HANDOFF.md)

- [x] 14 modals carry icon-well + eyebrow; provider sheets untouched.
- [x] Zero unicode UI glyphs (`⌄ ★ ▮ ×`) in modal HTML.
- [x] Meters are 7-bar `.im` with `data-level`; Default uses `reasoning-ring`.
- [x] Popover search rows have search SVG; runtime popovers have refresh.
- [x] Popover options use `opt-ic` / `opt-copy` / `opt-check` with no invented data.
- [x] No CSS changes; only permitted JS change is createChip SVG.
- [x] `verify.mjs` PASS after header/CSS contract update.
