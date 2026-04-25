---
status: resolved
file: internal/daemon/daemon.go
line: 559
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4175824057,nitpick_hash:ccf986f6bbb4
review_hash: ccf986f6bbb4
source_review_id: "4175824057"
source_review_submitted_at: "2026-04-25T16:07:33Z"
---

# Issue 006: Make the agent probe interval configurable.
## Review Comment

`2*time.Second` bakes daemon polling behavior into the composition root, so operators and tests cannot tune it without a code change. Please source this from `deps.Config.Observability` or an option instead of hardcoding it.

As per coding guidelines, `Never hardcode configuration — use TOML config or functional options`.

## Triage

- Decision: `valid`
- Root cause: `applyObserverFactoryDefault` hardcodes the agent probe timeout as `2*time.Second`, so the composition root cannot be tuned through runtime configuration.
- Fix approach: add a minimal out-of-scope config field under `internal/config` for observability agent probe timeout, default it to two seconds, validate it, merge TOML overlays, and pass that configured value into `observe.WithAgentProbeSource`.
- Scope note: `internal/config/*` is outside the batch code-file list but is required to satisfy the review's configuration-source requirement without introducing a daemon-only test option that operators cannot use.
