---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/daemon/model_catalog.go
line: 18
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6sj,comment:PRRC_kwDOR5y4QM6-6bsN
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Decouple the daemon refresh deadline from the `models.dev` timeout.**

`sourceTimeout` is parsed from `state.cfg.ModelCatalog.Sources.ModelsDev` and then reused for the models.dev HTTP client, the default timeout passed into provider-live discovery, and the overall `modelCatalogRuntime` refresh deadline. That makes tuning one upstream source change unrelated discovery paths and can cause whole-catalog refreshes to time out for reasons that are hard to predict from config. Please introduce a catalog-wide refresh timeout and keep per-source timeouts scoped to the individual source instead of falling back to the hardcoded `10s` default here.

As per coding guidelines, "Never hardcode configuration values in Go code — use configuration files or environment variables instead."
 


Also applies to: 232-245

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/model_catalog.go` at line 18, The code currently uses
sourceTimeout (parsed from state.cfg.ModelCatalog.Sources.ModelsDev) and the
hardcoded constant defaultModelCatalogRefreshTimeout when setting timeouts for
both the models.dev HTTP client, provider-live discovery, and the
modelCatalogRuntime refresh deadline; decouple these by adding a new
configurable catalog-wide timeout (e.g., cfg.ModelCatalog.RefreshTimeout or env
var) and use that for modelCatalogRuntime while keeping sourceTimeout only for
ModelsDev-related HTTP client and provider-live discovery; remove reliance on
the hardcoded defaultModelCatalogRefreshTimeout in model_catalog.go and ensure
modelCatalogRuntime uses the new catalog-wide setting (leave sourceTimeout
scoped to ModelsDev and referenced only in the code paths that create the
models.dev client and provider-live discovery).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - This asks for a new catalog-wide timeout configuration that is not part of the approved `provider-model-catalog` config contract.
  - The current TechSpec explicitly defines source-level timeouts and states that provider discovery defaults to the model catalog source timeout; boot then passes that configured timeout into the runtime on purpose.
  - In production boot the runtime does not rely on the hardcoded `10s` default; the constant only backs direct constructor callers and tests when no timeout is supplied.
  - Implementing a new `model_catalog.refresh_timeout` would be a design expansion, not a remediation of a current defect in the scoped files.
