# TC-LLM-001: Real LLM Network Smoke

**Priority:** P1
**Type:** E2E / Production-like
**Status:** Pass with Caveat
**Created:** 2026-04-24

## Objective

Run one real provider-backed AGH agent turn, when local credentials and provider binaries are available, and verify it can receive a network-formatted prompt without crashing or losing audit records.

## Preconditions

- A supported ACP-compatible agent binary is installed.
- Required provider credentials exist in the local environment or provider config.
- Credentials are detected by name only; values are never printed.

## Test Steps

1. Detect installed provider binaries and credential variable names.
   **Expected:** at least one supported real provider is available, or the test is marked blocked with exact missing prerequisite.

2. Start AGH with an isolated temp home/workspace and real provider config.
   **Expected:** daemon starts, session can be created, network status is enabled.

3. Send a small direct network message to the real agent session.
   **Expected:** the agent receives the `<network-message>` wrapper and returns a normal turn completion.

4. Inspect audit/timeline.
   **Expected:** message has received/delivered records and no retry/backpressure errors.

## Execution History

| Date       | Tester | Build | Result | Notes                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ---------- | ------ | ----- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-24 | Codex  | local | Pass   | `OPENAI_API_KEY` and `codex` CLI were available. `codex exec` returned `AGH-LLM-SMOKE-OK`. An isolated AGH daemon using Codex ACP created real sessions, returned `AGH_REAL_NETWORK_OK` on a normal prompt, accepted a `direct` network envelope, and reached `messages_delivered=1`. Caveat: the real Codex agent followed AGH/network safety guidance and agentic behavior rather than returning only the token over the network message. |
