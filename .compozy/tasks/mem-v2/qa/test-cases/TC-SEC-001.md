# TC-SEC-001: Redaction, `_system` Non-Injection, And Public Payload Hygiene

**Priority:** P0
**Type:** Security
**Status:** Not Run
**Estimated Time:** 50 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify Memory v2 never leaks `_system/` artifacts, raw decision replay bytes, raw LLM traces, prompt-only memory sections, tokens, or secret-shaped values into recall, SSE, web, docs, or public API payloads.

## Preconditions

- [ ] Isolated daemon has memories, `_system` extractor/dreaming/ad_hoc artifacts, decisions, and session events.
- [ ] SSE/prompt stream can be exercised.
- [ ] Public docs and generated fixtures are available.

## Test Steps

1. **Run redaction-focused tests**
   - Input: `go test ./internal/sse ./internal/api/core ./internal/memory ./internal/memory/scan -run "Scrub|Redact|System|Reject|Decision" -count=1`
   - **Expected:** Scrub, reject, and redaction tests pass.

2. **Recall system sentinel**
   - Input: search for a string present only under `_system/`.
   - **Expected:** No result in default recall/search output.

3. **Inspect public decision payloads**
   - Input: `agh memory decisions show <id> -o json` and API equivalent.
   - **Expected:** Public payload omits raw `post_content`, `prior_content`, raw LLM output, and prompt text.

4. **Exercise SSE/log-facing surface**
   - Input: produce output containing literal and JSON-escaped `<memory-context>`.
   - **Expected:** SSE/log payloads scrub or neutralize prompt-only memory context markers.

5. **Search web fixtures and docs**
   - Input: grep web fixtures/docs for forbidden raw token/legacy memory/tool patterns.
   - **Expected:** Only explicit negative test regexes contain forbidden literals; runtime docs and fixtures are clean.

6. **Threat scan rejection**
   - Input: attempt to write invisible Unicode or prompt-injection text.
   - **Expected:** Controller returns REJECT, writes no curated file, and emits redaction-safe `memory.write.rejected`.

## Evidence To Capture

- Go test logs.
- Recall/search output.
- Decision payload JSON.
- SSE/log payload samples.
- Grep results.
- Rejection event row.

## Failure Criteria

- `_system` content appears in default recall.
- Public payload exposes raw replay content or raw LLM trace.
- SSE leaks `<memory-context>` markers.

