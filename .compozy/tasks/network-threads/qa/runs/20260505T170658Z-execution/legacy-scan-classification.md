# Legacy Surface Scan Classification

## Command

```bash
rg -n "interaction_id|kind:\s*\"direct\"|--interaction-id|DirectBody|KindDirect|network_selected_peer" packages/site/content openapi web/src/generated web/e2e web/src/systems/network internal/skills/bundled .agents/skills internal/network internal/api internal/cli -g '*.md' -g '*.mdx' -g '*.ts' -g '*.tsx' -g '*.json' -g '*.go'
```

## Evidence

- Raw scan: `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/legacy-scan.txt`
- Matches: 30.

## Classification

- Active runtime/API rejection paths:
  - `internal/network/validate.go` rejects `interaction_id`.
  - `internal/api/contract/contract.go` rejects `interaction_id` with a hard-cut error.
- Active tests and mocks:
  - `internal/network/validate_test.go`, `internal/api/contract/contract_test.go`, `internal/cli/network_test.go`, `internal/skills/bundled/bundled_test.go`, `web/e2e/network.spec.ts`, and Web component/action tests assert legacy fields and flags are absent or rejected.
  - `web/src/systems/network/mocks/handlers.ts` rejects `interaction_id` and `kind:"direct"` in test/storybook mock boundaries.
- Documentation:
  - `packages/site/content/protocol/*` mentions `kind:"direct"` and `interaction_id` only as rejected legacy/invalid shapes in protocol guidance and conformance material.
- Internal Web UI model:
  - `web/src/systems/network/components/activity/activity-feed.tsx` uses `kind: "direct"` as an internal activity-feed discriminant for linking direct-room summaries. It is not a wire envelope kind and does not construct a network send payload.

## Verdict

PASS. The scan found no active CLI/API/OpenAPI/Web generated contract or send path that still accepts or emits legacy `interaction_id`, `--interaction-id`, `network_selected_peer`, or wire `kind:"direct"` behavior. The remaining matches are rejection code, negative tests, documentation of invalid shapes, or a non-wire UI discriminant.
