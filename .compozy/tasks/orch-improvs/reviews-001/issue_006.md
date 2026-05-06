---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/bridges/task_notifier.go
line: 349
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:f5af1cc8e237
review_hash: f5af1cc8e237
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 006: Redact task errors before building the bridge notification.
## Review Comment

`run.Error` is copied verbatim into `TerminalTaskNotification.Error`, and that field is then sent in both `ProviderMetadata` and the rendered message text. Any raw claim token in the task error can leak over the bridge transport from this path.

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`) ... MUST NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory — use hash forms (`claim_token_hash`) over the wire".

## Triage

- Decision: `valid`
- Notes:
  - `resolveTerminalTaskNotification` still copies `run.Error` verbatim into `TerminalTaskNotification.Error`.
  - That value is later sent over bridge delivery metadata/rendered text, so raw claim tokens can escape through bridge transports.
  - Planned fix: redact claim tokens at notification construction time and cover the replay path with a notifier regression test.
  - Resolved: terminal bridge notifications now redact claim tokens with `taskpkg.RedactClaimTokens`, and a regression test asserts the delivered payload no longer leaks the raw claim token.
