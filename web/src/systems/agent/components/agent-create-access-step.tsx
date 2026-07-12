import { ShieldCheck } from "lucide-react";

import { Field, FieldDescription, FieldLabel, FormSection, RadioCard } from "@agh/ui";

import {
  AGENT_CREATE_PERMISSION_OPTIONS,
  type AgentCreateDialogDraft,
  type AgentCreatePermissionChoice,
} from "../lib/agent-create-draft";
import { TokenListField } from "./token-list-field";

const PERMISSION_DESCRIPTIONS: Record<AgentCreatePermissionChoice, string> = {
  "": "Use the runtime's default approval mode.",
  "deny-all": "Ask before every tool call.",
  "approve-reads": "Auto-approve read-only tools; ask for the rest.",
  "approve-all": "Auto-approve every allowed tool call.",
};

export interface AgentCreateAccessStepProps {
  draft: AgentCreateDialogDraft;
  errors: Record<string, string | undefined>;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
}

export function AgentCreateAccessStep({
  draft,
  errors,
  onDraftChange,
}: AgentCreateAccessStepProps) {
  return (
    <FormSection
      data-testid="agent-create-access"
      icon={ShieldCheck}
      size="compact"
      title="Access"
      description="Constrain the tools and skills available to sessions started from this agent."
    >
      <Field>
        <FieldLabel id="agent-create-permissions-label">Permissions</FieldLabel>
        <FieldDescription>Optional default approval posture for this agent.</FieldDescription>
        <div
          aria-labelledby="agent-create-permissions-label"
          className="grid gap-2 sm:grid-cols-2"
          data-testid="agent-create-permissions"
          role="radiogroup"
        >
          {AGENT_CREATE_PERMISSION_OPTIONS.map(option => (
            <RadioCard
              key={option.value || "inherit"}
              data-testid={"agent-create-permissions-" + (option.value || "inherit")}
              description={PERMISSION_DESCRIPTIONS[option.value]}
              onSelect={() => onDraftChange({ ...draft, permissions: option.value })}
              selected={draft.permissions === option.value}
              title={option.label}
            />
          ))}
        </div>
      </Field>

      <div className="grid gap-3.5 md:grid-cols-2">
        <TokenListField
          description="Canonical tool IDs or namespace wildcards."
          error={errors.tools}
          label="Tools"
          onChange={tools => onDraftChange({ ...draft, tools })}
          placeholder="agh__skill_view, mcp__github__*"
          testId="agent-create-tools"
          values={draft.tools}
        />
        <TokenListField
          description="Tool groups enabled for the agent."
          error={errors.toolsets}
          label="Toolsets"
          onChange={toolsets => onDraftChange({ ...draft, toolsets })}
          placeholder="agh__catalog"
          testId="agent-create-toolsets"
          values={draft.toolsets}
        />
        <TokenListField
          description="Canonical tools to deny after allow rules."
          error={errors.denyTools}
          label="Denied tools"
          onChange={denyTools => onDraftChange({ ...draft, denyTools })}
          placeholder="agh__task_*"
          testId="agent-create-deny-tools"
          values={draft.denyTools}
        />
        <TokenListField
          description="Skill names disabled only for this agent."
          label="Disabled skills"
          onChange={disabledSkills => onDraftChange({ ...draft, disabledSkills })}
          placeholder="code-review, release-notes"
          testId="agent-create-disabled-skills"
          values={draft.disabledSkills}
        />
      </div>
    </FormSection>
  );
}
