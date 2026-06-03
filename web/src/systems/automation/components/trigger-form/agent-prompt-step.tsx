import { Field, FieldLabel, Input, NativeSelect, NativeSelectOption } from "@agh/ui";

import { PromptTemplateField } from "./prompt-template-field";

interface AgentPromptStepProps {
  agent: string;
  agents: string[];
  prompt: string;
  variables: string[];
  onAgentChange: (next: string) => void;
  onPromptChange: (next: string) => void;
}

/** "Then" step: the agent to run and the prompt template it receives. */
export function AgentPromptStep({
  agent,
  agents,
  prompt,
  variables,
  onAgentChange,
  onPromptChange,
}: AgentPromptStepProps) {
  const initial = agent.trim().charAt(0).toUpperCase() || "·";

  return (
    <div className="space-y-4">
      <Field>
        <FieldLabel htmlFor="trigger-agent">Agent</FieldLabel>
        <div className="relative">
          <span
            aria-hidden="true"
            className="absolute top-1/2 left-2 z-10 flex size-5 -translate-y-1/2 items-center justify-center rounded-sm bg-accent-strong font-mono text-mono-id font-semibold text-accent-ink"
          >
            {initial}
          </span>
          {agents.length > 0 ? (
            <NativeSelect
              className="w-full [&>select]:pl-9"
              data-testid="trigger-agent-input"
              id="trigger-agent"
              onChange={event => onAgentChange(event.target.value)}
              value={agent}
            >
              {agents.includes(agent) ? null : (
                <NativeSelectOption value={agent}>{agent || "Select an agent"}</NativeSelectOption>
              )}
              {agents.map(name => (
                <NativeSelectOption key={name} value={name}>
                  {name}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          ) : (
            <Input
              className="pl-9"
              data-testid="trigger-agent-input"
              id="trigger-agent"
              onChange={event => onAgentChange(event.target.value)}
              placeholder="reviewer"
              value={agent}
            />
          )}
        </div>
      </Field>
      <PromptTemplateField onChange={onPromptChange} value={prompt} variables={variables} />
    </div>
  );
}
