---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/daemon/task_event_bridge_notifier.go
line: 14
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:0a2c7e46c80d
review_hash: 0a2c7e46c80d
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 015: Move the bridge notification timeout into config.
## Review Comment

This bakes an operator-facing runtime policy into code, so bridge delivery latency/backpressure cannot be tuned without recompiling. Please thread it through config or env and inject it here.

As per coding guidelines, "Never hardcode configuration values in Go code; always read from `config.toml` or environment variables".

## Triage

- Decision: `valid`
- Notes:
  - `task_event_bridge_notifier.go` still hardcodes the bridge notification timeout to `10s`.
  - That is an operator-tunable runtime policy, not a compile-time invariant, and it currently cannot be adjusted without a code change.
  - Planned fix: thread the timeout through daemon config and inject it into the bridge notification observer, documenting any minimal out-of-scope config file changes required.
  - Resolved: bridge notification timeout now lives in task orchestration config plus overlay/validation, and the daemon injects the configured value into the bridge observer.
