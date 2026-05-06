# TC-REG-001: Generated Contracts, CLI References, Site Docs, And Memory Drift

**Priority:** P1

**Objective:** Prove generated contracts and public documentation match implemented runtime
surfaces, and that lessons/glossary updates do not preserve false or obsolete claims.

**Requirements Covered:** tasks 18-19, 21, 23-30; ADR-001 through ADR-010.

## Preconditions

- Current working tree includes all orchestration-improvements implementation files.
- Generated OpenAPI, TypeScript, and CLI reference docs are present.
- Site docs source and generated source maps are available.

## Test Steps

1. Run `make codegen-check`.
   **Expected:** No OpenAPI or generated TypeScript drift is reported.

2. Run `make cli-docs` or the project-equivalent generated CLI reference check.
   **Expected:** `agh task profile` and `agh task review` docs match current Cobra output.

3. Run site source/content generation, typecheck, tests, and build.
   **Expected:** Site navigation includes execution profiles, review gate, and notification cursors;
   docs build without broken generated content.

4. Search site docs for disallowed stale claims.
   **Expected:** No docs claim public cursor reset, web verdict submission, channel-owned verdicts,
   old bridge route aliases, review statuses of `approved|rejected|blocked`, or PUT-as-patch
   profile updates.

5. Search generated contracts and web code for duplicated DTO definitions.
   **Expected:** Web orchestration types derive from generated OpenAPI types through the existing
   contract helpers.

6. Check `docs/_memory/lessons/` and `docs/_memory/glossary.md`.
   **Expected:** Lessons cite concrete evidence and glossary names match current canonical terms.

## Behavioral Evidence

- Command outputs for codegen, site generation, site tests, and build.
- Search results proving forbidden stale claims are absent.
- Links or file paths for generated CLI references and site pages checked.
- Memory lesson and glossary paths checked.

## Disruption Probes

- Remove a generated DTO field locally during task 32 only if needed for a controlled drift probe,
  then regenerate or revert through normal editing before final gates.
- Compare docs examples with live CLI JSON output from the QA lab.

