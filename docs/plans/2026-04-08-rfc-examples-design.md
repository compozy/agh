# RFC Examples Design

## Goal

Enrich `docs/rfcs/003_agh-network-v1.md` with more illustrative, real-feeling examples without weakening its normative character.

## Approved Direction

Use a hybrid approach:

1. Add compact inline JSON examples at the most important protocol points.
2. Add a dedicated informative appendix with two end-to-end worked examples.

## Why This Shape

- Inline examples reduce lookup friction for core message kinds that are easy to misunderstand in the abstract.
- A dedicated worked-examples section keeps the main RFC readable while still showing realistic flows across multiple message kinds.
- This avoids turning every normative subsection into a tutorial while still making the protocol feel concrete.

## Planned Insertions

### Inline examples

- Under `9.5 direct`: one envelope showing a peer opening a targeted interaction with `interaction_id`, `reply_to`, and correlation fields.
- Under `9.6 recipe`: one envelope showing a portable recipe artifact with realistic metadata.

### Informative appendix

Add `Appendix A. Worked Examples` with two scenarios:

1. Space-visible request followed by direct handoff and lifecycle completion.
2. Recipe advertisement followed by direct follow-up and recipe use.

Each scenario should include:

- a short narrative
- one Mermaid sequence diagram
- selected envelope JSON examples, not every possible message
- a short note explaining what the example illustrates normatively

## Non-Goals

- No transport tutorial beyond the already-defined NATS profile.
- No attempt to define new message kinds or new normative requirements.
- No giant appendix of exhaustive permutations.

## Validation

- Ensure examples use fields already defined in section 6.1.
- Ensure message kinds and lifecycle states stay consistent with sections 8, 9, and 10.
- Run `make verify` after editing.
