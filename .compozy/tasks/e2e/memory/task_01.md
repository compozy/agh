# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Build `internal/testutil/e2e/` as the shared runtime harness for later daemon and browser E2E work.
- Completed: external-process daemon boot, seeded config/workspace helpers, stable artifact manifest plumbing, and one migrated existing integration-suite consumer.

## Important Decisions

- The harness will start the real daemon as an external `agh daemon start --foreground` subprocess instead of importing `internal/daemon`, to avoid package cycles when HTTP/UDS integration suites import the harness.
- CLI access will be provided through a subprocess-backed helper around the built `agh` binary; HTTP and UDS access will use raw `http.Client` instances against the daemon’s public endpoints.
- Artifact manifests will record only captured surfaces using versioned entries with relative paths under a per-run artifacts directory.

## Learnings

- `internal/api/httpapi` and `internal/api/udsapi` both already carry large custom `newIntegrationRuntime` boot paths; reusing an in-process harness from those packages would cycle through `internal/daemon`.
- Existing integration tests already use current-test-binary ACP helper processes via `os.Executable()` and env-gated `-test.run=...` commands, which the new harness integration tests can reuse for real daemon session flows.
- The shared artifact collector needs explicit path-containment checks around canonical output paths to satisfy the repository's static analysis gate while keeping the manifest contract stable.

## Files / Surfaces

- Target package: `internal/testutil/e2e/`
- Migrated consumer surface: lightweight `internal/api/httpapi` integration coverage that only needs real daemon boot and public status/workspace surfaces.
- Public surfaces required now: HTTP `/api/daemon/status`, UDS `/api/daemon/status`, UDS/HTTP `/api/workspaces/resolve`, session transcript/events endpoints for artifact capture, and CLI `daemon status`.

## Errors / Corrections

- Initial in-process harness idea was rejected after confirming it would create import cycles for HTTP/UDS suites because `daemon` imports both transports.
- The initial `execabs` refactor left a stale `exec.Cmd` type import behind; the harness now keeps the `os/exec` type import while using `execabs` for subprocess launches.
- Artifact writes now validate containment within the collector root before hitting disk, which resolved the final gosec finding in `make verify`.

## Ready for Next Run

- Task complete. Next tasks should extend this harness with ACP mock drivers, multi-agent fixtures, and broader runtime/browser scenarios instead of cloning new package-local boot helpers.
