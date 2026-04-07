# Codex Review: `refac-v2` Analysis Set

## Overall assessment

This document set has good architectural instincts, but it is not internally trustworthy in its current form.

The biggest problems are:

- It mixes current decisions with stale recommendations. The summary was updated, but the underlying reports and the verification report were not fully brought along.
- At least one prominent finding is now false against the current tree: the claimed `fileSnapshot` / `snapshotsEqual` duplication between `skills` and `workspace` is already solved by `internal/filesnap`.
- Package/file/LOC metrics are inconsistent across documents. Some tables appear to count production files, some total Go files, some include test-only imports in coupling counts, and some are simply stale.
- The severity model is inflated. Several P0 items are maintainability issues, not critical risks.

The net result: the summary is useful as a direction-of-travel document, but it should not be treated as an accurate source of record until the stale findings and count methodology are corrected.

## `20260406-summary.md`

- `20260406-summary.md:5` says "20 packages" and "210 Go files". The current tree has 21 internal packages and 217 Go files under `internal/` + `cmd/agh`. The omitted package is `internal/filesnap`.
- The current and proposed dependency diagrams both omit `internal/filesnap` even though it is a real shared package used by both `skills` and `workspace`. See `internal/filesnap/filesnap.go:1-58`, `internal/skills/loader.go:13`, `internal/workspace/scanner.go:13`, and `internal/workspace/resolver.go:14`.
- `20260406-summary.md:188` proposes extracting "File snapshot/diffing" to `internal/fileutil` or `internal/snapshot`. That is outdated. The code already has `internal/filesnap`, and both relevant packages already use it.
- `20260406-summary.md:51`, `20260406-summary.md:498`, and the roadmap still carry F-SKL-04/05 as active work. That task is obsolete in the current tree.
- `20260406-summary.md:579` says "20 flat packages -> 16 top-level packages". That math is wrong if `filesnap` exists, which it does.
- `20260406-summary.md:596` says the infra report contributes 14 findings. That count is not reliable. The infra report contains 20 finding IDs, and even the filtered findings represented in the summary are more than 14.
- The agreed package naming decisions are present here and mostly sensible: `api/httpapi`, `api/udsapi`, `api/contract`, flat `fileutil` and `procutil`, and keeping `ComposedAssembler` in `daemon` are all better than the older alternatives.
- The P0 bucket is overstated. `A-1`, `F-MEM-01`, `F-MEM-04`, and `3.1` are important, but they are not "critical" in a normal engineering severity model. They are refactoring priorities, not service-severity incidents.

## `20260406-core.md`

- The package/file metrics are stale. `20260406-core.md:3-5` says `session` has 17 files with 9 production files, `acp` has 11, `store` has 18, and total non-test LOC is ~8,300. Current production counts are materially different: `session` 10 prod files, `acp` 7, `store` 14, `config` 6, for about 10,109 production LOC across the four packages.
- `20260406-core.md:310-318` still says `SessionRegistry` has 13 methods. The current interface has 11 methods at `internal/store/store.go:44-56`. The summary was corrected, but the core report was not.
- The coupling sections mix production and test imports. Example: `20260406-core.md:135` lists `testutil` as an `acp` import, and `20260406-core.md:268` lists `testutil` as a `store` import. Those are test-only imports, not production package dependencies.
- The substantive architectural calls are mostly reasonable: extracting `transcript`, narrowing `SessionRegistry`, and slimming `config` all make sense.
- The report is too package-happy in places. Ideas like `idgen` and `jsonutil` should be treated as last-resort extractions, not default moves. This codebase already has a flat-package discipline problem; it does not need a new tiny shared package for every duplicated helper.

## `20260406-api-layer.md`

- This report is out of sync with the agreed package naming. `20260406-api-layer.md:341-345` still recommends `api/http` and `api/uds`; the agreed structure is `api/httpapi` and `api/udsapi`. It also does not reflect the agreed `api/contract` package.
- The package-level file counts are wrong. `20260406-api-layer.md:104` says `httpapi` has 10 source files and 11 test files; the current tree has 11 production files and 10 test files. `20260406-api-layer.md:179` says `udsapi` has 8 source files and 12 test files; the current tree has 12 production files and 8 test files.
- `20260406-api-layer.md:360` claims ~1,200 lines of duplicate tests, while `20260406-api-layer.md:181` says ~800. The document contradicts itself on one of its headline findings.
- The `ServerBase` recommendation needs more discipline. If shared server lifecycle code is extracted, it should probably live in a transport-oriented helper package, not `apicore`. Putting transport lifecycle into `core` would blur the boundary between handler logic and transport boot/shutdown logic.
- Severity is inflated here too. `3.1` is a major maintenance drag, but calling duplicated transport tests P0 is not credible. That should be P1 at most.
- The `apisupport` merge recommendation is sound and aligns with the summary.

## `20260406-domain-features.md`

- This is the weakest individual report in the set because it contains a materially false finding.
- `20260406-domain-features.md:183-213` claims `fileSnapshot` and `snapshotsEqual` are duplicated between `skills` and `workspace`. That is not true in the current tree.
- There is no `type fileSnapshot` in `internal/skills/types.go`, no `type fileSnapshot` in `internal/workspace/scanner.go`, and no duplicated `snapshotsEqual` in either package.
- The current code already has a shared package for this concern: `internal/filesnap/filesnap.go:1-58`. `skills` imports it (`internal/skills/loader.go:13`, `internal/skills/registry.go:17`, `internal/skills/watcher.go:15`), and `workspace` imports it (`internal/workspace/scanner.go:13`, `internal/workspace/resolver.go:14`, `internal/workspace/clone.go:5`).
- Because of that, the coupling analysis is wrong too. `20260406-domain-features.md:229-232` says `skills` imports only `workspace` and `skills/bundled`; it also imports `filesnap`. `20260406-domain-features.md:323-324` says `workspace` imports only `config` and `store`; it also imports `filesnap`.
- `20260406-domain-features.md:5` says the scope is ~4,200 non-test LOC. Current production LOC across `memory`, `skills`, `workspace`, and `observe` is about 5,171. This is not a rounding issue; it is a stale snapshot.
- `20260406-domain-features.md:22-26` has internal severity drift. It labels the context-helper duplication as a top P1 opportunity, but the actual finding `F-MEM-07` is rated P2.
- The summary is missing several real domain findings from this report. The most notable are `F-SKL-03`, `F-OBS-05`, `F-WS-05`, and `F-SKL-08`.
- Architecturally, `memory/consolidation`, `frontmatter`, and moving `defaultPermissionModeResolver` out of `observe` all still make sense. The bad part is the stale snapshot-finding cluster, not the whole report.

## `20260406-infra-utils.md`

- `20260406-infra-utils.md:136-145` is out of sync with the agreed decision on `ComposedAssembler`. This report still says move it to `session`; the summary correctly keeps it in `daemon`.
- `20260406-infra-utils.md:102-110` still says the `Daemon` struct has 37 fields. The current struct has 45 fields at `internal/daemon/daemon.go:111-158`.
- `20260406-infra-utils.md:190` lists `testutil` as a daemon import. That is not true for production code. This is another example of coupling metrics mixing test-only imports into runtime architecture.
- `20260406-infra-utils.md:223` says logger could be consolidated under a `shared/` parent, but `20260406-infra-utils.md:384-393` correctly argues against a `shared/` parent. That is an internal contradiction.
- `20260406-infra-utils.md:338-343` undercounts its own filtered findings. `F4.2` and `F5.2` disappear from the group-level count, even though both are real findings in the document.
- The strongest recommendations in this report are still solid: centralize duplicated test helpers, move `syncDir` into `fileutil`, and slim `daemon/dream.go`.
- `F1.8` should not remain as an active refactoring recommendation if the agreed decision is to keep `ComposedAssembler` in `daemon`. Mark it resolved or remove it from the active set.

## `20260406-cli.md`

- This report is still using the old contract-package idea. `20260406-cli.md:77-80` recommends `internal/apitypes` / `internal/apicontract`; the agreed decision is `internal/api/contract`.
- `20260406-cli.md:5` says `internal/cli` has 26 files. The current tree has 27 Go files in `internal/cli`.
- The coupling analysis is overstated. `20260406-cli.md:408-414` says current CLI imports 14 non-test packages. The production CLI package does not import `acp`, `observe`, or `udsapi`; those show up in tests and integration helpers. The current production internal imports are much closer to 10 packages.
- Because of that, the summary claim that CLI is a 14-package presentation layer is overstated too.
- `CLI-9` is a real omitted finding. If the summary is going to carry low-value P3 items like `CLI-10` and `CLI-11`, it should also carry `CLI-9`, which is more architecturally meaningful.
- The main architectural recommendation still holds: extracting the API contract and skill-resolution logic out of CLI is the right direction.
- One caution: `api/contract` is only justified if the server also adopts it as the canonical wire model. If the server keeps serializing domain structs directly and only the CLI uses the contract package, this becomes a third model layer, not a simplification.

## `20260406-verification.md`

- This report should not be treated as a reliable backstop.
- `20260406-verification.md:154-162` marks F-SKL-04 as PASS, but the claimed duplicated types/functions do not exist in the current tree. This is the most important miss in the entire review set.
- `20260406-verification.md:237` still refers to `apitypes`, which is already obsolete relative to the agreed `api/contract` decision.
- `20260406-verification.md:247-253` says there are no contradictions in grouping recommendations. That misses at least three obvious contradictions: old `api/http` vs `api/httpapi`, old `apitypes` vs `api/contract`, and the unresolved `ComposedAssembler` move.
- `20260406-verification.md:304-311` concludes that the roadmap is internally consistent and the analysis is highly accurate. That conclusion no longer holds after the stale `filesnap` finding and the partially propagated naming decisions.
- The verification report also did not catch that several coupling sections across the analysis set mix production imports with test-only imports.

## Architectural structure assessment

The proposed end-state is mostly good after correction:

- `api/httpapi`, `api/udsapi`, `api/core`, and `api/testutil` under an `api/` subtree make sense.
- `api/contract` is a good idea if it becomes the single canonical wire contract for both transports and the CLI.
- `frontmatter`, `transcript`, and `memory/consolidation` are all architecturally coherent extractions.
- Keeping `fileutil` and `procutil` flat is correct.

The biggest architectural holes are:

- The final package structure omits `internal/filesnap`, which already exists and is already the correct home for snapshot metadata and equality.
- The reports still flirt with too many tiny-package extractions (`idgen`, `jsonutil`, `filelock`, `discovery`). That would recreate the same sprawl problem under different names.
- The API tree proposal is only clean if transport lifecycle helpers stay out of `api/core`.

## Roadmap assessment

The broad phase order is directionally good: cleanup, then shared extractions, then file splits, then interface/coupling changes, then tree reorg.

The problems are:

- Phase 0 is not "zero architectural risk" while it includes deleting `migrate_workspace.go`. That is a behavior change in persistence bootstrapping, not a trivial cleanup.
- Phase 1 still includes the obsolete snapshot extraction work that `internal/filesnap` already solved.
- The package-count math in the roadmap is stale because it ignores `filesnap`.
- If `api/contract` is part of the plan, the documents should say explicitly whether server responses are being moved onto contract DTOs before or during the API subtree reorg. Right now that dependency is implicit.

## Severity assessment

The severity ladder needs recalibration.

- Most current P0 items should be P1. Large files, duplicated tests, and multi-responsibility packages are high-priority refactors, not critical defects.
- The strongest candidate for a true high-severity structural issue is the duplicated `TokenUsage` type because it creates drift risk across boundaries.
- `F-MEM-01`, `F-MEM-04`, and `3.1` are important but not P0.
- The set would be more credible with a narrower P0 bucket and more disciplined P1/P2 differentiation.

## Findings present in individual reports but missing from the summary

These are the omissions I would either add to the summary or explicitly mark as intentionally excluded:

- `F-SKL-03`: duplicated frontmatter sentinel errors. This belongs with the frontmatter extraction story and is already referenced in the roadmap, but not in the summary finding tables.
- `F-OBS-05`: `activeCounts()` conflates active sessions and active agents. That is more important than several cosmetic P3 items that did make the summary.
- `F-WS-05`: `workspace.resolveOptions()` calls `ResolveHomePaths()` during option resolution. This is a real constructor-side-effect issue and should not silently disappear.
- `CLI-9`: domain helper logic living in CLI. This is a better architectural finding than `CLI-10` and `CLI-11`, which were included.
- `F4.2`: `fileutil` underutilization is omitted even though the report calls it out directly.
- `F5.2`: orphan-process logic sitting in `daemon` rather than `procutil` is omitted from the summary.

Some omitted items are acceptable as intentional exclusions:

- `A-6`, `CP-6`, `CLI-6`, and `F-OBS-06` are closer to analysis notes than must-track refactors.
- Positive observations like `F2.1` and `F3.1` do not need summary-table rows.

## Bottom line

I would not approve this document set as "done" yet.

What should happen next:

1. Regenerate all package/file/LOC counts from the current tree instead of hand-maintaining them.
2. Remove the stale `fileSnapshot` / `snapshotsEqual` finding and replace it with an explicit note that `internal/filesnap` already exists and should be reflected in the architecture docs.
3. Propagate the agreed naming decisions into `20260406-api-layer.md`, `20260406-cli.md`, and `20260406-verification.md`.
4. Resolve the `ComposedAssembler` contradiction everywhere, not just in the summary.
5. Re-score the P0/P1 boundary so the severity model matches actual risk.

Until that cleanup is done, the summary is useful as a brainstorming artifact, not as a precise refactoring plan.
