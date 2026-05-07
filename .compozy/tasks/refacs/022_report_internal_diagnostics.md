# Refacs 022: `internal/diagnostics`

## Scope

- Package: `github.com/pedronauck/agh/internal/diagnostics`
- Iteration: 022
- Goal: deep refactoring and performance audit for diagnostic redaction, dynamic secret registration, and bounded diagnostic output.
- Subagents:
  - Read-only refactoring audit for `internal/diagnostics`.
  - Read-only performance/concurrency audit for `internal/diagnostics`.

## Baseline

Initial package state:

```bash
rtk go test ./internal/diagnostics -count=1
rtk golangci-lint run ./internal/diagnostics
rtk proxy go test ./internal/diagnostics -cover -count=1
rtk proxy go test ./internal/diagnostics -run '^$' -bench . -benchmem -count=3
```

Observed baseline:

- Package tests: `10 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `75.6% of statements`.
- The package had no committed benchmarks.
- After adding package-local benchmarks and before production optimization:
  - `BenchmarkRedactStaticSecrets`: `9270-9413 ns/op`, `1652-1653 B/op`, `22 allocs/op`.
  - `BenchmarkRedactDynamicSecrets`: `9768-9902 ns/op`, `1791-1794 B/op`, `17 allocs/op`.

## Findings

### P1: Static redaction missed AGH-critical composite secret keys

The redaction taxonomy relied on generic `token` / `secret` terms plus a few explicit token names. Because `_` is a word character in Go regexp word boundaries, those generic terms did not match composite keys such as `claim_token`, `lease_token`, `client_secret`, `oauth_client_secret`, `webhook_secret`, or `bot_token`.

Impact: diagnostic text containing `claim_token=...`, JSON `{"claim_token":"..."}`, or OAuth client secrets could leak material that AGH security rules require to be redacted.

### P1: `token:` assignments were not redacted consistently

`token=...` was handled by a special regex, while `token:...` was not. This made redaction behavior inconsistent across CLI/API error text and JSON-ish diagnostics.

Impact: colon-delimited token assignments are common in free-form error output and should be treated the same as equals-delimited assignments.

### P1: Caller smoke exposed a double-redaction regression risk

After broadening `token:` redaction, a CLI caller smoke test showed that already protected `agh_claim_[REDACTED]` markers could be redacted again when the surrounding text said `claim token: ...`.

Impact: diagnostics must remove raw secret material without destroying canonical redacted claim-token markers that downstream contracts assert.

### P2: Dynamic secret redaction rebuilt ordering on the read path

The original dynamic registry built and sorted a snapshot on every `Redact` call. Dynamic registrations are writes; diagnostic redaction is the read-heavy path.

Impact: every redaction paid unnecessary allocation/sort work when dynamic secrets were registered.

### P2: A lock-held redaction loop would block secret registration/cleanup

A simple ordered-slice cache avoids per-call sorting, but holding the registry read lock while running `strings.ReplaceAll` over large diagnostic text can block writers for the full redaction duration.

Impact: long crash/stderr payloads and high dynamic-secret cardinality can create registration/cleanup contention.

### P2: `RedactAndBound` could split UTF-8 code points

`RedactAndBound` caps by byte budget, which is correct for storage limits, but raw byte slicing can cut through a multibyte rune.

Impact: invalid UTF-8 can make JSON/HTTP/UI diagnostic rendering less deterministic even when byte budgets are respected.

## Changes Made

### Redaction correctness

- Centralized static sensitive-key policy into one regex alternation.
- Added explicit AGH and provider secret keys:
  - `claim_token`
  - `lease_token`
  - `client_secret`
  - `oauth_client_secret`
  - `webhook_secret`
  - `bot_token`
- Folded generic `token` into the normal assignment regex so `token:` and `token=` follow the same path.
- Replaced simple assignment replacement with `redactSecretAssignments`, preserving values already containing `[REDACTED]` or protected redaction sentinels.
- Added regression coverage for raw composite keys, colon-delimited tokens, benign token text, and already redacted claim markers.

### Dynamic secret registry

- Replaced per-redaction dynamic snapshot construction with an immutable ordered snapshot stored in `atomic.Value`.
- Kept a mutex around registration counts and snapshot writes.
- Rebuilds and sorts the ordered snapshot only when a secret is newly added or fully removed.
- Dynamic redaction now reads the immutable snapshot without holding the registry lock while scanning/replacing text.

### Bounded output

- Extracted `truncationSuffix`.
- Added `truncateUTF8WithinBytes` so `RedactAndBound` remains within the byte budget while backing up to a UTF-8 rune boundary.

### Tests and benchmarks

- Added `redact_bench_test.go` with:
  - `BenchmarkRedactStaticSecrets`
  - `BenchmarkRedactDynamicSecrets`
- Expanded `redact_test.go` to cover:
  - composite AGH secret keys;
  - `token:` assignment redaction;
  - benign non-assignment token text;
  - already redacted claim-token markers;
  - truncation marker behavior;
  - UTF-8-safe truncation;
  - dynamic duplicate refcount cleanup;
  - longest-first dynamic secret ordering;
  - blank/short dynamic secret no-ops.

## Performance Results

Final focused benchmark command:

```bash
rtk proxy go test ./internal/diagnostics -run '^$' -bench 'BenchmarkRedact(Static|Dynamic)Secrets$' -benchmem -count=5
```

Observed final results:

- `BenchmarkRedactStaticSecrets`: `9872-9974 ns/op`, `1467-1469 B/op`, `20 allocs/op`.
- `BenchmarkRedactDynamicSecrets`: `10178-10287 ns/op`, `944-946 B/op`, `11 allocs/op`.

Interpretation:

- The final static path handles a larger sensitive-key taxonomy and protects already-redacted values. It is not claimed as a CPU-speed win; the correctness fix is the priority.
- The dynamic path now avoids per-redaction snapshot/sort and lock-held replacement. Final allocations dropped from the first measured baseline of `1791-1794 B/op` and `17 allocs/op` to `944-946 B/op` and `11 allocs/op`.
- The performance subagent found regex replacement dominates CPU/allocation profiles. A broad regex-to-scanner rewrite was intentionally deferred because redaction is a security boundary and would need a larger golden/fuzz equivalence harness.

## Deferred / Cross-Package Notes

- Several callsites discard the cleanup returned by `RegisterDynamicSecret`, especially in MCP, daemon settings/bridge secret, and CLI auth paths. Some daemon-lifetime secrets may intentionally remain registered, but per-call MCP registrations should be revisited in their owning package iterations.
- A broader static redaction rewrite should wait for representative no-secret, 32 KiB crash payload, registry-cardinality, prefix-overlap, and parallel contention benchmarks plus a golden/fuzz equivalence harness.
- `RedactAndBound` still redacts before bounding. Pre-bounding crash evidence is risky because a secret can cross the truncation boundary; defer unless whole-daemon profiling proves crash payload redaction is a hot path.

## Validation

Final validation commands:

```bash
rtk go test ./internal/diagnostics -count=1
rtk golangci-lint run ./internal/diagnostics
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/diagnostics/redact_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/diagnostics/redact_bench_test.go
rtk proxy go test ./internal/diagnostics -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/diagnostics -count=20
rtk go test -tags integration ./internal/diagnostics -count=1
rtk proxy go test ./internal/diagnostics -run '^$' -bench 'BenchmarkRedact(Static|Dynamic)Secrets$' -benchmem -count=5
rtk go test ./internal/session ./internal/api/core ./internal/api/contract ./internal/soul ./internal/heartbeat ./internal/mcp ./internal/cli -run 'Redact|redact|Secret|secret|Failure|Diagnostic|MCPAuth' -count=1
rtk rg -n "_\\s*=|context\\.Background\\(|strings\\.Contains\\(.*err\\.Error\\(\\)" internal/diagnostics --glob '*.go'
rtk make verify
```

Observed final results:

- Package tests: `28 passed in 1 packages`.
- Package lint: no issues.
- AGH test-shape checks: passed for `redact_test.go` and `redact_bench_test.go`.
- Package coverage: `90.5% of statements`.
- Race package tests with `-count=20`: passed.
- Integration-tag package tests: `28 passed in 1 packages`.
- Focused benchmarks: passed with the final measurements above.
- Caller smoke package set: `96 passed in 7 packages`.
- Scoped production/test scan for ignored errors, `context.Background`, and `strings.Contains(err.Error())`: no matches.
- `make verify`: passed.
