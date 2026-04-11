# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build `internal/extension/manifest.go` and tests for task_03: parse `extension.toml` first, fall back to `extension.json`, validate manifest schema/compatibility, and return a flat `Manifest` without executing extension code.

## Important Decisions
- Decoder will accept the wrapped tech spec document shape (`[extension]` / `"extension"`) and flatten it into the exported `Manifest` returned by `LoadManifest`.
- `resources.hooks` will use extension-local typed config structs modeled after `internal/config/hooks.go` because `internal/hooks.HookDecl` is not TOML-decodable.
- `resources.mcp_servers` will be normalized from a named map so equivalent TOML and JSON inputs compare equal in tests.
- Schema validation covers required fields, semver parsing, daemon-version compatibility, dotted capability/security identifiers, slash-separated action names, and wildcard `security.capabilities = ["*"]`.

## Learnings
- The task docs’ exported `Manifest` shape and the tech spec examples differ: examples nest core fields under `extension`, so the parser needs a wrapper layer even though callers should receive a flat manifest.
- Existing repo semver logic is local to `internal/cli/skill_marketplace.go`, so task_03 likely needs its own package-local semantic version parsing or a small shared helper.
- Full-manifest fixtures are sufficient to cover resources, capabilities, actions, subprocess env placeholders, TOML-first precedence, and forward-compatible unknown top-level sections without executing any extension code.

## Files / Surfaces
- `.compozy/tasks/ext-architecture/task_03.md`
- `.compozy/tasks/ext-architecture/_techspec.md`
- `.compozy/tasks/ext-architecture/_examples.md`
- `.compozy/tasks/ext-architecture/adrs/adr-003.md`
- `.compozy/tasks/ext-architecture/adrs/adr-005.md`
- `internal/config/hooks.go`
- `internal/config/provider.go`
- `internal/version/version.go`
- `internal/extension/manifest.go`
- `internal/extension/manifest_test.go`

## Errors / Corrections
- No implementation errors yet. Baseline check confirmed `internal/extension/` does not exist and unrelated worktree changes must remain untouched.
- Initial focused package coverage was 75.0%; added helper-oriented tests for typed errors, duration helpers, semantic-version branches, invalid action names, and manifest directory edge cases to reach 82.2%.

## Ready for Next Run
- Task implementation, verification, tracking, and the local code-only commit are complete.
- Verification evidence:
  - `go test ./internal/extension -coverprofile=/tmp/internal-extension.cover.out -covermode=count` → pass, 82.2% statements.
  - `make verify` → pass before commit and again after commit hook formatting.
- Local commit: `fe5978f` (`feat: add extension manifest parser`).
