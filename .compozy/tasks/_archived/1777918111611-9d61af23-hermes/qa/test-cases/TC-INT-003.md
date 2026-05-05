## TC-INT-003: ACP Lifecycle Failure Diagnostics

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 50 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that ACP/session lifecycle failures are classified at source, persisted, exposed through API/SSE/CLI, summarized in observe health, and backed by bounded redacted crash bundles where applicable.

### Traceability

- Task: task_03, ACP and Session Lifecycle Hardening.
- TechSpec: issues 14, 15, and 16; Testing Approach failure-kind classification, agent probes, crash bundles.
- ADR: ADR-001 lifecycle hardening track.
- Surfaces: `internal/acp`, `internal/session`, `internal/observe`, `internal/api/contract`, `internal/api/core`, CLI session read paths, session SSE events, web session/daemon fixtures, site session lifecycle docs.

### Preconditions

- Mock ACP providers can simulate startup, handshake, load-session, prompt, protocol, cancellation, permission, timeout, transport, and process-exit failures.
- Crash bundle output path points to an isolated temp AGH home.
- Test secrets are present in stderr/config only as known sentinel values for redaction checks.

### Test Steps

1. Trigger each supported failure class with mock ACP/session fixtures.
   - **Expected:** Persisted session metadata contains a normalized `failure.kind` matching the source error and a bounded redacted `summary`.

2. Trigger a provider process-exit or startup crash with stderr containing a sentinel secret.
   - **Expected:** A crash bundle is written with owner-only permissions, bounded content, and redacted sentinel values.

3. Read the failed session through CLI and API.
   - **Expected:** `agh session status -o json`, session list JSON, and HTTP session DTO expose the same `failure.kind`, `summary`, and optional `crash_bundle_path`.

4. Subscribe to session events or inspect an SSE terminal event sample.
   - **Expected:** Terminal event payload includes the same failure object without leaking stderr secrets.

5. Query observe health after failures and agent probes.
   - **Expected:** `health.failures.by_kind`, `health.failures.recent`, and `health.agent_probes` summarize redacted diagnostics and command availability.

6. Review web/session and site docs surfaces.
   - **Expected:** Web types/fixtures accept failure payloads, and site session lifecycle/observe health docs describe failure kinds and crash bundles accurately.

### Evidence To Capture

- `qa/logs/TC-INT-003/go-test-acp-session.log`
- `qa/logs/TC-INT-003/session-status.json`
- `qa/logs/TC-INT-003/session-event.json`
- `qa/logs/TC-INT-003/observe-health-failures.json`
- Redacted crash bundle path and content excerpt under `qa/logs/TC-INT-003/`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Unknown wrapped error | No structured kind | `unknown_failure` with redacted summary |
| Context cancellation | Canceled prompt context | `cancellation`, not process failure |
| Missing provider command | Probe command absent | Agent probe status reports missing command |
| Secret in crash path/text | Sentinel token in stderr/path | Redacted in all surfaced payloads |

### Related Test Cases

- TC-UI-003: Web generated contract compatibility for failure DTOs.
- TC-REG-002: Site docs cover failure diagnostics.
