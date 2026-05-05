# ADR-004: Add Minimal Explicit Task Orchestration Config

## Status

Accepted

## Date

2026-05-05

## Context

The orchestration hardening work introduces runtime limits and thresholds that should not be hardcoded:

- Summary body limits.
- Context bundle bounds.
- Prior-attempt and recent-event counts for context enrichment.
- Spawn-failure circuit breaker thresholds.
- Scheduler bad-tick telemetry thresholds.
- Default runtime watchdog behavior.
- Task execution profile defaults and gates for provider/model overrides and sandbox disabling.

The config surface should be explicit enough for operators and agents to inspect and manage, but narrow enough to avoid turning the MVP into a full policy engine.

The review-gate child spec adds a nested review policy surface. That surface belongs under the same `[task.orchestration]` config family because it controls task-run continuation behavior, context bounds, reviewer timeouts, and circuit breakers.

## Decision

Add a minimal `[task.orchestration]` config section with documented defaults:

```toml
[task.orchestration]
summary_max_bytes = 4096
context_body_max_bytes = 8192
context_prior_attempts = 5
context_recent_events = 50
spawn_failure_limit = 5
scheduler_bad_tick_threshold = 6
scheduler_bad_tick_cooldown = "5m"
default_max_runtime = "0s"

[task.orchestration.profile]
default_coordinator_mode = "inherit"
default_worker_mode = "inherit"
default_sandbox_mode = "inherit"
allow_task_provider_override = true
allow_task_sandbox_none = true

[task.orchestration.review]
default_policy = "none"
max_rounds = 3
max_review_attempts = 2
timeout = "20m"
rapid_terminal_window = "2m"
rapid_terminal_limit = 3
missing_work_max_items = 20
missing_work_item_max_bytes = 512
reason_max_bytes = 2048
review_text_max_bytes = 12000
next_round_guidance_max_bytes = 4096
failure_policy = "block_task"
```

Semantics:

- `summary_max_bytes` bounds `task_runs.summary` inputs from APIs, tools, and CLI.
- `context_body_max_bytes` bounds rendered task context bundle body sections.
- `context_prior_attempts` bounds prior run attempts in task context.
- `context_recent_events` bounds recent task/run events in task context.
- `spawn_failure_limit` controls the task-service-owned spawn-failure circuit breaker.
- `scheduler_bad_tick_threshold` and `scheduler_bad_tick_cooldown` govern scheduler health telemetry, not scheduler authority.
- `default_max_runtime = "0s"` disables a default task runtime watchdog. Individual tasks may override through `tasks.max_runtime_seconds`.
- `[task.orchestration.profile]` controls default task execution profile modes and gates task-level provider/model and sandbox disabling overrides.
- `[task.orchestration.review]` controls default review policy, round limits, reviewer attempt limits, timeout, rapid-terminal circuit breaking, verdict/guidance bounds, and failure policy. It defaults to `default_policy = "none"` so review gate is opt-in unless an operator changes config or task policy.

The config layer owns duration parsing. `default_max_runtime` is a TOML duration string at the config boundary, but task service/store code receives integer seconds. Validation rejects negative durations, fractional seconds, and values above `24h`.

The section must be surfaced through AGH's config lifecycle: defaults, validation, config docs, agent-operable CLI/HTTP/UDS read/update surfaces where the surrounding config system supports them, and tests.

## Consequences

### Positive

- Avoids hardcoded production behavior.
- Gives operators and agents a small, inspectable lifecycle surface.
- Keeps MVP boundaries clear by avoiding a general orchestration policy DSL.
- Makes review gate bounded and explicit instead of prompt-defined.
- Makes task-specific agent/provider/model/sandbox selection explicit without creating a broad policy engine.
- Supports deterministic test fixtures.

### Negative

- Adds config parsing, validation, docs, and generated surface work.
- Requires careful duration parsing and zero-value semantics.
- Requires downstream web/docs impact analysis in generated tasks.

### Risks

- Too many knobs can hide unclear product behavior. The TechSpec should keep this section narrow and avoid adding a programmable review policy DSL.
- If config writes are exposed, concurrent QA must respect AGH's rule against parallel config writes against one isolated home.

## Rejected Alternatives

### Hardcode all thresholds

Rejected because orchestration behavior affects runtime safety, context size, and operator expectations.

### Add a broad orchestration policy engine

Rejected because the MVP needs explicit defaults, not a programmable policy system.

### Put all settings under existing coordinator config

Rejected because several settings govern task service, context, scheduler telemetry, and notification behavior, not only coordinator bootstrap.

### Let every task override sandbox/provider behavior without config gates

Rejected because task-level overrides need operator-visible defaults, validation, docs, and tests. The MVP keeps the gates explicit under `[task.orchestration.profile]`.

## References

- `.compozy/tasks/orch-improvs/analysis/analysis.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-dashboard.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_task-execution-profile.md`
- `.compozy/tasks/orch-improvs/_techspec_review_gate.md`
- `internal/config/`
- `packages/site/`
