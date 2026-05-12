# Co-Ship Checklist

Confirm every item before committing the bundle.

## Source

- [ ] `internal/api/contract/**` change is intentional and minimal.
- [ ] No test-only types leaked into the contract package.
- [ ] No `interface{}`/`any` snuck into a wire shape (concrete types only).
- [ ] Interface assertions exist for any new exported types: `var _ Interface = (*Type)(nil)`.

## Generated

- [ ] `make codegen` ran with no errors.
- [ ] `openapi/agh.json` diff is committed alongside source.
- [ ] `web/src/generated/agh-openapi.d.ts` diff is committed alongside source.
- [ ] If JSON-RPC: Go→TS extension types regenerated.
- [ ] `make codegen-check` passes (no drift).

## Web

- [ ] Affected `web/src/systems/<system>/types.ts` no longer mirrors generated DTOs.
- [ ] Adapters use the generated shape end-to-end.
- [ ] Hooks (queries/mutations) have updated return types.
- [ ] Components props match the new exposed shape.
- [ ] Storybook stories use updated payloads.
- [ ] MSW fixtures match the new contract.
- [ ] `make web-typecheck` passes.
- [ ] `make web-test` passes.

## Site

- [ ] `make cli-docs` ran if CLI verbs changed.
- [ ] HTTP endpoint docs updated under `packages/site/content/runtime/`.
- [ ] Protocol docs updated under `packages/site/content/protocol/`.
- [ ] Configuration docs updated under `packages/site/content/runtime/configuration/`.
- [ ] Removed verbs/endpoints/keys: doc pages deleted (no "deprecated" markers).
- [ ] `cd packages/site && bun run source:generate && bun run typecheck && bun run test && bun run build` passes.

## Backend

- [ ] HTTP handler returns `claim_token_hash`, never raw `claim_token` (security invariant).
- [ ] All redaction tests still pass after the contract change.
- [ ] Observability fields (`claim_token_hash`, `lease_until`, etc.) updated if shape changes affect logs/metrics.
- [ ] `make verify` passes end-to-end.

## Commit

- [ ] One commit message summarizes both the source and the generated/web/site changes.
- [ ] Commit prefix is `feat:` or `refactor:` per `CLAUDE.md` commit style.
- [ ] PR description lists the co-shipped artifacts.
