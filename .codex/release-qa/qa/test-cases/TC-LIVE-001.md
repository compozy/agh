## TC-LIVE-001: Real LLM And AGH Network Smoke

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

### Objective

Verify AGH can use an installed real LLM-capable agent and route a network message in an isolated local runtime.

### Preconditions

- At least one supported CLI agent is installed (`codex`, `claude`, or equivalent).
- Required provider credentials are available in the environment or already authenticated.
- Isolated `AGH_HOME` can be created under `/tmp`.

### Test Steps

1. Run a direct real LLM smoke command.
   **Expected:** The agent returns the requested token or a deterministic success response.

2. Start AGH daemon with isolated `AGH_HOME` and network enabled.
   **Expected:** Daemon starts and `agh daemon status -o json` reports network `running`.

3. Create an ACP session using the real agent and send a normal prompt.
   **Expected:** Prompt completes and transcript/events are persisted.

4. Send a network `direct` message to the session.
   **Expected:** Network status/audit reports the direct message delivered, or the exact live-agent behavior is captured if the agent takes tool actions instead of returning a token.

### Edge Cases & Variations

| Variation                | Input                    | Expected Result                                  |
| ------------------------ | ------------------------ | ------------------------------------------------ |
| Missing credentials      | No provider auth         | Test is blocked with exact missing prerequisite. |
| Agent streams tool calls | Real agent invokes tools | AGH preserves events and does not crash.         |
