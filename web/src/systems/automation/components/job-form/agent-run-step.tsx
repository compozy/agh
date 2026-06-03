import { Field, FieldLabel, Input, NativeSelect, NativeSelectOption, Textarea } from "@agh/ui";

interface AgentRunStepProps {
  agent: string;
  agents: string[];
  prompt: string;
  onAgentChange: (next: string) => void;
  onPromptChange: (next: string) => void;
}

/** Agent output path: the agent to run and the plain prompt it receives verbatim. */
export function AgentRunStep({
  agent,
  agents,
  prompt,
  onAgentChange,
  onPromptChange,
}: AgentRunStepProps) {
  const initial = agent.trim().charAt(0).toUpperCase() || "·";

  return (
    <div className="space-y-4">
      <Field>
        <FieldLabel htmlFor="job-agent">Agent</FieldLabel>
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
              data-testid="job-agent-input"
              id="job-agent"
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
              data-testid="job-agent-input"
              id="job-agent"
              onChange={event => onAgentChange(event.target.value)}
              placeholder="reviewer"
              value={agent}
            />
          )}
        </div>
      </Field>
      <Field>
        <FieldLabel htmlFor="job-prompt">Prompt</FieldLabel>
        <Textarea
          data-testid="job-prompt-input"
          id="job-prompt"
          onChange={event => onPromptChange(event.target.value)}
          value={prompt}
        />
        <p className="text-form-hint leading-snug text-subtle">
          Sent to the agent verbatim; jobs don&apos;t template, there&apos;s no event to
          interpolate.
        </p>
      </Field>
    </div>
  );
}
