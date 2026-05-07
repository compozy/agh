# TC-SCEN-002: Agent Manages Catalog Through CLI/HTTP/UDS/Host API

**Priority:** P0
**Type:** Real Scenario
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-05-07

---

## Behavioral Scenario Charter

- **Startup situation**: Same isolated lab as TC-SCEN-001. Catalog state already includes the curated row produced in TC-SCEN-001 (artifact reuse).
- **Operator intent**: An external agent (script or another AGH agent) manages the catalog without web UI: list rows, refresh, inspect status, select a manual model, and start a session via API.
- **Expected business outcome**: The agent's CLI/HTTP/UDS/Host API operations are deterministic, structured, byte-equal between transports for steady-state list payloads, and reflect the same persisted state as the web UI.
- **AGH surfaces used**: CLI (`agh provider models {list|refresh|status}`), HTTP (`/api/providers/...`, `/api/openai/v1/models`), UDS (`/api/providers/...`), Host API (`models/list|refresh|status`).
- **Real provider/LLM expectation**: ACP fake driver creates the session; opt-in `MODELCATALOG_LIVE=1` annex covers a real provider.
- **Blocked live-provider boundary**: documented in the verification report when not running live.
- **Scenario contract minimums covered**: agent role, CLI/HTTP/UDS/Host API channels, refresh storm, redaction, deterministic JSON output, capability gating.

## Actors and Agent Roles

| Actor / Agent | Role | Expected Behavior | Evidence Source |
|---------------|------|-------------------|-----------------|
| Remote agent | Catalog reader/operator | Drives CLI + HTTP + UDS + Host API | CLI transcripts |
| Daemon | Catalog authority | Serves identical projection | HTTP/UDS/Host API responses |
| Extension | `model.source` provider | Returns valid + invalid rows on demand | Extension subprocess transcript |

## Preconditions

- [ ] Bootstrap manifest exists; `AGH_HOME`, ports, sockets unique.
- [ ] Daemon running with extension fixture installed.
- [ ] TC-SCEN-001 catalog state present (`manual-gpt` curated row).

## Journey Steps

1. **Agent runs `agh provider models list -o json`.**
   - Surface: CLI.
   - Input: no flags.
   - **Expected:** JSON includes `manual-gpt` row from TC-SCEN-001; sources sorted deterministically; output structurally equivalent to HTTP `GET /api/providers/models`.
2. **Agent triggers refresh storm.**
   - Surface: CLI + HTTP concurrently for `codex`, `anthropic`, `gemini`.
   - **Expected:** Same-provider coalescing observed (TC-PERF-001); cross-provider parallel; redacted errors only.
3. **Agent inspects status via UDS.**
   - Surface: UDS client.
   - **Expected:** Same byte-equal status payload as HTTP for steady-state list; `refresh_request_id`, `last_refresh_at`, `last_error` redacted.
4. **Agent inspects OpenAI projection.**
   - Surface: HTTP `GET /api/openai/v1/models?provider_id=codex`.
   - **Expected:** OpenAI shape with `agh` metadata; UDS does NOT expose this route.
5. **Agent calls Host API `models/list` (with grant).**
   - **Expected:** Daemon-owned projection; structurally equivalent to HTTP/CLI; raw extension payload not leaked.
6. **Agent revokes Host API grant and retries.**
   - **Expected:** Deterministic capability error; no rows leaked.
7. **Agent creates a session via HTTP `POST /api/sessions` selecting manual model.**
   - **Expected:** Session creation succeeds; ACP control uses `session/set_config_option`; session fixture confirms.
8. **Disruption probe - extension returns invalid row mid-storm.**
   - **Expected:** Invalid row dropped; valid rows persist; redacted error surfaced; refresh request id correlated in logs.

## Required Evidence

- CLI transcripts with structured JSON output.
- HTTP/UDS response bodies (canonical sort) for byte-equality check.
- Host API response bodies before and after grant revoke.
- Daemon log entries with `refresh_request_id`, `provider_id`, `source_id`, `source_kind`, `extension_name` correlation keys.
- Extension subprocess transcript.

## Audit Coverage

- C4 (agent), C5 (CLI + HTTP + UDS + Host API), C8 (parity), C9 (provider boundary), C10 (artifact reuse from TC-SCEN-001), C11 (refresh storm + extension denial + invalid row), C14.

## Pass Criteria

- All four surfaces show identical persisted state.
- Refresh storm coalesced.
- Capability gate enforced.
- Manual model session created via API.

## Failure Criteria

- CLI/HTTP/UDS/Host API drift.
- OpenAI projection registered on UDS.
- Capability gate bypassed.
- Refresh storm causes SQLite `BUSY`.
