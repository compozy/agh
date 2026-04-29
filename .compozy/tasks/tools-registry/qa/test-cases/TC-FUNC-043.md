# TC-FUNC-043 — Public Go SDK builds without importing `internal/*`

- **Priority:** P1
- **Type:** Functional / Go SDK boundary
- **Trace:** Task 08, ADR-009

## Objective

Prove `sdk/go` is a public authoring surface and does not import `internal/*` from the AGH module. SDK consumers can import only `github.com/pedronauck/agh/sdk/go`.

## Test Steps

1. External-package test: build a Go fixture importing only `github.com/pedronauck/agh/sdk/go`.
   - **Expected:** Compiles.
2. Lint-time check: `go list -deps ./sdk/go/...` shows zero `internal/...` imports.
3. SDK fixture defines `aghsdk.Tool[InputT]` with `aghsdk.ToolOptions` and a handler; produces a registered tool.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `go test ./sdk/go/... -run TestNoInternalImports`
