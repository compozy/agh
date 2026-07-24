# Modal State Matrix

The matrix is normative for the static playgrounds and future production implementation. A listed state without a visible treatment is incomplete.

| Component | Required states | Visible contract |
| --- | --- | --- |
| Dialog / sheet | open, closing, dismissed | Named surface, inert background, trapped focus, deterministic focus return |
| Simple / Advanced | simple, advanced | One selected mode; required fields never disappear |
| RuntimeSelector | closed, open, searching, loading, refreshing, stale, empty, no-match, error, disabled, read-only, no-model | Segmented value remains legible; popup reports catalog state; draft survives |
| CommandSelect | closed, open, searching, loading, empty, no-match, error, selected, disabled | Trigger retains selected identity; results use listbox semantics |
| AgentCommandMultiSelect | closed, open, searching, selected-none, selected-many, no-match, disabled | Trigger exposes count; options expose selected state without closing after every choice |
| ScopeSelector | global, workspace, disabled | Workspace selector appears only for workspace scope |
| RadioCard | default, hover, focus-visible, selected, disabled | Selected check plus neutral glaze/rim; no accent-only state |
| NativeSelect | default, focus-visible, selected, disabled, invalid | Native semantics, canonical control geometry, associated error |
| SecretField | absent, present, editing, invalid, saving, rotated | Plaintext never appears after save; rotation clears the temporary input |
| Form | pristine, dirty, invalid, saving, save-error, saved | Duplicate submit blocked; edit submit enabled only when dirty; failures retain draft |

## Deterministic review routes

Open a living surface HTML (not `index.html`) and append query flags supported by `modal-system.js`:

| Flag | Effect | Typical host |
| --- | --- | --- |
| `?save=error` | Submit fails; draft retained | Any dialog with a primary save |
| `?auth=error` | Auth/credential failure path | Provider sheets, bridge/MCP secret flows |
| `?delivery=error` | Delivery test failure | `edit-bridge.html` |

RuntimeSelector / CommandSelect catalog states (loading, stale, empty, error, no-model, disabled) are exercised in production Storybook pairs and by interacting with the living popovers on `create-agent.html` and `start-session.html`. Capture sizing: 1100×700 for matched primitive pairs; modal surfaces at 360, 768, and 1440px (dense dialogs and provider sheets add 1920px). See `VISUAL-VALIDATION.md`.
