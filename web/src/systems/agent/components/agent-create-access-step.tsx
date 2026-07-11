import { Plus, ShieldCheck, X } from "lucide-react";
import { useId, useState, type KeyboardEvent } from "react";

import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  FormSection,
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  Pill,
  RadioCard,
} from "@agh/ui";

import {
  AGENT_CREATE_PERMISSION_OPTIONS,
  appendAgentCreateTokens,
  removeAgentCreateToken,
  type AgentCreateDialogDraft,
  type AgentCreatePermissionChoice,
} from "../lib/agent-create-draft";

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

function TokenListField({
  description,
  error,
  label,
  onChange,
  placeholder,
  testId,
  values,
}: {
  description: string;
  error?: string;
  label: string;
  onChange: (values: string[]) => void;
  placeholder: string;
  testId: string;
  values: string[];
}) {
  const inputId = useId();
  const [inputValue, setInputValue] = useState("");

  const commit = () => {
    if (inputValue.trim().length === 0) return;
    onChange(appendAgentCreateTokens(values, inputValue));
    setInputValue("");
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      commit();
    }
  };

  return (
    <Field data-invalid={Boolean(error)}>
      <FieldLabel htmlFor={inputId}>{label}</FieldLabel>
      <FieldDescription>{description}</FieldDescription>
      <InputGroup>
        <InputGroupInput
          aria-invalid={Boolean(error)}
          data-testid={testId + "-input"}
          id={inputId}
          onBlur={commit}
          onChange={event => {
            const next = event.target.value;
            if (/[,\n]/.test(next)) {
              onChange(appendAgentCreateTokens(values, next));
              setInputValue("");
              return;
            }
            setInputValue(next);
          }}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          value={inputValue}
        />
        <InputGroupAddon align="inline-end">
          <InputGroupButton
            aria-label={"Add " + label.toLowerCase()}
            data-testid={testId + "-add"}
            disabled={inputValue.trim().length === 0}
            onClick={commit}
            size="icon-xs"
          >
            <Plus aria-hidden="true" className="size-3" />
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>
      {values.length > 0 ? (
        <div className="flex flex-wrap gap-1.5" data-testid={testId + "-tokens"}>
          {values.map(value => (
            <Pill key={value} className="gap-1 pr-1" size="sm">
              <span className="max-w-44 truncate">{value}</span>
              <button
                aria-label={"Remove " + value}
                className="inline-flex size-4 items-center justify-center rounded-sm text-subtle transition-colors hover:bg-hover hover:text-fg focus-visible:outline-none focus-visible:shadow-focus-ring"
                onClick={() => onChange(removeAgentCreateToken(values, value))}
                type="button"
              >
                <X aria-hidden="true" className="size-3" />
              </button>
            </Pill>
          ))}
        </div>
      ) : null}
      <FieldError data-testid={testId + "-error"}>{error}</FieldError>
    </Field>
  );
}
