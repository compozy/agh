# Task Memory: reviews-001

## Objective Snapshot

- Phase: D.2 manual review round 001 remediation.
- Goal: resolve the Claude Code Opus `cy-review-round` findings under `.compozy/tasks/mem-v2/reviews-001/` and close the round through the Codex Loop state helper.
- Current result: completed locally. All 5 issues are `resolved`; `check-rounds-clean` reports `clean=true critical=0 high=0 total=5`.

## Important Decisions

- Do not run one full `coderabbit review --agent` for this branch: CodeRabbit rejected the full diff with `payload_too_large`.
- Do not run one `--dir internal` review: CodeRabbit rejected it with `Too many files` because the scoped tree exceeded its 150-file limit.
- Use scoped reviews by changed surface and consolidate the JSONL outputs before calling `coderabbit-to-rounds.py`.
- The CodeRabbit CLI v0.4.5 emits agent output as JSONL event streams. The local converter now accepts JSONL, `fileName`, and `codegenInstructions` so it can convert the real current CLI schema.
- User explicitly replaced the blocked CodeRabbit lane with a manual Claude Code Opus review through `compozy exec` and `cy-review-round`.
- Treat `reviews-001/` as a manual provider-compatible round for `cy-fix-reviews`; do not merge it with the partial CodeRabbit outputs.

## Learnings

- CodeRabbit rate limiting can trigger after a few scoped reviews. The last retry returned `rate_limit` with `waitTime=14 minutes and 35 seconds`.
- A blocked D.1 iteration should not create `reviews-001/` from partial provider coverage as if it were a complete round.
- The manual Claude review rejected two candidate findings after direct verification before filing issues: root `-o` output is already a persistent CLI flag, and `_system/` entries are filtered by top-level directory scanning.

## Files / Surfaces

- Converter updated:
  - `.agents/skills/cy-codex-loop/scripts/coderabbit-to-rounds.py`
  - `.agents/skills/cy-codex-loop/tests/test_scripts.py`
- Captured CodeRabbit outputs:
  - `/tmp/cy-codex-loop-cr-mem-v2-29.json` — full diff attempt, failed with `payload_too_large`.
  - `/tmp/cy-codex-loop-cr-mem-v2-internal.jsonl` — `internal` attempt, failed with `Too many files`.
  - `/tmp/cy-codex-loop-cr-mem-v2-probe.jsonl` — `internal/resources`, completed with 0 findings.
  - `/tmp/cy-codex-loop-cr-mem-v2-internal-memory.jsonl` — `internal/memory`, completed with 22 findings.
  - `/tmp/cy-codex-loop-cr-mem-v2-internal-api.jsonl` — `internal/api`, completed with 5 findings.
  - `/tmp/cy-codex-loop-cr-mem-v2-internal-cli-retry2.jsonl` — `internal/cli`, completed with 4 findings.
  - `/tmp/cy-codex-loop-cr-mem-v2-internal-daemon.jsonl` — `internal/daemon`, completed with 0 findings.
  - `/tmp/cy-codex-loop-cr-mem-v2-internal-extension.jsonl` — `internal/extension`, completed with 0 findings.
  - `/tmp/cy-codex-loop-cr-mem-v2-internal-session.jsonl` — `internal/session`, blocked by `rate_limit`.
- Manual review prompt/output:
  - `/tmp/cy-codex-loop-mem-v2-manual-review-001.md`
  - `/tmp/cy-codex-loop-mem-v2-manual-review-001-output.jsonl`
- Manual review artifacts:
  - `.compozy/tasks/mem-v2/reviews-001/issue_001.md`
  - `.compozy/tasks/mem-v2/reviews-001/issue_002.md`
  - `.compozy/tasks/mem-v2/reviews-001/issue_003.md`
  - `.compozy/tasks/mem-v2/reviews-001/issue_004.md`
  - `.compozy/tasks/mem-v2/reviews-001/issue_005.md`

## Errors / Corrections

- The first helper conversion assumption was wrong for current CodeRabbit CLI output. Fixed root cause by supporting JSONL event streams and the current finding fields rather than post-processing by hand.
- A zsh batch used `status` as a variable name, which is read-only in zsh. Use `rc` for later shell loops.
- CodeRabbit provider rate limit prevented full D.1 coverage in the previous iterations. The latest blocked scope was `internal/session` with `waitTime=12 minutes and 51 seconds`.
- Claude initially wrote issue 002 with the wrong author value and corrected it to the required `author: claude-code` before completion.

## Ready for Next Run

- Update `state.yaml` with `--round-complete reviews-001 --critical 0 --high 0` after this memory file and the shared workflow memory are saved.
- Re-run `python3 .agents/skills/cy-codex-loop/scripts/detect-phase.py mem-v2`; it should either request the next Phase D review round or enter Phase E once the required clean streak is satisfied.

## Round Summary

- Round directory: `.compozy/tasks/mem-v2/reviews-001/`.
- Provider: manual.
- Author: Claude Code Opus via `compozy exec --ide claude --model opus --reasoning-effort xhigh`.
- Issue count: 5 total — 0 critical, 4 high, 1 medium, 0 low.
- Blocking issues before fix phase:
  - `issue_001.md` — high — bounded `SignalRecorder` queue not implemented.
  - `issue_002.md` — high — sub-agent native memory writes are not denied.
  - `issue_003.md` — high — Slice 1 HTTP/UDS memory routes still return `memory.unsupported`.
  - `issue_004.md` — high — extractor inbox DLQ replay is not idempotent for multi-candidate files.
  - `issue_005.md` — medium — extractor lacks mutual exclusion with explicit `agh__memory_propose` in the same turn.
- Verification evidence for the round:
  - Claude run reported `make lint` clean with `0 issues`.
  - Independent validation confirmed 5 issue files with matching manual frontmatter, shared `round_created_at`, `author: claude-code`, and no `_meta.md`.
  - Pre-remediation `python3 .agents/skills/cy-codex-loop/scripts/check-rounds-clean.py .compozy/tasks/mem-v2/reviews-001` reported `clean=false critical=0 high=4 total=5`.
  - `git diff --check` passed.
  - Remediation focused tests passed:
    - `go test ./internal/memory/recall ./internal/memory ./internal/memory/extractor -count=1`
    - `go test ./internal/api/core -run 'TestMemoryHandlersAndHelpers|TestMemoryExtractorHandlersUseInjectedService|TestMemoryProviderHandlersUseInjectedService|TestMemorySessionLedgerHandlersUseInjectedService' -count=1`
    - `go test ./internal/daemon -run 'TestDaemonNativeTools' -count=1`
    - `go test ./internal/tools ./internal/cli -count=1`
    - `go test -race ./internal/memory/recall ./internal/memory ./internal/memory/extractor ./internal/api/core ./internal/daemon ./internal/tools ./internal/cli -count=1`
  - `make lint` passed with `0 issues`.
  - `make verify` passed: Bun 334 files / 2150 tests, Go `DONE 8393 tests in 90.274s`, package boundaries OK.
  - Post-remediation `check-rounds-clean` reported `clean=true critical=0 high=0 total=5`.

## Patterns Observed

- CodeRabbit findings are concentrated in Memory v2 internals so far: prompt input bounding, extractor replay idempotence, recall signal merging, error-specific tests, and small correctness issues around path/metadata handling.
- Manual review round 001 fixes landed in five concrete areas: async recall signal recording, native sub-agent write denial, Slice 1 API parity handlers, decision idempotency for DLQ replay, and extractor/tool-write mutual exclusion.
