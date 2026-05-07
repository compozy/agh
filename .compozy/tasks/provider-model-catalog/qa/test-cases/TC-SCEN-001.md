# TC-SCEN-001: Operator Edits Provider Catalog and Starts a Session

**Priority:** P0
**Type:** Real Scenario
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-05-07

---

## Behavioral Scenario Charter

- **Startup situation**: Operator runs an isolated AGH lab (unique `AGH_HOME`, ports, tmux socket) provisioned by `agh-qa-bootstrap`. At least one ACP-capable provider is configured with synthetic credentials; live discovery uses stub HTTP servers and fake subprocesses by default.
- **Operator intent**: Adjust the curated metadata for one provider, refresh its catalog, then start a new session against a chosen model with reasoning effort.
- **Expected business outcome**: Operator perceives a coherent catalog with deterministic source attribution; the new session creates against the chosen model and reasoning effort; selection persists across surfaces.
- **AGH surfaces used**: Web (Settings > Providers, new session dialog), HTTP (`/api/providers/...`), SQLite (`model_catalog_*` tables), ACP (`session/new`, `session/set_config_option`).
- **Real provider/LLM expectation**: ACP fake driver acts as the provider unless `MODELCATALOG_LIVE=1` is set; in that case one ACP-backed provider must produce a real `session/new` response.
- **Blocked live-provider boundary**: Default run uses fake ACP; `MODELCATALOG_LIVE=1` annex documents the real-provider boundary in the verification report.
- **Scenario contract minimums covered**: operator role, web channel, HTTP channel, SQLite truth, ACP control, manual entry, stale fallback observation.

## Actors and Agent Roles

| Actor / Agent | Role | Expected Behavior | Evidence Source |
|---------------|------|-------------------|-----------------|
| Operator | Catalog editor + session creator | Edits curated metadata, refreshes catalog, starts session | Web screenshots + DOM snapshot |
| ACP fake driver | Provider | Returns `configOptions` for model + reasoning | ACP fixture transcript |
| Daemon | Catalog authority | Persists curated edit, refreshes, projects to surfaces | SQLite + HTTP responses |

## Preconditions

- [ ] Bootstrap manifest exists; `AGH_WEB_API_PROXY_TARGET` exported.
- [ ] Daemon running; web app reachable; ACP fake driver registered.
- [ ] Catalog seeded with `models.dev` + `builtin` rows for `codex`.

## Journey Steps

1. **Operator opens Settings > Providers.**
   - Surface: Web.
   - Input: Browser navigation to `/settings/providers` via `browser-use:browser` (fallback `agent-browser`).
   - **Expected:** Provider cards render with source status; redacted `last_error` shown for any failed source; `default_model`/`supported_models`/`supports_reasoning_effort` strings absent in DOM and React Query cache.
2. **Operator adds a curated entry with reasoning efforts.**
   - Surface: Web form.
   - Input: `id="manual-gpt"`, `display_name="Manual GPT"`, `reasoning_efforts=["medium","high"]`, `default_reasoning_effort="medium"`.
   - **Expected:** PUT request matches generated TS contract; daemon persists; SQLite `model_catalog_rows` has a `config` row at priority 120 with snapshot-preserved metadata; CLI / HTTP / UDS / Host API agree (TC-INT-003).
3. **Operator refreshes catalog.**
   - Surface: Web refresh button.
   - Input: Click refresh on `codex` card.
   - **Expected:** UI shows pending state; on completion `last_refresh_at` updates; if a stub source is failing, `stale=true` flag visible with redacted error.
4. **Operator opens new session dialog and selects manual model.**
   - Surface: Web dialog.
   - Input: select provider `codex`, model `manual-gpt`, reasoning `medium`.
   - **Expected:** Dialog renders catalog rows from `useProviderModels`; manual entry valid; submission triggers `session/new` and `session/set_config_option` (TC-FUNC-010 invariant).
5. **Operator confirms session is live with chosen model.**
   - Surface: Web active session panel.
   - **Expected:** Session controls switch to ACP `configOptions`; chosen model + reasoning effort reflected; catalog metadata never overrides current option value (SI-7).
6. **Disruption probe - stale catalog while session lives.**
   - Probe: stub `models.dev` 5xx; trigger refresh.
   - **Expected:** Catalog rows flagged stale; running session unaffected; manual model selection still valid.

## Required Evidence

- Browser screenshots: settings page, dialog, active session controls.
- HTTP request/response logs (network panel exports).
- SQLite snapshots (rows + status) before and after edit.
- ACP fake driver transcript showing `session/new` + `session/set_config_option`.
- Daemon log capture with `refresh_request_id` correlation.

## Audit Coverage

- C4 (operator), C5 (Web + HTTP), C8 (cross-surface), C10 (artifact reuse: catalog row reused by TC-SCEN-002), C11 (stale probe), C14.

## Pass Criteria

- Operator goal achieved end-to-end without manual workaround.
- Catalog row visible across CLI/HTTP/UDS/Host API.
- ACP control matches TC-FUNC-010 invariants.
- Stale probe surfaces redacted error; session unaffected.

## Failure Criteria

- Settings form emits legacy fields.
- ACP control regresses to `session/set_model` despite advertised config option.
- Stale state hidden or session aborted.
