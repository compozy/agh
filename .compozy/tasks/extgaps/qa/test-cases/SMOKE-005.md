# SMOKE-005: make verify passes with all bundle code

**Priority:** P0 (Critical)
**Type:** Smoke
**Component:** Build pipeline

## Test Steps

1. Run `make fmt`
   **Expected:** No formatting changes needed

2. Run `make lint`
   **Expected:** Zero golangci-lint issues

3. Run `make test`
   **Expected:** All tests pass with -race flag

4. Run `make build`
   **Expected:** Binary compiles successfully

5. Run `make verify`
   **Expected:** All four steps pass in sequence
