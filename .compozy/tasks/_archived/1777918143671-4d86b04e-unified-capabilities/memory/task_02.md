# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Replace the remaining `recipe` protocol kind in `internal/network` with `kind:"capability"`, including envelope types, registry/decoder wiring, digest-backed validation, helper text, and regression coverage.

## Important Decisions

- Treat the approved task docs plus ADR-002 as the source of truth for transfer shape: the capability envelope must carry the canonical structured capability fields needed for digest verification, not a recipe-style artifact blob or a second reduced schema.
- Keep discovery behavior unchanged for this task: `greet` and `whois` continue to own brief and rich discovery while `kind:"capability"` becomes the sole transfer artifact.
- Keep digest ownership in `internal/config`: the network validator recomputes transfer digests via `CanonicalCapabilityDigest`, and router rejection maps integrity failures to `verification_failed`.

## Learnings

- The current branch still exposes `KindRecipe`, `RecipeBody`, recipe-specific validation, lifecycle branching, delivery guidance, and recipe fixtures/tests across `internal/network`.
- Task_01 already provides the canonical digest inputs in `internal/config` and the runtime projection in `internal/session.NetworkPeerCapability`; task_02 should derive wire validation from that model instead of recreating digest rules ad hoc.
- Legacy `kind:"recipe"` is now explicitly rejected by validation, and malformed `kind:"capability"` transfers fail before router delivery or lifecycle side effects.

## Files / Surfaces

- `internal/config/capabilities.go`
- `internal/network/envelope.go`
- `internal/network/validate.go`
- `internal/network/lifecycle.go`
- `internal/network/router.go`
- `internal/network/delivery.go`
- `internal/network/manager.go`
- `internal/network/envelope_integration_test.go`
- `internal/network/validate_test.go`
- `internal/network/helpers_test.go`
- `internal/network/router_test.go`
- `internal/network/lifecycle_test.go`
- `internal/network/delivery_test.go`
- `internal/network/manager_test.go`
- `internal/network/peer_test.go`
- `internal/network/perf_bench_test.go`

## Errors / Corrections

- The techspec contains a narrower sample `CapabilityEnvelopePayload`, but ADR-002 explicitly rejects a second wire-only shape. This run follows the canonical structured capability document so the transferred digest can be recomputed and verified coherently.

## Ready for Next Run

- Fresh evidence on this run: `go test ./internal/network`, `go test -tags integration ./internal/network`, `go test -cover ./internal/network` (`81.7%`), and `make verify` all passed after the final task_02 changes.
