# TC-SEC-001: Authored Context Bypass Rejection

**Priority:** P0

## Objective

Verify that extension, hook, and HTTP caller boundaries cannot bypass managed Soul/Heartbeat authoring services or agent-session identity validation.

## Preconditions

- Host API tests can run locally.
- Hook dispatch tests can run locally.
- Reused QA lab daemon is running for live HTTP negative checks.

## Test Steps

1. Attempt Host API Soul write with only read grant.
   **Expected:** Host API returns capability denied and the Soul authoring service receives zero write calls.
2. Attempt Host API Heartbeat wake/write behavior without write grant.
   **Expected:** Host API returns capability denied and the managed wake/write service receives zero calls.
3. Dispatch authored-context hook events with a malicious patch attempting to set `body`, `source_path`, or direct authored file targets.
   **Expected:** Hook patch is observation-only; payload state is unchanged and no direct authored file mutation control is exposed.
4. Enumerate Host API method names.
   **Expected:** No method exposes `SOUL.md`, `HEARTBEAT.md`, `file/write`, or `files/write`.
5. Call HTTP `/api/agent/context` without `X-AGH-Session-ID`.
   **Expected:** Response is `401` with an identity-required error body.
6. Call HTTP `/api/agent/context` with a stale or mismatched `X-AGH-Session-ID`.
   **Expected:** Response is `401` with an identity-stale or identity-mismatch error body; no context payload is returned.

## Behavioral Evidence

- Security/operator journey: untrusted automation and extension actors must use explicit managed grants and validated agent session identity.
- Artifacts: focused Go test logs and live HTTP negative response payloads.

## Disruption Probes

- Missing identity and invalid identity fail closed.
- Hook patches cannot become a hidden file mutation channel.

