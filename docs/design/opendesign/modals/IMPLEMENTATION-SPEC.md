# Modal System — Production Implementation Spec

Everything required to land the 16 designed entity editors in `web/`.

- **Design source of truth:** the 16 surfaces in this folder + `MODAL-STANDARD.md` + `../design-system/patterns.html` § Modals.
- **Runtime source of truth:** `internal/api/contract/*` (verified 2026-07-24 against the current tree — the line numbers in `analysis/02_analysis_runtime-contracts.md` are stale; `contract.go` was split into per-domain files).
- **Token source of truth:** `packages/ui/src/tokens.css` → generated `DESIGN.md`.

Authority on conflict: runtime contract > tokens/`packages/ui` primitives > this spec > the static HTML surfaces. Where a designed control has no write contract it is cut, not implemented (§5.2).

---

## 1. Current state — audit

Every target dialog was re-read on 2026-07-24. The production tree has drifted from `analysis/01`: Vault and Sandbox gained *inspect* sheets (their editors still run through `SettingsEditorDialog`), Provider moved from a sheet to `provider-detail-dialog.tsx`, and the Task editor was split into `task-editor-modal.tsx` + `task-editor-surface.tsx` and **lost its header entirely**.

| # | Surface | Production file | Host today | Header today | Mode switch | Primary gap |
|---|---|---|---|---|---|---|
| R1 | Task editor (reference) | `web/src/systems/tasks/components/task-editor-surface.tsx:101` | `--width-modal-md` ✓ | **none** — description is a `<p>` in the body (`:134`) | ✓ `ModeToolbar` | Header regression: no icon well / eyebrow / `DialogTitle` |
| R2 | Job / Trigger editor (reference) | `web/src/systems/automation/components/automation-editor-dialog.tsx:136` | `--width-modal-xl` ✓ | ✓ **canonical** icon well + `Eyebrow` + title + description | preview rail | Header is local — promote it |
| 1 | Create agent | `web/src/systems/agent/components/agent-create-dialog.tsx:117` | `--width-modal-lg` ✓ | plain `<header>` (`:124`) | ✗ 4-step wizard (`:63`) | Wizard → Simple/Advanced; header; MCP field is untruthful |
| 2 | Start session | `web/src/systems/session/components/session-create-dialog.tsx:145` | `--width-modal-sm` ✓ | ruled, bare (`:151`) | ✗ | Header; Simple/Advanced; overrides section |
| 3 | Add workspace | `web/src/systems/workspace/components/workspace-setup.tsx:171` | `max-w-xl` ✗ | ruled, bare (`:177`) | n/a | Split shell + `DirectoryBrowser`; keep the global-default card |
| 4 | Create knowledge | `web/src/systems/knowledge/components/knowledge-create-dialog.tsx:118` | `sm:max-w-2xl` ✗ | ruled, bare (`:123`) | n/a | Host token; header |
| 5 | Edit knowledge | `web/src/systems/knowledge/components/knowledge-edit-dialog.tsx:76` | `sm:max-w-2xl` ✗ | ruled, bare (`:82`) | n/a | Host token; header; `ImmutableIdentity` |
| 6 | Create channel | `web/src/systems/network/components/network-create-channel-dialog.tsx:61` | `sm:max-w-120` ✗ | **not ruled** (`:65`) | n/a | Host token; ruled header; fanout `RadioCard` |
| 7 | Edit channel | `web/src/systems/network/components/shell/channel-policy-dialog.tsx:157` | inherits | bare | n/a | Header; locked identity; fanout `RadioCard` |
| 8 | Create bridge | `web/src/systems/bridges/components/bridge-create-dialog.tsx:201` | `--width-modal-lg` ✓ | plain `<header>` (`:212`) | ✗ 3-step wizard (`:32`) | Wizard → Simple/Advanced; header; secret slots |
| 9 | Edit bridge | `web/src/systems/bridges/components/bridge-edit-dialog.tsx:62` | `sm:max-w-3xl` ✗ | ruled, bare (`:67`) | ✗ | Host token; header; rotation; locked identity |
| 10 | Add MCP server | `web/src/systems/settings/components/mcp-server-editor.tsx:92` | `max-w-xl`/`sm:max-w-3xl` ✗ | ruled + muted eyebrow (`:99`) | ✗ | Host token; accent eyebrow + icon well; Simple/Advanced |
| 11 | Edit MCP server | same file (`mode="edit"`) | same | same | ✗ | + `ImmutableIdentity` for name/scope |
| 12 | Add vault secret | `web/src/systems/vault/routes/vault-page.tsx:238` (via `SettingsEditorDialog`) | shell default ✗ | shell header, bare | n/a | Host token; header; `SecretField` |
| 13 | Create sandbox profile | `web/src/systems/sandbox/routes/sandbox-page.tsx:252` (via `SettingsEditorDialog`) | shell default ✗ | shell header, bare | ✗ | Header; Simple/Advanced; secret env rotation |
| 14 | Edit sandbox profile | same file (`mode="edit"`) | same | same | ✗ | + `ImmutableIdentity` |
| 15 | Create provider | `web/src/systems/settings/components/provider-detail-dialog.tsx` | dialog + `LaneTabs` | `Eyebrow` + `DialogTitle`, no icon well | ✗ | §14 D1 (sheet vs dialog); auth-ownership cards; slot gating |
| 16 | Edit provider | same file (`mode="edit"`) | same | same | ✗ | + rotation-only credentials |

**Shared shell:** `web/src/systems/settings/components/settings-editor-dialog.tsx:36` backs #12–#14. Upgrading it lifts three surfaces at once, and it is consumed by `settings-dialogs.stories.tsx`, so its story pins the contract.

Systemic findings:

1. **The designed header exists in exactly one place** — `automation-editor-dialog.tsx:136-160`. 15 of 16 surfaces need it; the Task reference needs it back.
2. **Six surfaces use ad-hoc Tailwind widths** instead of `--width-modal-*` (`sm:max-w-2xl`, `sm:max-w-3xl`, `sm:max-w-120`, `max-w-xl`).
3. **Simple/Advanced exists only in tasks** — `web/src/systems/tasks/components/task-form/mode-toolbar.tsx`, coupled to `TaskFormMode` and the task scope/workspace props.
4. **No shared secret or immutable-identity primitive.** The only secret control is `web/src/systems/marketplace/components/mcp-secret-field.tsx` (marketplace-local).
5. **Two step wizards remain** (agent 4-step, bridge 3-step) with per-step `canAdvance`/`stepValidity` gating that must survive the flattening.

---

## 2. Foundation — build before any surface migrates

Phase 0. Nothing in §4 starts until these land with stories + tests.

### F1 · `EntityDialogHeader` → `packages/ui`

Lift `EditorHeader` (`automation-editor-dialog.tsx:144-166`) verbatim into `packages/ui/src/components/custom/entity-dialog-header.tsx`, export from `primitives.ts`/`index.ts`.

```
<DialogHeader variant="ruled">
  icon well 36px · rounded-icon-well · bg-accent-tint · text-accent-strong · ring-1 ring-accent-dim ring-inset · 16px glyph
  Eyebrow (text-accent-strong) · DialogTitle (mt-1) · DialogDescription (mt-1)
  [optional] close button — Button variant="ghost" size="icon-sm"
</DialogHeader>
```

Props: `{ icon: LucideIcon; eyebrow: string; title: string; description?: ReactNode; onClose?: () => void }`. The icon well is the **only** accent-tinted surface in the shell (`MODAL-STANDARD.md` § Hosts). `ConfirmDialog` keeps its neutral well and is out of scope.

`automation-editor-dialog.tsx` must be refactored onto it in the same change (no duplicate header definition survives).

### F2 · `EntityDialogFooter`

Two footers exist today and neither is the modal contract: `DialogFooter variant="ruled"` (`dialog.tsx:109`) has the right chrome but no hint slot, and `EditorFooter` (`custom/editor-footer.tsx`) has `meta`/`secondary`/`primary` slots but page-editor geometry (`sticky bottom-0`, `bg-canvas`, `min-h-editor-footer`, `px-4 py-2.5`, `text-form-label text-muted`). Add the modal footer as a `DialogFooter` composition — do not restyle `EditorFooter`, which page and sheet editors depend on.

```
footer ruled · flex · gap-3 · px-6 py-4 · border-t border-line
  [hint]  flex-1 · text-form-hint text-subtle · leading info glyph   (optional)
  [Cancel] Button variant="outline"
  [Primary] Button min-w-32 · Spinner while saving · verb+object label
```

One primary action per dialog. Label switches by mode (`Create agent` → `Save changes`).

### F3 · Host size contract

Add a `dialogShellClass(size)` helper (`web/src/components/` or `packages/ui`) emitting the token classes, and route every migrated dialog through it. No raw `max-w-*` on an entity editor after Phase 3.

| Size | Token | Surfaces |
|---|---|---|
| `sm` | `--width-modal-sm` 560 | start session, create/edit knowledge, create/edit channel, add vault secret |
| `md` | `--width-modal-md` 720 | task (R1), add/edit MCP server, create/edit sandbox profile, edit bridge |
| `lg` | `--width-modal-lg` 880 | create agent, create bridge |
| `xl` | `--width-modal-xl` 1180 | job/trigger (R2), add workspace (split) |
| `sheet` | 576 | provider — pending §14 D1 |

Height caps stay `min(var(--height-modal-*), calc(100vh - 2rem))`; the body is the only scroll owner (`grid-rows-[auto_auto_minmax(0,1fr)_auto]`).

### F4 · `ModeToolbar` → generic

Generalize `web/src/systems/tasks/components/task-form/mode-toolbar.tsx` into a domain-agnostic component (`web/src/components/entity-mode-toolbar.tsx`):

```ts
{ mode: "simple" | "advanced";
  onModeChange(mode): void;
  scope?: ScopeSelectorProps;      // optional trailing ScopeSelector + WorkspaceCommandSelect
  label?: string }
```

Contract from `MODAL-STANDARD.md`: `PillGroup` `role="group"` + `aria-pressed`, **one disclosure tier only**, Advanced never hides a required field, leaving Advanced snaps unsupported advanced-only selections back to a Simple-valid default (the task template snap-back at `task-editor-surface.tsx:75-85` is the reference).

Consumers: agent create, bridge create/edit, MCP create/edit, sandbox create/edit, start session, provider.

### F5 · `SecretField`

New shared component. Generalize `web/src/systems/marketplace/components/mcp-secret-field.tsx`; delete the marketplace copy and re-point `mcp-install-dialog.tsx`.

States (normative — `STATE-MATRIX.md`): `absent` · `present` · `editing` · `invalid` · `saving` · `rotated`.

- Create → single write input, `type="password"`, no value ever read back.
- Edit → `•••• present · [Replace]`; pressing Replace opens the write input; Cancel restores `present` and preserves the existing binding.
- Never prefill from a GET. Every AGH secret read path returns presence/ref/hash only (`internal/api/contract/vault.go:12-19`, `settings_collection_payloads.go:64-69`, `bridges.go:83-91`).
- Emits the preservation signal each contract needs: MCP `preserve_secrets` (`settings_collection_payloads.go:116-119`), provider `secrets[]` write list, bridge secret-binding PUT.

### F6 · `ImmutableIdentity`

Readable summary row for create-only fields. **Disabled inputs are forbidden as data display** (`MODAL-STANDARD.md` § Component grammar).

Applies to: bridge platform / extension / scope, channel name + members, workspace `root_dir`, knowledge name+type, MCP name + scope, sandbox profile name, provider name, vault ref.

### F7 · Split-dialog shell

Two-pane body for `add-workspace` only: `--width-modal-xl`, `grid-template-columns: minmax(0,1fr) 420px`, right pane on `--color-canvas` with a left hairline, each pane its own scroll owner, collapsing to one column below 980px. Ship as a `variant="split"` on the entity dialog body, not a bespoke layout.

### F8 · Reuse register (build nothing new here)

| Need | Use | Location |
|---|---|---|
| provider+model+reasoning | `RuntimeSelector` | `@/systems/runtime` |
| agent pick / multi-pick | `AgentCommandSelect`, `AgentCommandMultiSelect` | `@/systems/agent` |
| scope + workspace | `ScopeSelector`, `WorkspaceCommandSelect` | `@/systems/workspace` |
| filesystem root | `DirectoryBrowser` | `@/systems/onboarding` |
| catalog pick | `CommandSelect` | `@agh/ui` |
| fixed enum ≤7 | `NativeSelect` | `@agh/ui` |
| consequence choice 2–4 | `RadioCard` | `@agh/ui` — selected = `bg-surface-glaze` + `line-strong` rim, **never accent** |
| section frame | `FormSection`, `NumberedSection` | `@agh/ui`, `tasks/task-form` |
| settings row | `FieldRow` / `SettingsFieldRow` | `@agh/ui`, `@/systems/settings` |
| notices | `Alert` | `@agh/ui` |
| effort meter | `IntensityMeter` (7 bars) | `@agh/ui` |

Shadowing any of these in `web/` is a blocked lint error (`compozy-ui-reuse/no-shadow-ui-primitive`).

---

## 3. Shell contract (applies to every surface in §4)

```
Dialog (base-ui, modal, focus-trapped, Escape-dismissible, focus returns to opener)
└ DialogContent unframed · dialogShellClass(size) · showCloseButton={false}
  ├ EntityDialogHeader           (F1)
  ├ EntityModeToolbar            (F4 — only where Simple/Advanced exists)
  ├ body  min-h-0 flex-1 overflow-y-auto px-6 py-5   ← sole scroll owner
  │   └ FormSection / NumberedSection blocks
  └ EntityDialogFooter           (F2)
```

Geometry (production components, non-negotiable): buttons 22/26/30px at `--radius-md`, 12px/500 labels · inputs / selects / `CommandSelect` triggers 36px on `--color-elevated` · `RuntimeSelector` trigger 34px with `--color-line-strong` border · `PillGroup` segments 24px in a 2px track · switch 18×32 with `--color-fg-strong` thumb · scrim `--color-overlay-scrim` + `--overlay-blur` + `--shadow-overlay` + `--radius-lg`.

---

## 4. Per-surface migration

Each entry: designed source → production target, host, header copy, body, and the delta to implement. Field → contract bindings are consolidated in §5.

### 4.1 Create agent — `create-agent.html`

**Target:** `agent-create-dialog.tsx` (+ `agent-create-runtime-step.tsx`, `agent-create-access-step.tsx`). **Host:** `lg` 880.
**Header:** robot glyph · `Operate · Agent` · *Create agent* · "An agent is a reusable **definition** — instructions plus a runtime. Sessions launch from it and inherit its access policy."

- Delete the 4-step wizard (`WIZARD_STEPS`, stepper nav, Back/Continue, `Step N of M`) and the plain `<header>`.
- Mode toolbar: Simple/Advanced + `ScopeSelector` (+ `WorkspaceCommandSelect` when `scope=workspace`).
- **Simple:** `01 The definition` (name, prompt) · `02 Runtime` (`RuntimeSelector`) · `03 What can it do on its own?` (3 `RadioCard`s → `permissions`, with an `Alert` translating the pick into the contract value).
- **Advanced adds:** `04 Runtime details` (command, category path) · `05 Tools & skills` (tools, toolsets, deny tools, disabled skills).
- Per-step `canAdvance` gating becomes field-level validation + a footer submit gate. Name and prompt stay required; nothing else blocks submit.
- **Cut:** the MCP servers multi-select (§5.2 T1).
- Edit parity: `agents.$name.settings.tsx` remains the edit surface; this dialog stays create-only (§14 D3).

### 4.2 Start session — `start-session.html`

**Target:** `session-create-dialog.tsx`. **Host:** `sm` 560.
**Header:** play glyph · `Operate · Session` · *Start session* · launcher framing — a session is a run, not a stored record, so there is no edit twin.

- **Simple:** `Who runs, and where` — `AgentCommandSelect`, `WorkspaceCommandSelect`, session name.
- **Advanced:** `Launch overrides` — `RuntimeSelector` override, network channel, working path.
- Keep the existing `CatalogStatusLine` states (loading / refreshing / stale / error / empty) inside the Runtime field — a Simple view that hides catalog truth is a regression. It stays a status line: **no `Catalog current · Refresh` banner in the dialog chrome**; catalog refresh lives only inside the `RuntimeSelector` popover.

### 4.3 Add workspace — `add-workspace.html`

**Target:** `workspace-setup.tsx` (`WorkspaceSetupDialog`). **Host:** `xl` 1180, **split** (F7).
**Header:** layers glyph · `Workspace` · *Add workspace*.

- Left pane `Location`: `DirectoryBrowser` (home/up toolbar, mono path, "Use this folder", hover-reveal pick, reading/empty states) → selected-root summary card → display name with folder-name autofill. A plain path input is forbidden for root selection.
- Right pane `Session defaults`: default agent (`AgentCommandSelect`), sandbox profile (`CommandSelect`), additional directories.
- **Preserve the production "register the global default workspace" card** (`workspace-setup.tsx:58-89`) — the design does not cover it; keep it as a left-pane shortcut above Location.
- `variant="onboarding"` (`WorkspaceOnboarding`) is out of scope and must keep rendering from the same `WorkspaceSetupContent`.

### 4.4 / 4.5 Knowledge create + edit — `create-knowledge.html`, `edit-knowledge.html`

**Targets:** `knowledge-create-dialog.tsx`, `knowledge-edit-dialog.tsx`. **Host:** `sm` 560 (replaces `sm:max-w-2xl`).
**Header:** book glyph · `Catalog · Knowledge` · *Create knowledge entry* / *Edit knowledge entry*.

- Create: `What kind of knowledge?` (`RadioCard` grid, already correct at `:143`) → `The content` (name, description, content).
- Edit: `ImmutableIdentity` for name + type (retrieval stability), then description + content only.
- Both dialogs hold state locally (`useState`/`useReducer`); lift to the route-owned draft shape so validation and dirty-state gating match the other editors.

### 4.6 / 4.7 Network channel create + edit — `create-network-channel.html`, `edit-network-channel.html`

**Targets:** `network-create-channel-dialog.tsx`, `shell/channel-policy-dialog.tsx`. **Host:** `sm` 560.
**Header:** graph glyph · `Network · Channel` · *Create channel* / *Edit channel*.

- Create: `The channel` (name, purpose) · `Members` (`AgentCommandMultiSelect`) · `Who receives a message?` (fanout `RadioCard`s + coordinator `CommandSelect` revealed only for `coordinator`).
- Edit: name and members render as `ImmutableIdentity` — `UpdateNetworkChannelRequest` cannot mutate them (`network_payloads.go:93-98`). Purpose + fanout + coordinator stay editable.
- Replace the non-ruled `<DialogHeader>` at `:65` with F1.

### 4.8 Create bridge — `create-bridge.html`

**Target:** `bridge-create-dialog.tsx`. **Host:** `lg` 880.
**Header:** link glyph · `Catalog · Bridge` · *Create bridge*.

- Delete the 3-step wizard; per-step `stepValidity` (`:170`) becomes field validation.
- Mode toolbar: Simple/Advanced + `ScopeSelector`.
- **Simple:** `Where does it connect?` (provider `RadioCard`s from `GET /api/bridges/providers`) · `Identity` (display name) · `Credentials` — one `SecretField` per provider-declared `secret_slots`, rendered dynamically.
- **Advanced:** `Routing & delivery` (DM policy, delivery mode, routing policy, provider config).
- Secrets go through secret bindings only. Never into `provider_config` — the contract rejects secret-shaped JSON there (`ErrUnsafeBridgeProviderConfigPayload`, `bridges.go`).

### 4.9 Edit bridge — `edit-bridge.html`

**Target:** `bridge-edit-dialog.tsx`. **Host:** `md` 720 (replaces `sm:max-w-3xl`).

- Platform / extension / scope → `ImmutableIdentity` (`UpdateBridgeRequest` omits them, `bridges.go:71-81`).
- Credentials → `SecretField` in rotate mode.
- Keep the `?delivery=error` path: the delivery test lives in `bridge-test-delivery-dialog.tsx`; either fold it into an Advanced "Test delivery" action inside this dialog or keep it as a launched sub-dialog — do not render two competing entry points.

### 4.10 / 4.11 MCP server add + edit — `create-mcp-server.html`, `edit-mcp-server.html`

**Target:** `mcp-server-editor.tsx`. **Host:** `md` 720.
**Header:** connector glyph · `System · MCP server` · *Add MCP server* / *Edit MCP server*. The eyebrow becomes accent-strong (it is muted today at `:100`).

- **Simple:** `How does it run?` — transport `RadioCard` (local / remote) revealing command or endpoint URL, plus server name.
- **Advanced:** `Process environment` (args, env, secret env) · `Remote authentication` (OAuth method, issuer, client id, token URL, scopes, client secret) — remote transport only.
- Edit: name + scope → `ImmutableIdentity`; secrets rotate-only and emit `preserve_secrets` / `preserve_env` for untouched bindings.

### 4.12 Add vault secret — `create-vault-secret.html`

**Target:** `vault-page.tsx:238` via `SettingsEditorDialog`. **Host:** `sm` 560.
**Header:** lock glyph · `System · Vault` · *Add vault secret*.

- `The reference` (ref, kind) · `The value` (`SecretField`, write-only).
- Overwrite of an existing ref requires explicit confirmation copy in an `Alert` — `PUT` is an upsert (`vault.go:32-36`).
- Replacement of an existing secret continues to run through `vault-secret-sheet.tsx` (inspect + replace); this dialog is create-only.

### 4.13 / 4.14 Sandbox profile create + edit — `create-sandbox-profile.html`, `edit-sandbox-profile.html`

**Target:** `sandbox-page.tsx:252` via `SettingsEditorDialog`. **Host:** `md` 720.
**Header:** cube glyph · `System · Sandbox` · *Create sandbox profile* / *Edit sandbox profile*.

- **Simple:** `The profile` (name) · `Where does it run?` (backend `RadioCard`: local / Daytona).
- **Advanced:** `Isolation & lifecycle` (sync mode, persistence, runtime root, env, secret env) · `Network policy` (allow/deny lists) · `Daytona workspace` (api url, target, image, snapshot, class, auto stop, auto archive) — Daytona backend only.
- Edit: profile name → `ImmutableIdentity`; secret env → `SecretField` rotation.
- `sandbox-profile-sheet.tsx` stays the inspect surface and must keep launching this editor via `onRequestEdit`.

### 4.15 / 4.16 Provider create + edit — `create-provider-sheet.html`, `edit-provider-sheet.html`

**Target:** `provider-detail-dialog.tsx` + `provider-edit-form*.tsx`. **Host:** pending §14 D1.
**Header:** gear glyph · `Settings · Provider` · *Create provider* / *Edit provider*.

- `Provider basics` (name, display name, command) · `Who owns authentication?` (`RadioCard`: `native_cli` vs `bound_secret`) · `Runtime & models` (harness, runtime provider, transport, base URL, default model, curated models, env policy, home policy).
- **Auth gate is a security boundary:** credential slots and secret writes render **only** for `auth_mode = bound_secret`. Under `native_cli` the provider owns login via `auth_login_command` and AGH must not offer credential inputs (`internal/CLAUDE.md` § Provider auth boundary).
- Rows use `SettingsFieldRow` (17 in create, 11 in edit).
- Edit: name → `ImmutableIdentity`; credential values rotate-only.
- **RuntimeSelector is forbidden here** — a provider sheet configures a provider, it does not choose a runtime.

### 4.17 Reference surfaces (regression fixes, same wave)

- **R1 Task** (`task-editor-surface.tsx`): re-apply `EntityDialogHeader` (clipboard glyph · `Autonomy · Task` · *Create task* / *Edit task* · the `TASK_DESCRIPTION` string at `:48`) and delete the in-body description `<p>` at `:134`. Without this the reference no longer matches the standard it defines.
- **R2 Job/Trigger** (`automation-editor-dialog.tsx`): refactor `EditorHeader` onto F1; behavior unchanged.

---

## 5. Runtime-truth register

### 5.1 Field → contract

Verified against the current `internal/api/contract` tree.

| Surface | Contract | Fields |
|---|---|---|
| Create agent | `CreateAgentRequest` / `CreateAgentPayload` — `agent_observe_payloads.go:21,28` | `scope`, `workspace`, `name`, `provider`, `command`, `model`, `reasoning_effort`, `tools[]`, `toolsets[]`, `deny_tools[]`, `permissions`, `category_path[]`, `skills.disabled[]`, `prompt` |
| Edit agent | `UpdateAgentRequest` — `agent_definitions.go:14` | same payload + **`expected_digest`** (409 conflict path required) |
| Start session | `CreateSessionRequest` — `session_runtime_payloads.go:18` | `agent_name`, `provider`, `model`, `reasoning_effort`, `name`, `workspace`, `workspace_path`, `network_participation` |
| Add workspace | `CreateWorkspaceRequest` — `workspace_payloads.go:6` | `root_dir`, `name`, `add_dirs[]`, `default_agent`, `sandbox_ref` |
| Knowledge create | `MemoryCreateRequest` — `memory.go:118` (`POST /api/memory`) | `scope`, `workspace_id`, `agent_name`, `type`, `name`, `description`, `content` |
| Knowledge edit | `MemoryEditRequest` — `memory.go:136` (`PATCH /api/memory/{filename}`) | `description`, `content` (+ scope keys) |
| Channel create | `CreateNetworkChannelRequest` — `network_payloads.go:83` | `channel`, `workspace_id`, `purpose`, `fanout_policy`, `coordinator_peer_id`, `agent_names[]` |
| Channel edit | `UpdateNetworkChannelRequest` — `network_payloads.go:93` | `purpose`, `fanout_policy`, `coordinator_peer_id` **only** |
| Bridge create | `CreateBridgeRequest` — `bridges.go:13` | `scope`, `workspace_id`, `platform`, `extension_name`, `display_name`, `enabled`, `dm_policy`, `routing_policy`, `provider_config`, `delivery_defaults`, `notification_suppress` |
| Bridge edit | `UpdateBridgeRequest` — `bridges.go:71` | `display_name`, `dm_policy`, `routing_policy`, `provider_config`, `delivery_defaults`, `notification_suppress` |
| Bridge secret | `PutBridgeSecretBindingRequest` — `bridges.go:83` | `secret_ref`, `kind`, `secret_value` (write-only) |
| MCP server | `PutSettingsMCPServerRequest` — `settings_mutations.go:39`; payload `settings_collection_payloads.go:87` | `name`, `transport`, `command`, `args[]`, `env{}`, `secret_env{}`, `url`, `auth{}` + `secret_values`, `preserve_secrets`, `preserve_env` |
| Sandbox profile | `PutSettingsSandboxRequest` — `settings_mutations.go:46`; payload `:181` | `backend`, `sync_mode`, `persistence`, `runtime_root`, `env{}`, `secret_env{}`, `network{}`, `daytona{}` |
| Provider | `PutSettingsProviderRequest` — `settings_mutations.go:33`; payload `:9` | `command`, `display_name`, `models`, `harness`, `runtime_provider`, `transport`, `base_url`, `auth_mode`, `env_policy`, `home_policy`, `auth_status_command`, `auth_login_command`, `credential_slots[]` + `model_curation`, `secrets[]` |
| Vault secret | `PutVaultSecretRequest` — `vault.go:32` | `ref`, `kind`, `secret_value` (write-only) |

Enums (never invent): permissions `{deny-all, approve-reads, approve-all}` · reasoning `{none, minimal, low, medium, high, xhigh, max}` + provider default · fanout `{capability_match, coordinator, all_members}` · scope `{global, workspace}` with `workspace_id` required iff `scope=workspace` and forbidden otherwise.

### 5.2 Designed controls with no write contract — cut or gate

| ID | Control | Finding | Action |
|---|---|---|---|
| **T1** | `create-agent.html` → *MCP servers* multi-select | `CreateAgentPayload` (`agent_observe_payloads.go:28-40`) has **no** `mcp_servers` field. `AgentPayload.MCPServers` (`agent_definitions.go:89`) is read-only, populated on the response path in `internal/api/core/agent_conversions.go:37-72`. There is no authoring route. | Remove the control from the implementation **and** from `create-agent.html`. Re-introducing it requires a contract change first (`agh-contract-codegen-coship`). |
| **T2** | `create-agent.html` → *Category path* as a `/`-joined string | Contract is `category_path []string`. | Keep the single mono input; split/join on `/` at the adapter boundary and document it. |
| **T3** | Provider credential slots under `native_cli` | Security boundary, not a layout choice. | Render slots only when `auth_mode = bound_secret`; under `native_cli` show the login command as read-only guidance. |

Nothing else in the 16 surfaces renders a field the daemon does not accept.

---

## 6. States

`STATE-MATRIX.md` is normative. Every listed state needs a visible treatment; a state without one is an incomplete surface.

- **Dialog:** open, closing, dismissed — named, inert background, trapped focus, deterministic focus return.
- **Form:** pristine, dirty, invalid, saving, save-error, saved. Duplicate submit blocked. **Edit submit is enabled only when dirty** (PATCH contracts reject empty patches via `HasChanges()`). A save error retains every entered value.
- **Catalog** (`RuntimeSelector`, `CommandSelect`): closed, open, searching, loading, refreshing, stale, empty, no-match, error, disabled, read-only, no-model. Drafts survive every catalog state; only dependent controls disable.
- **Secret:** absent, present, editing, invalid, saving, rotated.
- **Optimistic concurrency:** agent `expected_digest` must round-trip and surface a 409 as a "this changed elsewhere — reload" recovery path, not a silent failure.

Deterministic review routes already supported by `modal-system.js`: `?save=error`, `?auth=error`, `?delivery=error`. Mirror them as Storybook story args.

---

## 7. Accessibility

- Named, modal, focus-trapped dialogs; Escape dismisses the innermost popup before the dialog; focus returns to the trigger.
- Every field has a visible programmatic label; errors associate via `aria-describedby` and focus moves to the first invalid field on submit.
- `CommandSelect`: button trigger with `aria-controls`, searchable popup, listbox semantics. ArrowDown from search enters the first option.
- `RuntimeSelector`: dialog + combobox + provider-filter radiogroup (arrow keys + Home/End) + model listbox + **separate** favorite toggle keyed by `provider:model` + reasoning `role="group"` with `aria-pressed`, Default separate from the seven efforts, meter rendered by `IntensityMeter` (7 bars — never glyph characters).
- `PillGroup` uses production `role="group"` + `aria-pressed`; do not add roving-arrow behavior production lacks.
- Distinct 2px `focus-visible` indicator on every interactive element. Text contrast ≥ 4.5:1, non-text/focus ≥ 3:1. Color never carries meaning alone.
- Motion uses `--duration-*` / `--ease-*`; reduced motion drops transforms and collapses durations without losing state clarity.

---

## 8. Responsive

- ≤ 760px: dialogs become full-width bottom surfaces; field grids and selection grids stack; interactive targets reach 44px.
- ≤ 980px: the split shell (F7) collapses to one column with the side pane below.
- 360px must preserve all copy; 200% zoom must retain every control without body-level horizontal scrolling.

---

## 9. Testing

Test placement first (`consolidate-test-suites`): name the invariant, owning layer, and canonical suite; extend the existing suite rather than adding standalone regressions.

| Layer | Owns | Suite |
|---|---|---|
| `packages/ui` unit | F1/F2/F5/F6 render + state contracts | `packages/ui/src/components/custom/__tests__/` |
| Web component | per-surface field wiring, Simple/Advanced disclosure, immutable rendering, submit gating | each system's existing `components/__tests__/` |
| Web adapter | request shape per §5.1, `category_path` split/join (T2), preserve-secret signals | each system's `adapters/__tests__/` |
| Storybook | one story per surface × {create, edit, saving, save-error, secret-present} | each system's `components/stories/` — `settings-dialogs.stories.tsx` already pins the shared shell |
| E2E | agent create, bridge create + secret binding, vault write | existing Playwright specs |

Forbidden: CSS-literal, snapshot, prose, or file-existence tests. Gates: `bunx turbo run lint typecheck test --filter=./web` and `--filter=./packages/ui` during iteration; `make verify` once at completion.

---

## 10. Visual validation

`VISUAL-VALIDATION.md` is the contract. Implementation-only screenshots are **not** parity evidence.

- All 16 surfaces at 360×800, 768×900, 1440×900; agent create, bridge create, and the provider surface add 1920×1080.
- Matched reference/implementation pairs at 1100×700 for `RuntimeSelector`, `AgentCommandSelect`, `AgentCommandMultiSelect`, `ScopeSelector`, `WorkspaceCommandSelect`, `CommandSelect`.
- `RuntimeSelector` additionally captures open, stale, no-model, disabled, empty, and error, including the separate favorite toggle and the Default + seven-effort footer.
- Capture through `agh-ui-screenshot` against Storybook (resolve ids with `list-stories.mjs`). Every bundle carries source identity, reference, implementation, side-by-side, diff, comparison JSON, and a passing review with zero unresolved structural mismatches.
- Teardown is mandatory on every terminal path (`eval "$TEARDOWN_COMMAND"` / `make qa-reap`); cite `teardown.json` with `"clean": true`.

---

## 11. AGH Impact Audit

- **Native tools:** no impact — no `agh__*` tool id, toolset, descriptor, schema digest, risk flag, or capability gate changes. Surfaces checked: `internal/tools`, tool descriptors, capability gates.
- **Extensibility and hooks:** bridge create reads the extension-declared provider catalog (`GET /api/bridges/providers` → `secret_slots`, `config_schema`) and MCP add/edit reads the settings MCP collection — both are consumption of existing extension surfaces, no registry/bundle/hook/config-lifecycle change. `config.toml` keys unchanged.
- **Workspace data isolation:** every scoped surface (agent, bridge, channel, MCP, sandbox, provider, session, workspace) propagates `scope` + `workspace_id` on the existing contracts; the invariant "workspace_id required iff scope=workspace" is enforced in the form, and list/read/cache/SSE paths are untouched by this UI wave. No new cross-workspace read.
- **Official AGH skill:** no impact — no public tool id, CLI path, hook event, capability, bundle, or memory/network/task semantic changes. `skills/agh/` unchanged.

**QA tracker:** user-visible behavior changes across 16 surfaces. Add `untested` content-addressed scenario files under `docs/qa/scenarios/` for the new Simple/Advanced flows and the secret rotate/replace path; reset `qa_status` to `untested` on existing scenarios covering agent create, bridge create, MCP add, vault write, workspace add, session start, knowledge create/edit, and channel create/edit. Flag, don't retest.

---

## 12. Phasing

| Phase | Content | Exit gate |
|---|---|---|
| **0 · Foundation** | F1–F7 + reuse register; refactor R2 onto F1 | `packages/ui` stories + tests green; `bunx turbo run lint typecheck test --filter=./packages/ui` |
| **1 · Reference repair** | R1 task header (§4.17) | Task modal matches the standard it defines; capture pair |
| **2 · Shared shell** | `SettingsEditorDialog` upgraded (F1+F2+F3) → lifts vault, sandbox ×2 | Existing `settings-dialogs.stories.tsx` + tests green |
| **3 · Low-risk surfaces** | knowledge ×2, channel ×2, vault, start session | Host tokens applied; no `max-w-*` left on these |
| **4 · Dense surfaces** | agent create (wizard→Simple/Advanced, T1 cut), bridge create/edit, MCP ×2, sandbox ×2 | Step validation preserved as field validation; no submit regression |
| **5 · Provider** | after §14 D1 is answered | Auth gate (T3) proven by test |
| **6 · Close** | full capture matrix, QA scenario flags, `make verify` | Visual bundles reviewed; `make verify` clean; teardown evidence |

Two-touch rule: if any surface needs a third patch in this wave, it becomes a new TechSpec rather than a third patch.

---

## 13. Delete targets

Greenfield alpha — hard cuts, no compat shims.

1. `WIZARD_STEPS`, stepper nav, Back/Continue, and `Step N of M` in `agent-create-dialog.tsx` and `bridge-create-dialog.tsx`.
2. The local `EditorHeader` in `automation-editor-dialog.tsx:144-166` (replaced by F1).
3. The in-body description `<p>` in `task-editor-surface.tsx:134` (moves into the header).
4. All ad-hoc modal widths: `sm:max-w-2xl` ×2, `sm:max-w-3xl` ×2, `sm:max-w-120`, `max-w-xl` ×2.
5. `web/src/systems/marketplace/components/mcp-secret-field.tsx` (absorbed by F5).
6. The MCP servers multi-select in `create-agent.html` (T1) — design file edited in the same change.
7. Any disabled input used as read-only data display (replaced by F6).

---

## 14. Open decisions

| ID | Decision | Options | Recommendation |
|---|---|---|---|
| **D1** | Provider host | (a) 576px sheet per the design; (b) keep `provider-detail-dialog.tsx` (tri-mode inspect/edit/create with `LaneTabs`) | **(b)** — production moved to the dialog after the design was drawn, and it owns inspect+edit+create in one surface. Apply the sheet's *body grammar* (`SettingsFieldRow` rows, auth-ownership `RadioCard`s, write-only slots) and F1; update `MODAL-STANDARD.md` § Hosts to match. |
| **D2** | Bridge delivery test | fold into edit-bridge Advanced vs keep `bridge-test-delivery-dialog.tsx` | Fold in — one entry point; delete the standalone dialog. |
| **D3** | Agent edit symmetry | create is a modal, edit is a route (`agents.$name.settings.tsx`) | Keep the asymmetry this wave; the route carries more than a modal should. Revisit separately. |
| **D4** | `NumberedSection` vs `FormSection` | task uses mono ordinals; `FormSection` is a card with no ordinal slot | Add an optional ordinal to `FormSection` and standardize; otherwise the two grammars keep diverging. |
| **D5** | T1 MCP-on-agent | cut permanently vs add `mcp_servers` to `CreateAgentPayload` | Cut now; if the capability is wanted, it is a contract change with CLI/UDS parity, not a UI change. |
