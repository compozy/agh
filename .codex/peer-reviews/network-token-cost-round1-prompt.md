You are performing a strict, read-only implementation peer review for AGH.

Return only a single JSON object with this shape:

{
"verdict": "SHIP|FIX_BEFORE_SHIP|REWORK",
"completed": false,
"confidence": 0.0,
"summary": "",
"evidence": {
"files_inspected": [],
"commands_or_logs_inspected": [],
"verification_evidence": []
},
"blockers": [],
"risks": [],
"nits": [],
"missing_work": [],
"next_round_guidance": "",
"review_notes": {
"readiness": "",
"uncertainty": "",
"skipped_checks": []
}
}

Set completed=true only for SHIP. Do not edit files, do not commit, and do not run destructive commands.

Original objective:

Implement the accepted AGH Network thread routing and prompt-cost accounting plan end to end: make thread prompt delivery target/participant bounded, persist delivered prompt size and deterministic estimated token aggregates keyed by workspace/channel/thread/peer, expose persisted audit size and coordination-cost payloads through existing network contracts, update bundled AGH skill and site protocol/runtime docs, co-ship OpenAPI/TypeScript codegen, keep unrelated worktree changes untouched, run focused tests and final `make verify`, and do not commit or push. After implementation, peer review until SHIP.

Out of scope:

- NATS transport swaps.
- CRDT substrates.
- Learned/RL routing.
- Web UI redesign beyond changed contracts.
- Turn-taking/interruption/budgets.

Important review focus:

- Thread prompt delivery must be bounded:
  - Directed thread envelopes deliver only to the explicit target.
  - Untargeted thread envelopes deliver only to persisted participants.
  - New untargeted thread messages without participants/resolver must not fall back to channel fanout.
- Prompt cost aggregate must persist delivered rendered prompt size and deterministic estimated tokens keyed by workspace/channel/thread/peer.
- Existing persisted audit `Size` must be exposed in network conversation message payloads.
- Coordination-cost payloads must appear on existing network thread contracts and generated OpenAPI/TypeScript artifacts.
- Schema migration must be numbered and registered.
- Tests should follow AGH conventions and cover the behavioral invariants, not just static drift.
- Docs and bundled `skills/agh` network guidance must match runtime truth.

Evidence already produced by the implementation pass:

- `go test ./internal/network ./internal/store ./internal/store/globaldb ./internal/api/core ./internal/acp` passed with 2205 tests.
- `make verify` passed end-to-end after formatting three previously unrelated files with explicit user authorization.

Relevant changed areas to inspect:

- `internal/network/router.go`
- `internal/network/router_test.go`
- `internal/network/delivery.go`
- `internal/network/delivery_test.go`
- `internal/network/manager.go`
- `internal/network/manager_test.go`
- `internal/acp/types.go`
- `internal/store/types.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_network_conversations.go`
- `internal/store/globaldb/global_db_network_messages.go`
- `internal/store/globaldb/global_db_network_token_stats.go`
- `internal/store/globaldb/global_db_network_conversation_repository_test.go`
- `internal/store/network_conversation_types_test.go`
- `internal/api/contract/contract.go`
- `internal/api/contract/responses.go`
- `internal/api/core/network_conversations.go`
- `internal/api/core/network_details.go`
- `internal/api/core/network_test.go`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`
- `sdk/typescript/src/generated/contracts.ts`
- `skills/agh/references/network.md`
- `packages/site/content/protocol/delivery.mdx`
- `packages/site/content/protocol/envelope.mdx`
- `packages/site/content/runtime/core/network/threads.mdx`
- `packages/site/content/runtime/guides/coordinate-agents-over-network.mdx`

Use the current worktree as authoritative. Include concrete file and line references for every blocker/risk/nit. If you cannot verify a requirement, report it as missing work.
