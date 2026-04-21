# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Tasks 01-05 are implemented and verified. Capability catalogs now load during agent-directory discovery in `internal/config`, flow through the session-owned network join payload, project brief discovery into local `PeerCard` state, power explicit rich `whois` discovery through envelope `ext`, and now have a runtime-facing authoring guide in `docs/agents/capabilities.md`.

## Shared Decisions
- Downstream runtime and network tasks should consume `AgentDef.Capabilities` rather than rereading capability files; task 01 also updated workspace/daemon/extension clone paths so the loaded catalog survives those hops.
- Missing local capability catalogs leave `AgentDef.Capabilities == nil`; explicit capability files/directories normalize to a non-nil `CapabilityCatalog`, even when the catalog is empty.
- `session.NetworkPeerJoin` remains the runtime/network join contract, but after task 04 its runtime-owned capability slice now carries the full structured catalog fields needed for rich discovery. `internal/network` still does not depend on config loader types or filesystem formats.
- When no catalog was loaded, the join payload still carries a deterministic empty capability slice rather than `nil`, keeping downstream peer-card projection stable.
- Task 03 centralizes brief capability projection in `internal/network/capability_brief.go`; local peer cards now derive both `PeerCard.Capabilities` and `PeerCard.Ext["agh.capabilities_brief"]` from the same ordered join payload, and downstream router/registry/API surfaces preserve that metadata by cloning the finished `PeerCard` instead of rebuilding it.
- Task 04 stores the full local rich catalog separately from `PeerCard` in network-local state and projects it only when `whois` requests `ext["agh.include"]` containing `capability_catalog`; the rich catalog is returned in envelope `ext["agh.capability_catalog"]`, not `PeerCard.Ext`.

## Shared Learnings
- Capability JSON parsing reuses the strict unknown-field and trailing-data discipline used by `mcp.json`, but capability validation remains a separate config-owned surface.
- `internal/network` can still use `DefaultPeerCard(peerID)` as a base for local peer registration, but capability IDs from the join payload are now the authoritative source for the local peer card's `Capabilities` field.
- Rich `whois` filtering is response-local: responder selection still uses the existing `WhoisBody.Query` semantics, while `agh.capability_ids` only filters the returned rich catalog and preserves normalized catalog order.

## Open Risks
- None currently.

## Handoffs
- Task 06 should treat `docs/agents/capabilities.md` as the author-facing source for local layouts and validation rules, and `docs/rfcs/003_agh-network-v0.md` as the wire-facing source for brief and rich capability discovery keys.
