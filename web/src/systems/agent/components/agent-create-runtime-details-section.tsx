import { Terminal } from "lucide-react";

import { Field, FieldDescription, FieldError, FieldLabel, FormSection, Input } from "@agh/ui";

import type { AgentCreateDialogDraft } from "../lib/agent-create-draft";

export interface AgentCreateRuntimeDetailsSectionProps {
  draft: AgentCreateDialogDraft;
  errors: Record<string, string | undefined>;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
}

/**
 * Advanced tier: overrides for how the provider subprocess launches.
 *
 * Category path stays a single mono input; the adapter splits it into the
 * contract's `category_path []string` on `/` (T2).
 */
export function AgentCreateRuntimeDetailsSection({
  draft,
  errors,
  onDraftChange,
}: AgentCreateRuntimeDetailsSectionProps) {
  return (
    <FormSection
      data-testid="agent-create-runtime-details"
      description="Overrides for how the provider subprocess launches."
      icon={Terminal}
      size="compact"
      title="Runtime details"
    >
      <div className="grid gap-3.5 md:grid-cols-2">
        <Field>
          <FieldLabel htmlFor="agent-create-command">Runtime command</FieldLabel>
          <FieldDescription>Optional provider command override for this agent.</FieldDescription>
          <Input
            className="font-mono"
            data-testid="agent-create-command"
            id="agent-create-command"
            onChange={event => onDraftChange({ ...draft, command: event.target.value })}
            placeholder="provider default"
            value={draft.command}
          />
        </Field>

        <Field data-invalid={Boolean(errors.categoryPath)}>
          <FieldLabel htmlFor="agent-create-category-path">Category path</FieldLabel>
          <FieldDescription>
            Slash-separated catalog grouping, stored as one segment per level.
          </FieldDescription>
          <Input
            aria-invalid={Boolean(errors.categoryPath)}
            className="font-mono"
            data-testid="agent-create-category-path"
            id="agent-create-category-path"
            onChange={event => onDraftChange({ ...draft, categoryPath: event.target.value })}
            placeholder="operations/incident"
            value={draft.categoryPath}
          />
          <FieldError data-testid="agent-create-category-path-error">
            {errors.categoryPath}
          </FieldError>
        </Field>
      </div>
    </FormSection>
  );
}
