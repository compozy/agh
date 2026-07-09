# Analysis: {competitor_name}

Read-only exploration of `.resources/{competitor_name}/` for the AGH task `{task_slug}`. Cross-referenced with the relevant TechSpec / `_idea.md`.

## Scope

- Path explored: `.resources/{competitor_name}/`
- Topic: {one-line topic the orchestrator passed}
- Files read in full vs. sampled:
- Total available files:

## Overview

(2-4 paragraphs)

What does this system do? What is its scope? Where does it overlap with AGH's current or planned behavior? What is its license / governance model if relevant?

## Mechanisms / Patterns

(Bulleted; each item names the pattern + the mechanism)

- **Pattern name:** what it does, where in the code, why it matters.
- ...

## Relevant Code Paths

(Cite files and line ranges that implement the patterns above. Real paths, not paraphrased.)

- `.resources/{competitor_name}/path/to/file.ts:NNN-NNN` — purpose
- `.resources/{competitor_name}/path/to/file.go:NNN-NNN` — purpose
- ...

## Transferable Patterns

(Patterns that map cleanly to AGH's architecture. Each item: what to take, where it would live in AGH, what it replaces/augments.)

- {Pattern name} → applies to `internal/{package}` because {reason}.
- ...

## Risks / Mismatches

(Patterns that look attractive but conflict with AGH constraints — composition root, hooks-not-bus, ClaimNextRun exclusivity, greenfield-no-compat, etc.)

- {Pattern name} would violate {AGH rule} because {reason}.
- ...

## Open Questions

(Things this analysis can't resolve. The orchestrator decides.)

- ...

## Evidence

(Final list of file paths cited above, deduplicated. The implementation agent will read these directly.)

- `.resources/{competitor_name}/...`
- ...
