# SMOKE-001: Provider Model Catalog Smoke Readiness

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-05-07
**Last Updated:** 2026-05-07

---

## Objective

Confirm the isolated QA lab is healthy enough to execute release-grade catalog scenarios. Smoke is **entry criteria only**; passing this case proves nothing about feature behavior.

## Preconditions

- [ ] `agh-qa-bootstrap` produced `bootstrap-manifest.json` for the run.
- [ ] Unique `AGH_HOME`, ports, and tmux socket allocated.
- [ ] `AGH_WEB_API_PROXY_TARGET` exported from manifest.
- [ ] No production code changes pending beyond Task 12 / Task 13 QA artifacts.

## Test Steps

1. **Verify daemon binary builds.**
   - Command: `make build`.
   - **Expected:** Exit 0; binary present at `bin/agh`.
2. **Verify codegen contracts are clean.**
   - Command: `make codegen-check`.
   - **Expected:** No drift in `openapi/agh.json` or `web/src/generated/agh-openapi.d.ts`.
3. **Verify Bun typecheck and unit tests.**
   - Command: `make bun-typecheck && make bun-test`.
   - **Expected:** All workspaces pass; vitest catches no regression.
4. **Verify focused Go gates compile and pass.**
   - Command: `go test -race -count=1 ./internal/config ./internal/store/globaldb ./internal/modelcatalog/... ./internal/acp ./internal/api/... ./internal/cli ./internal/extension/...`.
   - **Expected:** Exit 0.
5. **Boot the daemon and request status.**
   - Command (in lab): `agh daemon start --foreground &` then `agh provider models status -o json`.
   - **Expected:** JSON payload includes `sources` array with `idle` or `succeeded` `refresh_state`.

## Audit Coverage

- Smoke entry only. Does **not** satisfy any release-grade audit minimum.

## Pass Criteria

- All five steps exit 0.
- Daemon responds within 5s.

## Failure Criteria

- Any step exits non-zero.
- Daemon hangs or returns OS-level error.

## Notes

If smoke fails, halt the QA run and report the failing step in `qa/verification-report.md` before any TC-FUNC/INT execution.
