import { useMemo, useState } from "react";

import { CommandSelect, CommandSelectShell, CommandSelectTrigger, Eyebrow } from "@agh/ui";

import { AgentIcon } from "./agent-icon";
import { AgentCommandList } from "./agent-command-list";
import { formatCategoryLabel } from "../lib/agent-category";
import type { AgentPayload } from "../types";

export interface AgentCommandSelectProps {
  agents: AgentPayload[];
  value: string | null;
  onChange: (next: string | null) => void;
  placeholder?: string;
  disabled?: boolean;
  triggerTestId?: string;
  triggerId?: string;
  className?: string;
}

export function AgentCommandSelect({
  agents,
  value,
  onChange,
  placeholder = "Select an agent",
  disabled,
  triggerTestId,
  triggerId,
  className,
}: AgentCommandSelectProps) {
  const [open, setOpen] = useState(false);
  const selectedAgent = useMemo(
    () => agents.find(agent => agent.name === value) ?? null,
    [agents, value]
  );
  const isSelected = (agent: AgentPayload) => agent.name === value;
  const handleSelect = (agent: AgentPayload) => {
    onChange(agent.name);
    setOpen(false);
  };

  return (
    <CommandSelect open={open} onOpenChange={next => setOpen(next)}>
      <CommandSelectTrigger
        id={triggerId}
        aria-haspopup="listbox"
        aria-expanded={open}
        data-testid={triggerTestId}
        disabled={disabled}
        className={className}
        selected={Boolean(selectedAgent)}
      >
        {selectedAgent ? (
          <span className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <AgentIcon
              provider={selectedAgent.provider}
              size="xs"
              className="shrink-0 text-(--muted)"
            />
            <span className="truncate text-sm text-(--fg)">{selectedAgent.name}</span>
            <Eyebrow className="text-(--muted)">{selectedAgent.provider}</Eyebrow>
            {selectedAgent.category_path && selectedAgent.category_path.length > 0 ? (
              <Eyebrow
                className="text-(--muted) ml-auto truncate"
                data-testid="agent-command-select-trigger-category"
              >
                {formatCategoryLabel(selectedAgent.category_path)}
              </Eyebrow>
            ) : null}
          </span>
        ) : (
          <span className="truncate text-(--muted)">{placeholder}</span>
        )}
      </CommandSelectTrigger>
      <CommandSelectShell
        className="min-w-64"
        inputPlaceholder="Search agents..."
        inputProps={{ "data-testid": "agent-command-input" }}
      >
        <AgentCommandList agents={agents} isSelected={isSelected} onSelect={handleSelect} />
      </CommandSelectShell>
    </CommandSelect>
  );
}
