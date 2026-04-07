# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Extend ACP session start/resume so resolver-provided `AdditionalDirs` reach agent JSON-RPC payloads, using resolver-aligned path normalization and focused ACP/session regression coverage.

## Important Decisions
- The task `_techspec.md` is the source of truth for the wire contract, so ACP now sends a top-level snake_case `additional_dirs` field on both `session/new` and `session/load`.
- `internal/acp` keeps using the upstream ACP SDK for responses and most request fields, but wraps start/load requests in local wire structs because the SDK version in the repo has no `AdditionalDirs` field.
- `normalizeStartOpts` now canonicalizes both `Cwd` and `AdditionalDirs` through absolute-path resolution, `EvalSymlinks`, existence checks, directory checks, dedupe, and root-dir filtering to stay consistent with resolver semantics.

## Learnings
- Capturing helper-process stdin with `io.TeeReader` was enough to assert the exact outbound JSON-RPC payload; the helper agent did not need a custom protocol server.
- On macOS temp directories, canonicalization can rewrite `/var/...` temp paths to `/private/var/...`, so ACP transport assertions must compare against canonicalized expectations rather than raw `t.TempDir()` strings.

## Files / Surfaces
- `internal/acp/types.go`
- `internal/acp/client.go`
- `internal/acp/handlers.go`
- `internal/acp/client_test.go`
- `internal/acp/handlers_test.go`
- `internal/session/manager.go`
- `internal/session/manager_test.go`
- `internal/session/additional_test.go`

## Errors / Corrections
- The first full `make verify` run failed on `errcheck` because the new capture-file helper deferred `captureFile.Close()` without handling the returned error. The helper now closes the file via an explicit ignored-error closure and the full verification gate passed afterward.

## Ready for Next Run
- ACP permission sandboxing still scopes file access to `Cwd` only; if a later task needs agents to read additional roots through ACP file APIs, `internal/acp/permission.go` will need a follow-up design change.
