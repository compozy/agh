# QA And Verification

## Contents

- Test decision
- Existing-suite preference
- Real-system confidence
- Command selection
- Final verification
- QA labs

## Test Decision

Every AGH task needs a test decision, not automatic new tests. Before adding or changing tests, state:

- invariant
- owning layer
- canonical suite
- verification command

If no existing suite owns the invariant, create the smallest appropriate suite and explain why. If existing gates already prove the behavior, record a no-new-test rationale.

## Existing-Suite Preference

Prefer updating existing suites. Avoid duplicate regression coverage across layers unless each layer proves a distinct failure mode.

Do not add tests that only freeze implementation details, static prose, generated output, file existence, snapshots, CSS literal values, or config shape unless the artifact itself is the product contract.

## Real-System Confidence

Mocks are acceptable at boundaries not owned by the test, but final validation should exercise real wiring for the behavior being changed. Contract tests bridge unit tests and end-to-end tests. Real systems gate release confidence.

For task, session, network, or provider work, prefer tests that prove daemon behavior through the public surface or owning service, not prompt text alone.

## Command Selection

Match command scope to the claim:

- Narrow Go package behavior: targeted go test ./path.
- Bun workspace behavior: Turbo-backed bunx turbo run test --filter=...
- Site docs type/build behavior: site source generation and Turbo-backed test/typecheck.
- Whole task completion: make verify unless the change is docs-only and explicitly exempt.

Do not use package-local Bun test commands as validation evidence when repo rules require Turbo.

## Final Verification

Before claiming completion:

1. Identify what command proves the claim.
2. Run it fresh after the final change.
3. Read exit code and output.
4. Report failures honestly and fix root causes.
5. Re-run after fixes.

Passing tests are evidence only for what they cover. Audit the original requirements against artifacts before declaring the task done.

## QA Labs

For release or scenario QA, use isolated AGH homes, daemon ports, and bridge sockets. Reuse a QA lab only when continuing the same active QA session with a matching manifest. Fresh independent QA passes should use fresh labs.

Persist lab root, runtime home, base URL, and verification evidence when a QA pass creates a bootstrap manifest.
