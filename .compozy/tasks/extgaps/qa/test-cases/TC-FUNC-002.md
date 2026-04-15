# TC-FUNC-002: Bundle spec validation rejects invalid manifests

**Priority:** P0 (Critical)
**Type:** Functional
**Component:** `internal/extension/bundle.go` — `BundleSpec.Validate()`, `BundleProfile.Validate()`, etc.

## Objective

Validate that bundle spec validation catches all invalid configurations at extension load time, preventing corrupt bundles from entering the catalog.

## Preconditions

- Extension manifest available for validation context
- Bundle files in TOML or JSON format

## Test Steps

### Bundle-level validation

1. Bundle with empty name
   **Expected:** Error wrapping `ErrBundleInvalid`: "bundle.name is required"

2. Bundle with zero profiles
   **Expected:** Error: "must declare at least one profile"

3. Bundle with duplicate profile names
   **Expected:** Error: "profile X is duplicated"

### Profile-level validation

4. Profile with empty name
   **Expected:** Error: "profile[N].name is required"

5. Profile with channels declared but no primary
   **Expected:** Error: "must declare channels.primary"

6. Profile with primary channel not in items list
   **Expected:** Error: "primary channel X is not declared"

7. Profile with duplicate channel names
   **Expected:** Error: "channel X is duplicated"

### Job validation

8. Job with empty name
   **Expected:** Error: "job.name is required"

9. Job with duplicate name in same profile
   **Expected:** Error: "job X is duplicated"

10. Job referencing undeclared channel in task.network_channel
    **Expected:** Error: "references undeclared channel"

### Trigger validation

11. Trigger with empty name
    **Expected:** Error: "trigger.name is required"

12. Trigger with duplicate name in same profile
    **Expected:** Error: "trigger X is duplicated"

### Bridge validation

13. Bridge with empty display_name
    **Expected:** Error: "display_name is required"

14. Bridge with invalid routing_policy
    **Expected:** Error wrapping routing policy validation

15. Bridge with secret slot missing name
    **Expected:** Error: "secret_slots.name is required"

16. Bridge with secret slot missing kind
    **Expected:** Error: "secret slot X kind is required"

17. Bridge with no extension_name, no platform, and manifest without bridge.adapter capability
    **Expected:** Error: "must declare extension_name or platform"

### File loading

18. Bundle file with .yaml extension (unsupported)
    **Expected:** Error: "unsupported bundle path"

19. Bundle file with invalid TOML syntax
    **Expected:** Parse error

20. Duplicate bundle names across files
    **Expected:** Error: "duplicate bundle"

## Edge Cases

- Whitespace-only names should be treated as empty
- Valid bundle with all optional fields omitted → passes
- Bundle with both root-level and `[bundle]` profiles → "conflicting root and bundle profiles"
