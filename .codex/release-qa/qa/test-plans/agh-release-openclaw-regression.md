# AGH Release Regression Suite

## Execution Order

1. Smoke suite: build contract, CLI status, daemon start/stop, network status.
2. P0 network suite: join channel, direct delivery, whois/capability exchange, backpressure audit, persisted timeline.
3. P1 release suite: integration lane, runtime e2e, web e2e, browser network view.
4. Live smoke: real LLM prompt and real AGH network direct delivery when local credentials/tools exist.
5. Exploratory comparison: inspect OpenClaw-derived production patterns for uncovered gaps.

## Pass/Fail Criteria

PASS:

- All P0 cases pass.
- No critical bugs remain open.
- `make verify` passes after final code changes.
- Any blocked live scenarios list exact missing credentials or runtime prerequisites.

FAIL:

- Any P0 network delivery or audit case fails.
- `make verify` fails after final code changes.
- Network messages can be dropped without audit/status visibility.
- Daemon cannot start, stop, or report status in an isolated home.

CONDITIONAL:

- Credentialed live third-party channel flows are blocked, but all local network, e2e, and LLM smoke boundaries pass.
