# Coverage matrix — surface → scenario → charter (charter-set owned)

This is the **authoritative, charter-set-owned** surface→scenario→charter map for the
model-selector cycle. Every charter (CH-028..CH-036) links here via its
`coverage_matrix:` field; the dated plan report (`reports/2026-07-10-model-selector-plan.md`)
summarizes it but is **not** its sole owner. Every public CLI/HTTP/UDS/native-tool/web/docs/config
surface named in `_spec §11–§12` maps to ≥1 `state.csv` scenario, ≥1 owning charter, a concrete
`must_try`, and an expected observable/evidence artifact.

| Surface | Entry point (real route/verb) | Scenario(s) | Charter | Owning `must_try` |
|---|---|---|---|---|
| Web · session-create | session-create dialog (RuntimeSelector) | RT-063, RT-064, RT-065, RT-066, RT-067, RT-010 | CH-028 | pick provider→model→effort→create; reset-on-switch; custom-ID; favorites; keyboard |
| Web · session-create a11y | keyboard + screen reader | RT-068 | CH-034 | Tab/Arrow/Enter/Esc; external favorite button name + aria-pressed; live announce |
| Web · agent-create | wizard RuntimeStep (+reasoning) | RT-069, RT-070 | CH-029 | pick runtime + default reasoning; provider-change clears model+reasoning; complete wizard |
| Web · onboarding | default-model step (grid deleted) | RT-071, RT-004 | CH-030 | pick provider·model·reasoning; provider-change credential reset; commit |
| Web · settings/display (canary) | `/settings/providers` inspect/edit/status; `/agents/:name` + task detail echoes | MS-028, MS-058 | CH-033 | inspect/edit provider; check session-list + task profile + logo echoes; Back+refresh |
| HTTP · list | `GET /api/model-catalog/providers/{id}/models?view=curated\|all` | MS-042, MS-053, MS-055 | CH-031 | GET curated then `view=all`; compare membership |
| HTTP · session create (+reasoning) | `POST /api/sessions` | RT-010, RT-061, RT-063 · fail-loud RT-062 | CH-028, CH-032, CH-035 | POST with provider/model/reasoning_effort; assert wire + 422 codes |
| HTTP · agent create | web wizard `POST /api/agents`; direct structured create (+reasoning_effort) | RT-069, RT-029 | CH-029, CH-036 | wizard POST plus direct create; 201; fresh GET read-back |
| HTTP · agent read | `GET /api/agents[/:name]` (projection) | RT-028 | CH-029 | GET list + detail; assert reasoning_effort projection; 404 on unknown/internal |
| HTTP · curate | `POST /api/model-catalog/providers/{provider_id}/models/curate` | MS-054 | CH-031 | curate flags; re-list to confirm membership |
| HTTP · model source status | `GET .../models/status` | MS-044 | CH-031 | GET status; assert source freshness |
| HTTP · openai-compat | `GET /api/openai/v1/models` | MS-045 | CH-031 | GET the OpenAI-compatible model list; assert canonical ids |
| UDS parity | same agent/catalog routes over UDS; session negotiation over UDS | MS-055, RT-028, RT-029, RT-062 | CH-031, CH-029, CH-035, CH-036 | run identical applicable calls over UDS; assert parity with HTTP and exact session error codes |
| CLI · provider models | `agh provider models list/set/refresh/status` | MS-042, MS-043, MS-054, MS-055, MS-044 | CH-031 | list `--all`; set flags; refresh (provider + global); status |
| CLI · agent | `agh agent create`; `agh agent list/info` | RT-029, RT-028 | CH-029, CH-036 | create through the structured CLI, then list/info — assert reasoning_effort round-trip |
| Native tools · provider models | `provider_models_list/curate/refresh/status` + registry/descriptor/capability/availability | MS-042, MS-054, MS-055, MS-043, MS-044, ET-049 | CH-031 | invoke all four; verify registry id, descriptor/schema digest, capability gate, availability diagnostics |
| Native tools · agent | `agh__agent_create` (create only — no native read/update tool) | RT-029 | CH-036 | invoke create; read-back via HTTP/UDS/CLI/AGENT.md (not native) |
| Reasoning apply (session) | ACP `set_config_option` ordering | RT-061 (happy) · RT-062, MS-057 (fail-loud) | CH-032, CH-035 | model→effort→prompt ordering; explicit none vs empty; fail-loud codes |
| Config lifecycle | `providers.<id>.models.reasoning`; curated flags | MS-056 | CH-031 | set config keys; confirm apply + persistence |
| Docs (site) | published model-catalog / providers / config-toml pages | ET-053 | CH-031 | fetch each page as an agent-readable public surface and follow the documented steps |
| Bundled AGH skill | `skills/agh/` (new verb, `view`, curation fields, reasoning error) | ET-053 | CH-031 | follow the skill to curate + read a model as an agent |

**No touched public surface is without:** an exact `state.csv` scenario, an owning charter, a concrete `must_try`, and an expected observable/evidence artifact.
