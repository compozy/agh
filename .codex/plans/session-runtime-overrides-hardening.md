# Session Runtime Overrides Hardening

## Plan Metadata

- Status: Accepted for implementation.
- Accepted after review of the session-create modal and the backend provider/model/reasoning override work.
- Execution state: Not implemented by this plan artifact. Use this file as the handoff source before editing code.
- Primary surfaces: Go session runtime, provider config/settings, OpenAPI/codegen, web session modal, web settings providers, CLI docs, runtime docs.

## Summary

- Keep provider/model/reasoning work in the same PR, but finish missing backend, config, settings, docs, and test coverage so the feature is not UI-only.
- Targeted checks already pass as-is: `go test ./internal/config ./internal/api/core ./internal/session ./internal/cli -count=1` and the session dialog/hook Vitest files. The remaining work is correctness/completeness, not an obvious compile failure.
- Leave unrelated dirty files untouched. Use `rtk` for shell commands.

## Public APIs / Interfaces

- Extend session read payloads, not just create payloads:
  - Add `model,omitempty` and `reasoning_effort,omitempty` to `SessionPayload`.
  - Populate both from `session.Info` for create/get/list/resume responses and generated TypeScript types.
- Complete provider configuration surfaces:
  - `config.toml` provider overlays support `supported_models = [...]` and `supports_reasoning_effort = true|false`.
  - `SettingsProviderSettingsPayload` accepts/returns `supported_models?: string[]` and `supports_reasoning_effort?: boolean`, preserving explicit `false` in requests.
  - `agh config set providers.<name>.supported_models ...` and `providers.<name>.supports_reasoning_effort ...` are supported and classified as daemon-restart provider mutations.
- Keep `model` as a free-form runtime override. `supported_models` is a suggestion list for UI/agents, not a backend allowlist.

## Implementation Changes

- Backend/session:
  - Move the reasoning-effort enum and transport-agnostic validation into `internal/session`; expose one canonical ordered list: `minimal`, `low`, `medium`, `high`, `xhigh`.
  - Add a session-domain validation error such as `ErrInvalidRuntimeOverride`, map it to HTTP/UDS `400`, and use it for invalid enum/provider-runtime combinations.
  - Enforce provider capability after provider resolution: if `reasoning_effort` is set and the resolved provider does not support reasoning effort, reject creation before launching the ACP child.
  - Persist runtime selections: add `ReasoningEffort` to `store.SessionMeta`, `session.Info`, `Session.Meta()`, `sessionInfoFromMeta`, and `Session.Info()`.
  - Fix resume semantics: `prepareResumeStart` must carry both `meta.Model` and `meta.ReasoningEffort` into the start spec so resumed sessions keep the selected model and `AGH_REASONING_EFFORT`.
- Config/settings:
  - Make `supports_reasoning_effort` tri-state at the config layer, using a pointer/effective helper pattern so built-in `true` providers can be explicitly disabled with `false`.
  - Add provider overlay fields for `supported_models` and `supports_reasoning_effort`; merge, clone, and settings conversion must preserve explicit values.
  - Normalize `supported_models` by trimming, deduping in order, and rejecting blank entries during config validation.
  - Update settings provider read/write conversion, `providerSettingsMap`, and settings tests so HTTP settings and CLI config can manage the same fields agents see in the session modal.
- Web:
  - Keep the command-popover interaction for Provider, Model, and Reasoning.
  - Improve the create-session modal layout: widen to `sm:max-w-xl`, keep Agent/Provider full-width, place Model/Reasoning in a responsive two-column row on desktop and one column on mobile, and tighten field descriptions.
  - Normalize the model custom item to ASCII quotes and keep custom model entry available even when `supported_models` is empty.
  - Keep Reasoning disabled unless the selected provider has `supports_reasoning_effort === true`; submit should omit it when disabled/default.
  - Update settings provider UI so provider cards/editor show and edit supported models plus the reasoning-effort toggle. Use a newline-delimited textarea for models, parse to an ordered deduped array, and submit an empty array when the operator clears the field.

## Tests And Verification

- Backend tests:
  - Config overlay tests for `supported_models`, explicit `supports_reasoning_effort = false` disabling a built-in provider, and `true` enabling a custom provider.
  - Settings API/service tests for list/get/put round-tripping both fields, including explicit `false` and empty model arrays.
  - Session creation tests for invalid reasoning enum, model/reasoning without provider, and reasoning on an unsupported provider returning `400`.
  - Session manager tests proving model + reasoning are stored in meta, exposed in `Info`, included in `SessionPayload`, injected/unset as `AGH_REASONING_EFFORT`, and preserved across resume.
  - CLI tests proving `agh session new --model ... --reasoning-effort ...` forwards exact request fields.
- Web tests:
  - `use-session-create-dialog` tests for selecting model/reasoning, submit payload inclusion/omission, provider/agent change reset, and unsupported-provider reasoning clearing.
  - `SessionCreateDialog` tests for model selection, custom model entry, reasoning selection, disabled reasoning state, and updated modal width/layout expectations.
  - Settings provider page tests for editing/saving supported models and reasoning support.
- Codegen/docs/gates:
  - Run `make codegen` after contract changes and `make codegen-check` before final verification.
  - Run `make cli-docs` after CLI/config surface changes.
  - Update site docs for `agh session new --model/--reasoning-effort`, provider config keys, and provider model/reasoning behavior.
  - Run targeted checks first: `go test ./internal/config ./internal/settings ./internal/api/contract ./internal/api/core ./internal/session ./internal/cli -count=1`, the affected Vitest files, `make bun-typecheck`, and site build/typecheck if docs imports change.
  - Final blocking gate: `make verify`.

## Assumptions

- No SQLite migration is required because these runtime selections live in session metadata JSON, not a SQL schema column.
- No backward-compatibility bridge is needed; AGH is still greenfield alpha.
- `supported_models` is advisory and should not block custom model IDs in CLI/API/web.
- `reasoning_effort` is provider-gated because it changes ACP child launch environment and must not rely only on frontend disabling.
