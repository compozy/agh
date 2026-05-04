# BUG-004: HTTP prompt disconnect cancels terminal event drain too early

## Status

Fixed in Task 11.

## Severity

P0 API/runtime regression. HTTP prompt clients that disconnect before the agent finishes could drop terminal prompt events and durable error evidence, leaving API/SSE and daemon E2E flows without the final `tool_result`, `done`, or blocked-cancel diagnostics.

## Reproduction

1. Run the integration lane:

```bash
go test -tags integration ./internal/api/httpapi ./internal/daemon -run 'TestHTTPPromptPersistsTerminalEventsAfterClientDisconnect|TestDaemonE2EACPmockBlockedCancelStopsPromptWithoutOrphaning'
```

2. In the HTTP prompt case, disconnect the client while the agent stream is still producing events.

## Observed

The detached HTTP drain stopped before persisting the agent's terminal events. `TestHTTPPromptPersistsTerminalEventsAfterClientDisconnect` missed `tool_result` and `done`, and the daemon blocked-cancel E2E missed the expected terminal error evidence.

## Expected

HTTP prompt handling should mirror UDS behavior: if the client disconnects, continue draining the agent stream for a bounded timeout, persist terminal events, then cancel the prompt context.

## Root Cause

`internal/api/httpapi/prompt.go` canceled the prompt context before starting the detached drain. That cancellation raced ahead of event persistence and interrupted the same stream the drain was supposed to preserve.

## Fix

Changed HTTP prompt draining to defer `cancelPrompt` until the detached drain completes or times out. Updated handler and drain-helper tests to assert that the prompt context remains alive during detached drain and is canceled afterward.

## Verification Evidence

- Failing integration repro: `.compozy/tasks/hermes/qa/logs/final/failure-repro/httpapi-prompt-approve.log`
- Failing daemon repro: `.compozy/tasks/hermes/qa/logs/final/failure-repro/daemon-blocked-cancel.log`
- Focused unit fix proof: `.compozy/tasks/hermes/qa/logs/final/failure-repro/httpapi-drain-unit-after-fix.log`
- Focused integration proof: `.compozy/tasks/hermes/qa/logs/final/failure-repro/httpapi-prompt-approve-after-fix.log`
- Focused daemon proof: `.compozy/tasks/hermes/qa/logs/final/failure-repro/daemon-blocked-cancel-after-http-fix.log`
- Full integration after fixes: `.compozy/tasks/hermes/qa/logs/final/make-test-integration-after-fixes.log`
