# Task Memory: task_02

## Objective Snapshot

- Provide one shared, fence-preserving outbound chunker and wire provider-safe terminal continuation behavior through Slack, Telegram, Discord, Teams, Google Chat, and WhatsApp.
- Convert common Markdown constructs to Slack mrkdwn and Telegram MarkdownV2 before measuring provider limits, with a lossless plain-text fallback for Telegram parse failures.

## Important Decisions

- `internal/bridgesdk/chunk_test.go` owns boundaries, indicators, fence repair, UTF-16 measurement, and lossless reconstruction. Provider suites own API ordering, routing, dialect, and final ACK identity.
- Provider limits measure the actual wire representation: Slack/Discord/Teams/WhatsApp use Unicode code points, Telegram uses UTF-16 code units, and Google Chat uses UTF-8 bytes.
- Cumulative non-terminal overflow remains one mutable preview on edit-capable providers; only a terminal event materializes continuations. WhatsApp remains append-only because the Cloud API cannot edit text messages.
- Explicit `DeliveryOperationEdit` and `event.Reference` take precedence over state-based create/update inference in every edit-capable adapter.
- A nominally successful provider response without its remote message ID is transient and cannot advance delivery state or produce an ACK.
- Formatting happens before chunking. Provider-specific indicator escaping remains inside the final wire limit.

## Learnings

- Google Chat's current official `spaces.messages.create` cap is 32,000 bytes; the vendored Hermes 4,096-character value is stale.
- Slack must protect an unterminated streaming fence through end-of-input; otherwise Markdown inside code is transformed before the chunker repairs the fence.
- Telegram's reactive plain fallback must remove formatting markers only outside fenced and inline code. A global marker stripper corrupts literal `_`, `*`, and `~` in code.
- Horizontal review across all six adapters is necessary: state-first dispatch independently reproduced the same explicit-reference bug in Slack, Telegram, Discord, Teams, and Google Chat, while WhatsApp lost the replacement handle.

## Files / Surfaces

- Shared SDK: `internal/bridgesdk/chunk.go`, `internal/bridgesdk/chunk_test.go`.
- Shared dialect corpus: `internal/testutil/bridgeformat/corpus.go`.
- Provider delivery modules and canonical tests under `extensions/bridges/{slack,telegram,discord,teams,gchat,whatsapp}/`.
- Slack/Telegram pure formatters: `format.go` and `format_test.go` in each provider.
- Public behavior: six provider READMEs, `skills/agh/references/runtime-operations.md`, and QA scenario `NB-long-bridge-replies`.

## Errors / Corrections

- RED proved Slack transformed `**bold**` inside an open code fence; the fence matcher now protects balanced and unterminated regions while resuming conversion after a balanced fence.
- RED proved explicit edits could route to create when local state was empty. All edit-capable provider dispatchers now honor the operation/reference first; WhatsApp preserves the explicit replacement handle on its append-style edit.
- RED proved Slack/Telegram could accept empty remote IDs and Telegram's chunked fallback could corrupt code literals. Typed transient validation and code-aware fallback fixed both at the provider boundary.
- Documentation initially overstated restart safety and contained stale Discord Slack terminology. Claims now describe only the broker-recorded remote handle; durable partial-success recovery remains Task 06.

## Verification Evidence

- Fresh focused `CGO_ENABLED=1 go test -race -count=1 -cover` passed: bridgesdk 80.9%, Slack 82.1%, Telegram 81.1%, Discord 81.9%, Teams 80.0%, GChat 81.0%, WhatsApp 80.3%.
- Fresh scoped `golangci-lint` over the same seven packages returned `No issues found`.
- New chunk/formatter and modified clean provider-delivery suites pass the AGH test-shape checker; new cases added to legacy provider suites follow `Should ...` subtests with `t.Parallel()`.
- `git diff --check` is clean; all nine new production files are below 500 lines; legacy WhatsApp splitter symbols have zero occurrences.
- The living scenario `NB-long-bridge-replies` remains `untested` for the QA tail; `state.csv` is now only a generated ignored view.
- Per user direction, no global suite or `make verify` ran. The single global gate remains deferred until all tasks, QA, and Phase D remediation are complete.
- Safe scoped checkpoint `b103482` contains only Task 02 files; the two unrelated modified Daytona sidecar assets remain unstaged.

## Ready for Next Run

- Task 03 should reuse the Slack/Telegram formatter seams and the exact provider measurement units when rendering progress lines.
- Task 06 must persist ordered remote IDs and a partial-success cursor for multichunk delivery; the current single final ID cannot prevent prefix duplication after a later-chunk failure/restart.
- Task 09 must reconcile this behavior into the conflict-resistant living QA tree without minting a duplicate scenario.
- Per-task implementation peer review remains deferred to Phase D by `cy-loop-tasks`.
