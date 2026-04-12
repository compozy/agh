# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Deliver task 11 by adding the Telegram reference channel adapter under `sdk/examples/telegram-reference`, the reusable subprocess-backed harness under `internal/extensiontest`, the required unit/integration tests, and clean verification evidence.

## Important Decisions
- Kept the reference adapter in the SDK examples area instead of the daemon so the repo demonstrates the approved extension shape directly.
- Put the reusable conformance harness in `internal/extensiontest`; restart failure injection is opt-in via `HarnessConfig.CrashOnceOnFirstDelivery` so normal delivery tests do not crash implicitly.
- Treated Host API marker writes as best-effort side effects, but handled every write error explicitly by reporting it to adapter stderr so lint/verification stay strict.

## Learnings
- Channel adapters can observe RPC code `-32003` / message `Not initialized` if they call Host API methods too early during startup; the reference adapter now uses bounded retry rather than a fixed sleep.
- Broker delivery ordering is monotonic, not necessarily contiguous; coalescing can skip intermediate sequence numbers, so adapter ack logic must accept increasing `seq` values rather than exact `+1` steps.
- `auth_required` channel instances do not accept inbound ingest through the Host API, so auth-required observability coverage and restart/delivery coverage must stay in separate tests.

## Files / Surfaces
- `sdk/examples/telegram-reference/extension.toml`
- `sdk/examples/telegram-reference/README.md`
- `sdk/examples/telegram-reference/main.go`
- `sdk/examples/telegram-reference/main_test.go`
- `internal/extensiontest/channel_adapter_harness.go`
- `internal/extensiontest/channel_adapter_harness_test.go`
- `internal/extensiontest/channel_adapter_harness_integration_test.go`
- `internal/extension/telegram_reference_integration_test.go`

## Errors / Corrections
- Initial adapter startup attempted Host API calls before the subprocess transport finished initialize; fixed with bounded retry for `Not initialized`.
- Initial delivery ack logic assumed contiguous sequence numbers and failed when the broker coalesced intermediate events; fixed to enforce monotonic ordering only.
- `make verify` initially failed on ignored marker-write errors; fixed by surfacing those side-effect failures to stderr without turning them into delivery/runtime failures.

## Ready for Next Run
- Verification evidence captured:
  - `go test -cover ./sdk/examples/telegram-reference` -> `81.0%`
  - `go test -tags integration -cover ./internal/extensiontest` -> `85.8%`
  - `go test -tags integration ./internal/extension -run 'TestTelegramReferenceAdapter' -count=1 -timeout 60s`
  - `make verify`
- Remaining work for this run is tracking updates and the local commit only.
