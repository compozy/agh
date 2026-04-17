# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add task_05 runtime coverage in `internal/daemon` for bridge ingress + delivery progression and a real extension subprocess Host API bridge flow.
- Keep shared harness work in `internal/testutil/e2e` and `internal/extensiontest`; keep daemon truth in the composition-root runtime lane.
- Verification target is satisfied on commit `1fa0734e`: `make verify` passed, and integration-inclusive coverage passed for `internal/daemon`, `internal/testutil/e2e`, and `internal/extensiontest`.

## Important Decisions

- Reuse the real `sdk/examples/telegram-reference` subprocess as the extension-boundary scenario instead of introducing a new ad-hoc fixture extension.
- Extend `internal/testutil/e2e` with typed bridge/extension API helpers plus bridge-specific artifact capture so the daemon test stays readable and artifact output stays stable.
- Export marker-file wait/report helpers from `internal/extensiontest` so `internal/daemon` can consume the same provider markers without depending on the `*Harness` type.
- Treat bridge creation, secret binding, and runtime start/restart as distinct runtime operations in the daemon test; `PutSecretBinding` persists state only.
- Drive ingress deterministically with a fixture-backed ACP mock agent and a pair of fake Telegram inbound updates so the runtime test can prove first-route creation and second-route session reuse without external services.
- Build the `telegram-reference` adapter from the repo root but write the binary into a temp extension copy, so package-parallel integration runs do not race on `sdk/examples/telegram-reference/bin/telegram-reference`.

## Learnings

- `internal/daemon` already has bridge boot/restart package-level integration coverage, but no runtime-harness scenario proving real ingress, session reuse, and provider-side delivery from the public surfaces.
- The existing bridge artifact plumbing only captures health and generic provider calls; task_05 needs additional route/bridge/secret state snapshots to make failures actionable.
- `telegram-reference` already writes the exact provider markers needed for task_05: handshake, ownership, state reports, deliveries, ingests, starts, and shutdown.
- The daemon operator secret-binding surface is `/api/bridges/:id/secret-bindings` and `/api/bridges/:id/secret-bindings/:binding_name`; using `/secrets` fails at runtime.
- A copied extension directory cannot be built in-place because `sdk/examples/telegram-reference` relies on the repo module root; the stable approach is `go build` from the repo root with a temp output path.

## Files / Surfaces

- `internal/testutil/e2e/runtime_harness.go`
- `internal/testutil/e2e/bridges_extensions.go`
- `internal/testutil/e2e/runtime_harness_helpers_test.go`
- `internal/testutil/e2e/artifacts.go`
- `internal/extensiontest/bridge_adapter_harness.go`
- `internal/extensiontest/bridge_adapter_harness_test.go`
- `internal/daemon/daemon_bridge_extension_integration_test.go`
- `internal/daemon/bridge_extension_e2e_assertions_test.go`
- `internal/testutil/acpmock/testdata/bridge_ingress_fixture.json`

## Errors / Corrections

- Initial assumption corrected: secret binding writes do not reload managed bridge runtimes, so the runtime scenario must explicitly start or restart the bridge after binding.
- Initial helper implementation used the wrong UDS secret-binding path; corrected to `/secret-bindings` before final verification.
- Initial temp-copy flake fix tried to build inside the copied extension directory and failed module resolution; corrected by building from the repo root into the copied directory's `bin/`.

## Ready for Next Run

- Implementation and verification are complete for task_05.
- Code changes are committed locally as `1fa0734e` (`test: add runtime bridge ingress e2e`).
- Verified commands:
  - `go test -count=1 -tags integration -cover ./internal/daemon`
  - `go test -tags integration -cover ./internal/daemon ./internal/testutil/e2e ./internal/extensiontest`
  - `make verify`
