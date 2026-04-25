# BUG-006: Reference extension E2E used a symlinked SDK dependency rejected by path hardening

## Status

Fixed in Task 11.

## Severity

P1 integration-fixture regression. The reference extension E2E lane failed against the Task 05 symlink escape hardening, blocking real extension packaging validation.

## Reproduction

Run:

```bash
go test -tags integration ./internal/extension -run TestReferenceExtensionsEndToEnd
```

## Observed

Installing the reference `prompt-enhancer` extension failed because `node_modules/@agh/extension-sdk` was a symlink resolving outside the managed extension source root.

## Expected

Managed extension loading should continue rejecting dependency paths that escape the source root, while the reference E2E fixture should represent a packaged install source with dependency files materialized inside the install root.

## Root Cause

The test installed directly from the development workspace, where the SDK dependency is linked for local development. That is not a valid packaged extension source under the hardened loader.

## Fix

Changed `internal/extension/reference_integration_test.go` to build a temporary packaged install source for the prompt enhancer. The helper copies the extension manifest, package metadata, built `dist`, and a materialized `node_modules/@agh/extension-sdk` subset into the temp install root before installation.

## Verification Evidence

- Failing repro: `.compozy/tasks/hermes/qa/logs/final/failure-repro/extension-reference-e2e.log`
- Focused proof after fix: `.compozy/tasks/hermes/qa/logs/final/failure-repro/extension-reference-e2e-after-fix.log`
- Full integration after fixes: `.compozy/tasks/hermes/qa/logs/final/make-test-integration-after-fixes.log`
