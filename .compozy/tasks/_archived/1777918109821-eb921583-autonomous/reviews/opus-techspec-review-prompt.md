# Review Request: Autonomous AGH TechSpec

You are reviewing the technical specification for AGH's autonomous agent system.

## Files to Review

- `.compozy/tasks/autonomous/_techspec.md`
- `.compozy/tasks/autonomous/adrs/adr-001.md`
- `.compozy/tasks/autonomous/adrs/adr-002.md`
- `.compozy/tasks/autonomous/adrs/adr-003.md`
- `.compozy/tasks/autonomous/adrs/adr-004.md`
- `.compozy/tasks/autonomous/adrs/adr-005.md`
- `.compozy/tasks/autonomous/adrs/adr-006.md`
- `.compozy/tasks/autonomous/adrs/adr-007.md`
- `.compozy/tasks/autonomous/adrs/adr-008.md`
- `.compozy/tasks/autonomous/adrs/adr-009.md`
- `.compozy/tasks/autonomous/analysis/analysis.md`

## Review Goal

Perform a senior architecture review before this TechSpec is decomposed into implementation tasks.

Focus on whether the design is:

- technically coherent with AGH's existing architecture;
- appropriately extensible through the existing hooks/resources/provider model;
- realistic to decompose into tasks;
- clear about scheduler/coordinator boundaries;
- safe around task claim/lease, spawn, permissions, and recovery;
- not over-engineered for the first autonomy implementation.

## Important Constraints

- Do not modify code.
- Do not modify the TechSpec or ADR files.
- Do not run destructive git commands.
- Treat AGH as greenfield alpha: backward compatibility is not a reason to reject a cleaner design.
- Prefer root-cause architectural feedback over wording nits.
- If you reference code, use concrete file paths.

## Output

Write a review file at:

`.compozy/tasks/autonomous/reviews/opus-techspec-review.md`

Use this structure:

1. Executive verdict: approve / approve with changes / request redesign.
2. Critical issues: blocking problems that should be fixed before task decomposition.
3. Major issues: important design gaps or sequencing risks.
4. Minor issues: clarity, naming, or scope improvements.
5. Over-engineering candidates: items that should be deferred or narrowed.
6. Missing extensibility hooks/resources/providers.
7. Task-decomposition guidance: how to split this into high-quality `cy-create-tasks`.
8. Final recommendation.

Be direct. If the spec is sound, say so and focus on the most useful refinements.
