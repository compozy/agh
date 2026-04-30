---
name: cy-create-techspec
description: Creates a Technical Specification by translating PRD business requirements into implementation designs through interactive technical clarification, then optionally coordinates user-directed peer-review rounds after the user approves the draft. Use when a PRD exists and needs a technical plan, or when technical architecture decisions need documentation. Do not use for PRD creation, task breakdown, or direct code implementation.
argument-hint: "[feature-name] [prd-file]"
---

# Create TechSpec

Translate business requirements into a detailed technical specification.

<HARD-GATE>
Do NOT write the TechSpec file until the complete draft has been reviewed and approved by the user.
Do NOT skip the codebase exploration — every TechSpec MUST be informed by existing architecture.
Do NOT skip user interactions — the user MUST participate in shaping the TechSpec at every decision point.
Do NOT require section-by-section approval — generate the complete draft, then let the user review it.
Do NOT run `cy-spec-peer-review` automatically. Offer it only after the user approves the complete draft and `_techspec.md` has been saved.
Do NOT incorporate peer-review findings automatically. Present the findings and wait for the user to choose what to incorporate.
Do NOT auto-loop peer-review rounds. After each round, ask whether the user wants another round or wants to stop there.
This applies to EVERY TechSpec regardless of perceived simplicity.
</HARD-GATE>

## Asking Questions

When this skill instructs you to ask the user a question, you MUST use your runtime's dedicated interactive question tool — the tool or function that presents a question to the user and **pauses execution until the user responds**. Do not output questions as plain assistant text and continue generating; always use the mechanism that blocks until the user has answered.

If your runtime does not provide such a tool, present the question as your complete message and stop generating. Do not answer your own question or proceed without user input.

## Anti-Pattern: "This Is Too Simple To Need Technical Design Review"

Every TechSpec goes through the full design review process. A single endpoint, a minor refactor, a configuration change — all of them. "Simple" technical changes are where unexamined assumptions about existing architecture cause the most integration failures. The design review can be brief for genuinely simple changes, but you MUST ask technical clarification questions and get approval on the technical approach before writing the artifact.

## Anti-Pattern: End-Of-Flow Bureaucracy

Once the user has answered the technical clarification questions and approved an approach, do not force them through a second approval loop for System Architecture, Data Models, API Design, or other final document sections. Synthesize the approved direction into the TechSpec directly. The user can review and request edits in the generated file afterward.

## Required Inputs

- Feature name identifying the `.compozy/tasks/<name>/` directory.
- Optional: existing `_prd.md` as primary input.
- Optional: existing `_techspec.md` for update mode.

## Checklist

You MUST create a task for each phase and complete them in order:

1. **Gather context** — read PRD, ADRs, and explore codebase architecture
2. **Ask technical questions** — 3-6 targeted questions on architecture, data models, APIs, testing
3. **Create ADRs** — record significant technical decisions (architecture pattern, technology choices, data model approach)
4. **Draft the TechSpec** — write using the canonical template from `references/techspec-template.md`
5. **Review with user** — present the draft, iterate until the user approves the draft
6. **Save the approved draft** — write to `.compozy/tasks/<name>/_techspec.md`
7. **Offer optional peer review** — ask whether to run `cy-spec-peer-review`
8. **Run user-directed review rounds** — summarize findings, incorporate only user-selected changes, and ask whether to continue or stop

## Workflow

1. Gather context.
   - Check for `_prd.md` in `.compozy/tasks/<name>/`. If it exists, read it as the primary input.
   - If no PRD exists, ask the user for a description of what needs technical specification.
   - Read existing ADRs from `.compozy/tasks/<name>/adrs/` to understand decisions already made during PRD creation.
   - Create `.compozy/tasks/<name>/adrs/` directory if it does not exist.
   - Spawn an Agent tool call to explore the codebase for architecture patterns, existing components, dependencies, and technology stack.
   - If `_techspec.md` already exists, read it and operate in update mode.

2. Ask technical clarification questions.
   - Focus on HOW to implement, WHERE components live, and WHICH technologies to use.
   - Cover architecture approach and component boundaries.
   - Cover data models and storage choices.
   - Cover API design and integration points.
   - Cover testing strategy and performance requirements.
   - Ask only one question per message. If a topic needs more exploration, break it into a sequence of individual questions.
   - Prefer multiple-choice questions when the options can be predetermined.
   - Include a fallback option (e.g., "D) Other — describe") for flexibility.

3. Create ADRs for significant technical decisions.
   - For each significant decision (architecture pattern chosen, technology selected, data model approach, etc.):
     - Read `references/adr-template.md`.
     - Determine the next ADR number by listing existing files in `.compozy/tasks/<name>/adrs/`.
     - Fill the template: the chosen design as "Decision", rejected alternatives as "Alternatives Considered", and trade-offs as "Consequences". Set Status to "Accepted" and Date to today.
     - Write each ADR to `.compozy/tasks/<name>/adrs/adr-NNN.md` (zero-padded 3-digit sequential number).

4. Draft the TechSpec.
   - Read `references/techspec-template.md` and fill every applicable section.
   - **MANDATORY — Architecture Decision Records section:** The generated TechSpec MUST end with an "Architecture Decision Records" section listing every ADR created during this process. Each entry must include the ADR number (e.g., ADR-001), title, and a one-line summary formatted as a link to the `adrs/` directory. Even simple features require at least one ADR documenting the primary technical approach chosen and alternatives rejected. If no ADRs were created in step 3, go back and create at least one before generating the document.
   - **MANDATORY — Extensibility / Agent Manageability / Config Lifecycle sections:** Every TechSpec MUST include these sections, even when the conclusion is explicit no-impact with evidence.
   - Apply YAGNI ruthlessly: remove any component, interface, or abstraction that is not strictly necessary. Do NOT propose new packages or directories when the feature can be implemented by adding a single file to an existing package.
   - Every PRD goal and user story should map to a technical component.
   - Reference PRD sections by name but do not duplicate business context.
   - Include code examples only for core interfaces, limited to 20 lines each. The Core Interfaces section must contain at least one Go interface or struct definition as a code block, even for simple features — show the primary type that other components will depend on.
   - The Development Sequencing section MUST include a numbered Build Order where every step after the first explicitly states which previous steps it depends on.
   - Prefer active voice, omit needless words, use definite and specific language over vague generalities. Every sentence should earn its place.
   - Language: **English**. Tone: clear, technical, consistent with existing project artifacts.
   - Present the complete draft to the user for review.

5. Review with the user.
   - Present the draft and ask using the interactive question tool:
     - "Here is the TechSpec draft. Please review and let me know:"
     - A) Approved — save this draft and continue to the peer-review choice
     - B) Adjust specific sections (tell me which ones)
     - C) Rewrite section X (tell me what to change)
     - D) Discard and start over
   - If B or C: make the changes and present again.
   - If D: go back to step 2.

6. Save the approved draft.
   - Write the approved document to `.compozy/tasks/<name>/_techspec.md`.
   - Confirm the file path to the user.

7. Offer optional peer review.
   - Ask using the interactive question tool:
     - "The approved TechSpec draft is saved. Do you want to run a cross-LLM peer review now?"
     - A) Yes — run `cy-spec-peer-review`
     - B) No — finish with the current saved TechSpec
   - If A: continue to step 8.
   - If B: remind the user that the next step is to create tasks using `cy-create-tasks` from this TechSpec.

8. Run user-directed review rounds.
   - Invoke `cy-spec-peer-review` only if the user explicitly opted in.
   - After each review round, present a concise summary of:
     - readiness verdict
     - blockers
     - nits
     - artifact paths written under `.compozy/tasks/<name>/qa/`
   - Ask which findings to incorporate:
     - A) Incorporate all blockers
     - B) Incorporate selected findings (user lists them)
     - C) Incorporate nothing
     - D) Make additional manual edits before deciding
   - Apply only the changes the user selected. Do not silently rewrite the TechSpec or ADRs based on reviewer output alone.
   - After incorporation (or an explicit decision to incorporate nothing), ask:
     - A) Run another review round
     - B) Stop review and keep the current saved TechSpec
   - If A: repeat step 8.
   - If B: remind the user that the next step is to create tasks using `cy-create-tasks` from this TechSpec.

## Process Flow

```dot
digraph create_techspec {
    "Gather context (PRD + codebase)" [shape=box];
    "Ask technical questions (one at a time)" [shape=box];
    "Create ADRs for key decisions" [shape=box];
    "Draft TechSpec (canonical template)" [shape=box];
    "User approves draft?" [shape=diamond];
    "Save approved _techspec.md" [shape=box];
    "User wants peer review?" [shape=diamond];
    "Run peer review + choose incorporations" [shape=box];
    "Another review round?" [shape=diamond];
    "Finish" [shape=doublecircle];

    "Gather context (PRD + codebase)" -> "Ask technical questions (one at a time)";
    "Ask technical questions (one at a time)" -> "Create ADRs for key decisions";
    "Create ADRs for key decisions" -> "Draft TechSpec (canonical template)";
    "Draft TechSpec (canonical template)" -> "User approves draft?";
    "User approves draft?" -> "Draft TechSpec (canonical template)" [label="no, revise"];
    "User approves draft?" -> "Save approved _techspec.md" [label="approved"];
    "Save approved _techspec.md" -> "User wants peer review?";
    "User wants peer review?" -> "Finish" [label="no"];
    "User wants peer review?" -> "Run peer review + choose incorporations" [label="yes"];
    "Run peer review + choose incorporations" -> "Another review round?";
    "Another review round?" -> "Run peer review + choose incorporations" [label="yes"];
    "Another review round?" -> "Finish" [label="no"];
}
```

## Error Handling

- If the PRD is missing, proceed with user-provided context and note the absence in the Executive Summary.
- If codebase exploration reveals conflicting architectural patterns, document both and recommend one with rationale.
- If the user rejects the design proposal, incorporate all feedback and present a revised proposal.
- If the target directory does not exist, create it.
- If operating in update mode, preserve sections the user has not asked to change.
- If the user declines peer review, keep the saved approved TechSpec as the final artifact for this skill run.

## Key Principles

- **One question at a time** — Do not overwhelm with multiple questions in a single message
- **Multiple choice preferred** — Easier for users to answer than open-ended when possible
- **YAGNI ruthlessly** — Remove unnecessary components, abstractions, and interfaces from all designs
- **Draft then review** — Generate the complete TechSpec draft first, then iterate with the user until approved
- **Draft approval before peer review** — The user approves the baseline draft before any optional `cy-spec-peer-review` run
- **User controls peer review** — The user decides whether to run review, what to incorporate, and whether to run another round
- **Technical focus only** — Never ask business questions; that belongs in the PRD
- **Trade-offs are mandatory** — Every Executive Summary must state the primary technical trade-off of the chosen approach
- **PRD as input** — When `_prd.md` exists, use it as primary context; every PRD goal should map to a technical component
- **Pipeline awareness** — The TechSpec feeds into `cy-create-tasks`; focus on HOW, not WHAT or WHY
- **Template compliance** — Every TechSpec MUST follow the canonical template
- **Language consistency** — Write all TechSpec content in English
