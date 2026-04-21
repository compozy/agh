# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add a runtime-facing capability authoring guide under `docs/agents/capabilities.md` that documents supported layouts, invalid layouts, required and optional fields, no-catalog behavior, and the projection split between brief and rich discovery.

## Important Decisions

- Keep the new guide focused on local runtime authoring and validation; move wire-key details into concise cross-links to RFC 003 instead of restating local filesystem rules there.
- Update RFC 001 only enough to acknowledge capability catalogs as portable sidecars in self-contained agent directories.
- Update RFC 003 with short AGH-specific `greet` and `whois` extension notes because the current workspace copy did not yet name `agh.capabilities_brief`, `agh.include`, `agh.capability_ids`, or `agh.capability_catalog`.

## Learnings

- The shipped loader in `internal/config/capabilities.go` accepts exactly four local shapes: `capabilities.toml`, `capabilities.json`, `capabilities/*.toml`, and `capabilities/*.json`.
- Directory mode loads only regular files of one selected format, ignores dotfiles and non-matching extensions, requires basename-without-extension to match `id`, and returns a hard error for mixed formats.
- Brief discovery is projected through `peer_card.capabilities` plus `peer_card.ext["agh.capabilities_brief"]`, while rich discovery is explicit `whois` envelope `ext` data only.
- RFC 003 needed explicit AGH capability extension notes in the workspace copy so the runtime guide could point to a wire contract that names the shipped keys.

## Files / Surfaces

- `docs/agents/capabilities.md`
- `docs/rfcs/001_agent-md-with-skills-memory.md`
- `docs/rfcs/003_agh-network-v0.md`
- `internal/config/capabilities.go`
- `internal/network/capability_brief.go`
- `internal/network/capability_catalog.go`

## Errors / Corrections

- Corrected the no-catalog wording in the new guide so rich discovery empty-catalog behavior is described only for explicit `whois` requests, matching the implementation.
- Confirmed with explicit content checks plus `make verify` that the guide, RFC 001, and RFC 003 use the shipped field names and wire keys consistently.

## Ready for Next Run

- Task 05 is complete after guide/RFC updates, explicit doc consistency checks, and a clean `make verify` run. Tracking files remain intentionally out of the code commit unless repository policy changes.
