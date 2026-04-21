# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Rewrite `docs/rfcs/003_agh-network-v0.md` and `docs/agents/capabilities.md` so the unified capability model is the only steady-state explanation of authoring, discovery, and transfer.

## Important Decisions

- Treat ADR-001 through ADR-003 plus task_04’s implemented discovery/API contracts as the source of truth, then word the docs against the current code rather than older split-model prose.
- Keep the runtime guide explicit about the wire/API boundary: wire discovery still uses `peer_card.capabilities`, `agh.capabilities_brief`, and `agh.capability_catalog`, while daemon consumers should use typed `peer_card.capabilities` and `capability_catalog` payloads instead of reading capability discovery blobs from API-visible `ext`.
- Put the required end-to-end authored -> `greet` -> `whois` -> `kind:"capability"` flow in `docs/agents/capabilities.md`, and rewrite the RFC worked example so it no longer reintroduces a second artifact type.

## Learnings

- The implemented runtime computes `digest` after normalization, canonicalizes `requirements` ordering before hashing, and reuses the same structured capability document for rich discovery and transfer validation.
- The network/runtime behavior now allows `kind:"capability"` to participate in interaction lifecycle flow when `interaction_id` is present, so the RFC must not claim that only `direct` can open an interaction.
- Local peers advertise `artifacts_supported = ["capability"]` even when they have no local capability catalog; transfer support is protocol-level, not dependent on discovery inventory size.

## Files / Surfaces

- `docs/rfcs/003_agh-network-v0.md`
- `docs/agents/capabilities.md`
- `.compozy/tasks/unified-capabilities/task_05.md`
- `.compozy/tasks/unified-capabilities/_tasks.md`

## Errors / Corrections

- Corrected an initial RFC contradiction after the first draft: lifecycle sections still implied only `direct` could open an interaction, so the final rewrite now reflects the implemented `capability` interaction behavior and directed-subject guidance.

## Ready for Next Run

- Local docs commit created: `d381885b` (`docs: unify capability docs`).
- `make verify` passed after the doc rewrites and again on the committed `HEAD`.
- Task tracking is updated locally, and the automatic commit intentionally includes only the doc changes, not workflow memory or task-tracking files.
