# TC-SEC-006: Extension Reads Filtered by Granted Kinds

**Priority:** P0
**Type:** Security
**Package:** internal/extension
**Related Tasks:** 04, 05

## Objective

Validate that an extension can only read resource records of kinds explicitly granted in its manifest. Attempting to list or access records of unauthorized kinds must return an empty result or 403, with no information leakage about the existence of those records.

## Preconditions

- Extension `ext-tools-only` is registered with granted kinds: `["tool"]`.
- Extension `ext-hooks-only` is registered with granted kinds: `["hook.binding"]`.
- The resource store contains records of kinds: `tool`, `hook.binding`, `prompt.template`, and `config`.
- Both extensions have active sessions with valid nonces.

## Test Steps

1. As `ext-tools-only`, call `resources/list` with no kind filter.
   **Expected:** Response contains only records of kind `tool`. No records of kind `hook.binding`, `prompt.template`, or `config` appear.

2. As `ext-tools-only`, call `resources/list` with an explicit filter for `kind=hook.binding`.
   **Expected:** Response is an empty list (or 403 Forbidden). The response does not reveal how many `hook.binding` records exist.

3. As `ext-tools-only`, call `resources/get` for a specific `(hook.binding, known-hook-id)` record.
   **Expected:** 403 Forbidden. The error response does not confirm or deny the existence of the record.

4. As `ext-hooks-only`, call `resources/list` with no kind filter.
   **Expected:** Response contains only records of kind `hook.binding`. No `tool` records are visible.

5. As `ext-tools-only`, call `resources/list` with a wildcard or regex-style kind filter (e.g., `kind=*` or `kind=hook.*`).
   **Expected:** The wildcard is either rejected as invalid or interpreted strictly -- only matching granted kinds are returned. No unauthorized kinds leak through pattern matching.

## Edge Cases

- Extension granted an empty kinds list (`[]`) attempts to list any records.
- Extension sends a request with the kind field set to a SQL injection payload (e.g., `kind=tool' OR '1'='1`).
- Extension granted `tool` kind attempts to read `tool.deprecated` (kind prefix matching).
- Kind filter uses URL encoding or unicode normalization tricks to bypass string matching.
- Extension's granted kinds are updated at runtime (revocation scenario) -- reads after revocation must reflect the new grants immediately.

## Threat Model

This test prevents **unauthorized resource enumeration via kind escalation**. The kind-grant model is the primary mechanism for limiting what categories of resources an extension can discover. Without strict enforcement, an extension granted only `tool` access could enumerate hook bindings to understand the system's event routing, read prompt templates to extract proprietary prompts, or access configuration records to discover sensitive settings. Kind-level filtering is the access control layer that implements the principle of least privilege for extension reads.
