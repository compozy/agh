import { Bot } from "lucide-react";

import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  FormSection,
  Input,
  RequiredMark,
  Textarea,
} from "@agh/ui";

import type { AgentCreateDialogDraft } from "../lib/agent-create-draft";

export interface AgentCreateDefinitionSectionProps {
  draft: AgentCreateDialogDraft;
  errors: Record<string, string | undefined>;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
}

/**
 * Simple tier: who this agent is and what it is responsible for.
 *
 * Name and prompt are the only fields the create contract rejects when empty,
 * so they stay in Simple — Advanced never hides a required field.
 */
export function AgentCreateDefinitionSection({
  draft,
  errors,
  onDraftChange,
}: AgentCreateDefinitionSectionProps) {
  return (
    <FormSection
      data-testid="agent-create-definition"
      description="Who is this agent, and what is it responsible for?"
      icon={Bot}
      size="compact"
      title="The definition"
    >
      <Field data-invalid={Boolean(errors.name)}>
        <FieldLabel htmlFor="agent-create-name">
          Agent name
          <RequiredMark />
        </FieldLabel>
        <Input
          aria-invalid={Boolean(errors.name)}
          autoFocus
          className="font-mono"
          data-testid="agent-create-name"
          id="agent-create-name"
          onChange={event => onDraftChange({ ...draft, name: event.target.value })}
          placeholder="release-captain"
          value={draft.name}
        />
        <FieldError data-testid="agent-create-name-error">{errors.name}</FieldError>
      </Field>

      <Field data-invalid={Boolean(errors.prompt)}>
        <FieldLabel htmlFor="agent-create-prompt">
          Instructions
          <RequiredMark />
        </FieldLabel>
        <FieldDescription>Responsibility, boundaries, and escalation rules.</FieldDescription>
        <Textarea
          aria-invalid={Boolean(errors.prompt)}
          className="min-h-40"
          data-testid="agent-create-prompt"
          id="agent-create-prompt"
          onChange={event => onDraftChange({ ...draft, prompt: event.target.value })}
          placeholder="You are responsible for release readiness..."
          value={draft.prompt}
        />
        <FieldError data-testid="agent-create-prompt-error">{errors.prompt}</FieldError>
      </Field>
    </FormSection>
  );
}
