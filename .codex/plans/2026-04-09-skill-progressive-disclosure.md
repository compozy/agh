# Skill Progressive Disclosure

## Summary

- Split skills into metadata-first objects and explicit content reads.
- Keep prompt injection lightweight: only skill catalog metadata goes into agent context.
- Align backend, CLI, and web on the same explicit content-loading flow.

## Key Changes

- `internal/skills`
  - Remove eager `Content` retention from `skills.Skill`.
  - Keep load-time parsing/scanning of `SKILL.md`, but discard the body after verification.
  - Add lazy content loading for resolved skills, including filesystem and bundled skills.
- `internal/api`
  - Make `GET /api/skills` and `GET /api/skills/:name` metadata-only.
  - Add `GET /api/skills/:name/content` in HTTP and UDS.
  - Remove `content` from the main skill payload contract.
- `internal/cli`
  - Keep `agh skill view` as the explicit full-content reader, backed by lazy loading instead of preloaded `skill.Content`.
- `web`
  - Keep metadata/detail fetches lightweight.
  - Load full skill content only when the user clicks `View full content`.
  - Show loading/error/success states in the content card.

## Tests

- Verify registry load paths do not retain skill bodies.
- Verify lazy content reads for bundled and filesystem skills.
- Verify list/detail handlers are metadata-only and the new content endpoint returns the body.
- Verify CLI `skill view` still renders full content and resources.
- Verify web adapters/hooks/components only fetch content on explicit click.

## Assumptions

- Breaking `content` out of the generic skill payload is acceptable in this greenfield codebase.
- The new explicit-read endpoint covers only the main `SKILL.md` body; resource-file reads stay on the CLI path for now.
