import { useMemo, useState } from "react";
import { ChevronsUpDown } from "lucide-react";

import { cn, Popover, PopoverContent, PopoverTrigger } from "@agh/ui";

import { AgentIcon } from "./agent-icon";
import { AgentCommandList } from "./agent-command-list";
import { formatCategoryLabel } from "../lib/agent-category";
import type { AgentPayload } from "../types";

const TRIGGER_BASE =
  "flex h-9 w-full items-center justify-between gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-none outline-none transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-ring/50";

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
    <Popover open={open} onOpenChange={next => setOpen(next)}>
      <PopoverTrigger
        type="button"
        id={triggerId}
        aria-haspopup="listbox"
        aria-expanded={open}
        data-testid={triggerTestId}
        disabled={disabled}
        className={cn(TRIGGER_BASE, className)}
      >
        {selectedAgent ? (
          <span className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <AgentIcon
              provider={selectedAgent.provider}
              className="size-3.5 shrink-0 text-muted-foreground"
            />
            <span className="truncate text-sm text-foreground">{selectedAgent.name}</span>
            <span className="font-mono text-badge uppercase tracking-mono text-muted-foreground">
              {selectedAgent.provider}
            </span>
            {selectedAgent.category_path && selectedAgent.category_path.length > 0 ? (
              <span
                className="ml-auto truncate font-mono text-badge uppercase tracking-mono text-muted-foreground"
                data-testid="agent-command-select-trigger-category"
              >
                {formatCategoryLabel(selectedAgent.category_path)}
              </span>
            ) : null}
          </span>
        ) : (
          <span className="truncate text-muted-foreground">{placeholder}</span>
        )}
        <ChevronsUpDown aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-(--anchor-width) min-w-64 p-0">
        <AgentCommandList agents={agents} isSelected={isSelected} onSelect={handleSelect} />
      </PopoverContent>
    </Popover>
  );
}
