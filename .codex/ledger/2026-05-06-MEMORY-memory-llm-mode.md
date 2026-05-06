Goal (incl. success criteria):

- Verify and fix the config validation finding: `memory.controller.mode = "llm"` must be rejected when `memory.controller.llm.enabled = false`.
- Success means the issue is proven against current code, fixed minimally, covered by focused tests, and validated.

Constraints/Assumptions:

- Always prefix shell commands with `rtk`.
- Do not run destructive git commands (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`) without explicit user permission.
- Conversation in Brazilian Portuguese; code/tests/artifacts in English.
- Go runtime work requires `internal/CLAUDE.md`, `agh-code-guidelines`, `golang-pro`, and bug/test skills.
- `make verify` is the required final project gate unless explicitly blocked by unrelated pre-existing failures.

Key decisions:

- Treat the CodeRabbit item as a still-unverified bug until current `internal/config` code and tests prove it.
- Keep the fix localized to config validation unless investigation shows another validator path.
- The nearby `1656-1675` block is `MemoryControllerLLMConfig.Validate`, not a second `MemoryControllerConfig` validator; the guard belongs in the parent validator because it needs both `Mode` and `LLM.Enabled`.

State:

- Started 2026-05-06 in `/Users/pedronauck/Dev/compozy/agh3`.
- Loaded RTK, `internal/CLAUDE.md`, required Go/test/debugging/no-workaround skills, and relevant mem-v2 ledgers.
- Confirmed current code reproduces the issue: focused test returned `Validate() error = nil` for `mode="llm"` and `LLM.Enabled=false`.
- Complete; final gate passed after all code changes.

Done:

- Scanned existing ledger files for cross-agent awareness.
- Read relevant prior Memory v2 ledgers: `mem-v2`, `memory-contract`, `memv2-real-qa`.
- Added regression coverage in `internal/config/memory_v2_config_test.go`.
- Patched `MemoryControllerConfig.Validate` to reject `mode="llm"` when `LLM.Enabled=false`.
- Verified red/green behavior with the focused regression.
- `go test ./internal/config -count=1` passed: 581 tests.
- `go test -race ./internal/config -count=1` passed: 581 tests.
- `make lint` passed with `0 issues`.
- `git diff --check` passed.
- `make verify` passed with exit code 0.

Now:

- Final response.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-06-MEMORY-memory-llm-mode.md`
- `internal/config/config.go`
- `internal/config/memory_v2_config_test.go`
- `rtk go test ./internal/config -run TestMemoryV2ConfigValidationRejectsInvalidValues/Should_reject_llm_controller_mode_with_disabled_llm -count=1`
- `rtk go test ./internal/config -count=1`
- `rtk go test -race ./internal/config -count=1`
- `rtk make lint`
- `rtk git diff --check`
- `rtk make verify`
