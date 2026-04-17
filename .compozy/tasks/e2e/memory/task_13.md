# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add explicit repo-local `runtime`, `web`, combined, and nightly E2E entrypoints that match the tiered techspec instead of using the broad `test-integration` sweep.
- Keep the default PR-required lane separate from credentialed/nightly coverage while making local and automation invocations consistent across Make, Mage, and package scripts.

## Important Decisions

- Centralized the lane matrix in a normal Go package, `internal/e2elane`, so Mage targets and regression tests share one source of truth.
- Scoped the PR-required runtime lane to `internal/daemon` `^TestDaemonE2E`, HTTP transport parity `^TestHTTPTransport`, and UDS transport parity `^TestUDSTransport` instead of `./...` integration packages.
- Encoded the browser lane as daemon-served Playwright scripts in `web/package.json`; default browser runs use `test:e2e:daemon-served` with `--grep-invert @nightly`, and nightly browser runs use `--grep @nightly --pass-with-no-tests`.

## Learnings

- `make help` is a cheap way to verify Mage target exposure through the same repo-local surface that developers use, and `make -n` is sufficient for command-level wiring checks without rerunning the whole lane.
- The broad repo gate caught two support-code issues that the focused lane tests did not: a long constant line and `exec.Command` in a test helper. Fixing those before final verification kept the lane change clean under `make verify`.
- The nightly lane works cleanly without Daytona credentials because the targeted tests already skip on missing `DAYTONA_API_KEY`; the nightly Playwright selector also tolerates zero tagged specs.

## Files / Surfaces

- `internal/e2elane/lanes.go`
- `internal/e2elane/lanes_test.go`
- `internal/e2elane/command_wiring_test.go`
- `magefile.go`
- `Makefile`
- `package.json`
- `web/package.json`

## Errors / Corrections

- Initial `make verify` failed on `lll` for the Daytona nightly regex constant and `noctx` in the command-wiring helper. Split the regex constant across lines and switched the helper to `exec.CommandContext` with a one-minute timeout, then reran the full validation chain from scratch.

## Ready for Next Run

- Validation completed cleanly after the lint fixes:
  - `go test -cover ./internal/e2elane -count=1` (`100.0%` coverage)
  - `make test-e2e-runtime`
  - `make test-e2e-web`
  - `make test-e2e`
  - `make test-e2e-nightly` (Daytona tests skipped cleanly without credentials)
  - `make web-lint`
  - `make web-typecheck`
  - `make verify`
