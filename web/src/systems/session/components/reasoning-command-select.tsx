import { useState } from "react";
import { ChevronsUpDown, Gauge } from "lucide-react";

import {
  cn,
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@agh/ui";

const TRIGGER_BASE =
  "flex h-9 w-full items-center justify-between gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-none outline-none transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-ring/50";

export const REASONING_EFFORTS = ["minimal", "low", "medium", "high", "xhigh"] as const;
export type ReasoningEffort = (typeof REASONING_EFFORTS)[number];

const REASONING_LABELS: Record<ReasoningEffort, string> = {
  minimal: "Minimal · fastest",
  low: "Low",
  medium: "Medium",
  high: "High",
  xhigh: "Extra high · deepest",
};

export interface ReasoningCommandSelectProps {
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  disabled?: boolean;
  disabledHint?: string;
  triggerId?: string;
  triggerTestId?: string;
  className?: string;
}

export function ReasoningCommandSelect({
  value,
  onChange,
  placeholder = "Use provider default",
  disabled,
  disabledHint,
  triggerId,
  triggerTestId,
  className,
}: ReasoningCommandSelectProps) {
  const [open, setOpen] = useState(false);
  const trimmedValue = value.trim();

  const handleSelect = (next: string) => {
    onChange(next);
    setOpen(false);
  };

  const triggerLabel = trimmedValue
    ? (REASONING_LABELS[trimmedValue as ReasoningEffort] ?? trimmedValue)
    : disabled && disabledHint
      ? disabledHint
      : placeholder;

  const triggerEmphasis = trimmedValue ? "text-foreground" : "text-muted-foreground";

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
        title={disabled ? disabledHint : undefined}
      >
        <span className="flex min-w-0 flex-1 items-center gap-2 text-left">
          <Gauge aria-hidden="true" className="size-3.5 shrink-0 text-muted-foreground" />
          <span className={cn("truncate text-sm", triggerEmphasis)}>{triggerLabel}</span>
        </span>
        <ChevronsUpDown aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[var(--anchor-width)] min-w-64 p-0">
        <Command>
          <CommandInput
            placeholder="Filter effort levels..."
            data-testid="reasoning-command-input"
          />
          <CommandList>
            <CommandEmpty data-testid="reasoning-command-empty">
              No matching effort levels.
            </CommandEmpty>
            <CommandGroup heading="Reasoning effort">
              <CommandItem
                value="provider-default"
                onSelect={() => handleSelect("")}
                data-checked={trimmedValue === "" ? "true" : "false"}
                data-testid="reasoning-command-item-default"
              >
                <span className="truncate text-sm text-foreground">Use provider default</span>
              </CommandItem>
              {REASONING_EFFORTS.map(effort => (
                <CommandItem
                  key={effort}
                  value={effort}
                  onSelect={() => handleSelect(effort)}
                  data-checked={trimmedValue === effort ? "true" : "false"}
                  data-testid={`reasoning-command-item-${effort}`}
                >
                  <div className="flex min-w-0 flex-1 items-center gap-2">
                    <span className="truncate text-sm text-foreground">
                      {REASONING_LABELS[effort]}
                    </span>
                    <span className="ml-auto font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                      {effort}
                    </span>
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
