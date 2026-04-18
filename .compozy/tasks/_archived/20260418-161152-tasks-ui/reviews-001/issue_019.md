---
status: resolved
file: internal/observe/observer.go
line: 95
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb3,comment:PRRC_kwDOR5y4QM65B8fV
---

# Issue 019: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid hardcoded dashboard thresholds in observer defaults.**

These values should be injected from TOML/functional options instead of literals, so operators can tune behavior per environment.

As per coding guidelines, "Never hardcode configuration — use TOML config or functional options".


Also applies to: 232-236

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/observer.go` around lines 91 - 95, The struct
taskDashboardConfig currently uses hardcoded threshold literals; change it to
accept values from configuration or functional options by exposing these fields
as settable (or adding a constructor like NewTaskDashboardConfig(cfg *Config) or
NewTaskDashboardConfig(opts ...TaskDashboardOption)) and wire it into observer
construction so callers supply backlogWarnAfter, staleAfter and activeRunLimit
from the TOML-loaded config or provided options; update any place that
instantiates taskDashboardConfig with literal values (including the other
occurrences noted) to call the new constructor or option setters so thresholds
are not hardcoded.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Confirmed. `observe.New` hardcodes dashboard thresholds directly into the observer instance, while the related task-health thresholds already have an option surface. I’ll expose a dashboard configuration option/constructor so these thresholds are caller-configurable via functional options instead of being locked to constructor literals.
