# Systematic Project QA Checklist

Mark every item as complete before claiming the QA pass is done.

## Contract Discovery

- [ ] Root instructions and repository docs were read
- [ ] The canonical verify gate was identified or an explicit fallback was chosen
- [ ] The changed surface and regression-critical surface were identified
- [ ] Web UI surface presence was determined (yes/no with evidence)

## Baseline

- [ ] Dependencies were installed with the repository-preferred command
- [ ] The baseline verification gate was run before scenario testing
- [ ] Verification order followed fastest-first: lint, build, unit tests, integration tests
- [ ] Any pre-existing failures were isolated with evidence
- [ ] Dev server was started and confirmed ready (when Web UI surface exists)

## CLI and API Validation

- [ ] Changed workflows were exercised through public interfaces
- [ ] At least one unchanged regression-critical workflow was exercised
- [ ] Runtime readiness was confirmed with observable signals
- [ ] Scenario fixtures or disposable projects were realistic and minimal

## Behavioral Real Scenario Validation

- [ ] 2-4 high-risk operator/agent journeys were identified before low-level checks
- [ ] Each journey names operator intent, expected business outcome, AGH surfaces, expected agent behavior, expected artifacts, and realistic disruption probes
- [ ] The scenario contract was read when present, and the execution matrix satisfies its minimum agents, channels, task tree, provider, artifact, disruption, and cross-surface requirements
- [ ] The behavioral charter is filled as structured JSON-compatible YAML
- [ ] The journey log has one row for every meaningful CLI/API/Web/runtime/provider action
- [ ] The provider attempt file records live proof or an exact blocked boundary
- [ ] At least one provider-backed agent session was exercised when credentials and local prerequisites were available
- [ ] Blocked live provider/LLM validation names the exact credential, provider, or tool boundary
- [ ] Agent behavior was verified through AGH state, not only through terminal text
- [ ] Produced artifacts were inspected, coherent, connected to their producing session/task/channel, and used later in the scenario
- [ ] At least one realistic disruption probe was executed and recorded
- [ ] Smoke, CRUD, page-render, unit, integration, mock, and fake-provider checks were not counted as final behavioral proof
- [ ] The strict QA auditor was run and produced `qa-audit-report.json`
- [ ] Auditor blockers resulted in `FAIL` or `BLOCKED`, not `PASS`

## Web UI Validation

Skip this section if the project has no Web UI surface.

- [ ] Critical user flows were identified (3-5 flows covering changed and business-critical surfaces)
- [ ] Each flow followed the open/snapshot/interact/re-snapshot/verify loop
- [ ] Screenshots were captured at each verification checkpoint
- [ ] Browser flows used real scenario state and proved the operator can understand and act on it
- [ ] Form flows were tested with both valid and invalid data
- [ ] Navigation flows were verified (page transitions, deep links, 404 handling)
- [ ] Error and loading states were triggered and verified
- [ ] Responsive behavior was tested at relevant viewports (when in scope)
- [ ] Authentication flow was exercised or state was loaded (when applicable)
- [ ] Browser session was closed after all flows completed

## Regression Handling

- [ ] Every failure was reproduced before fixing
- [ ] Root cause was identified before implementation
- [ ] Regression coverage was added or updated when the repository supported it
- [ ] The narrow repro and impacted flows were rerun after each fix
- [ ] Web UI regressions include before/after screenshot evidence (when applicable)

## Final Verification

- [ ] The full verification gate was rerun after the last code change
- [ ] The highest-risk behavioral journey was rerun after the final gate
- [ ] The most important CLI and API flows were rerun after the final gate
- [ ] The most important Web UI flows were rerun after the final gate (when applicable)
- [ ] A verification report was produced from fresh evidence
- [ ] The verification report includes an Audit Result section with command, exit code, JSON report path, blockers, warnings, and verdict
- [ ] Blocked scenarios or missing prerequisites were disclosed explicitly
