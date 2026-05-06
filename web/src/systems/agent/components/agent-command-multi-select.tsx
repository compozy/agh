import { useMemo, useState } from "react";
import { ChevronsUpDown } from "lucide-react";

import { cn, Pill, Popover, PopoverContent, PopoverTrigger } from "@agh/ui";

import { AgentCommandList } from "./agent-command-list";
import type { AgentPayload } from "../types";

const TRIGGER_BASE =
  "flex min-h-9 w-full items-center justify-between gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-none outline-none transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-ring/50";

export interface AgentCommandMultiSelectProps {
  agents: AgentPayload[];
  value: string[];
  onToggle: (next: string[]) => void;
  placeholder?: string;
  disabled?: boolean;
  triggerTestId?: string;
  triggerId?: string;
  itemTestId?: (agent: AgentPayload) => string;
  className?: string;
  countTestId?: string;
}

export function AgentCommandMultiSelect({
  agents,
  value,
  onToggle,
  placeholder = "Select agents",
  disabled,
  triggerTestId,
  triggerId,
  itemTestId,
  className,
  countTestId,
}: AgentCommandMultiSelectProps) {
  const [open, setOpen] = useState(false);
  const selectedSet = useMemo(() => new Set(value), [value]);
  const isSelected = (agent: AgentPayload) => selectedSet.has(agent.name);

  const handleSelect = (agent: AgentPayload) => {
    if (selectedSet.has(agent.name)) {
      onToggle(value.filter(name => name !== agent.name));
    } else {
      onToggle([...value, agent.name]);
    }
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
        <span className="flex min-w-0 flex-1 items-center gap-2 text-left">
          {value.length === 0 ? (
            <span className="truncate text-muted-foreground">{placeholder}</span>
          ) : (
            <span className="truncate text-sm text-foreground">{value.length} selected</span>
          )}
        </span>
        <Pill mono data-testid={countTestId}>
          {value.length}
        </Pill>
        <ChevronsUpDown aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[var(--anchor-width)] min-w-64 p-0">
        <AgentCommandList
          agents={agents}
          isSelected={isSelected}
          onSelect={handleSelect}
          itemTestId={itemTestId}
        />
      </PopoverContent>
    </Popover>
  );
}
