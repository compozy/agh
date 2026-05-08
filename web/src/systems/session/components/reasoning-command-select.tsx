import { useMemo, useState } from "react";
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

import type { ReasoningOption } from "@/systems/model-catalog";

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
  options: ReasoningOption[];
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
  options,
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
  const knownOptions = useMemo(() => {
    const seen = new Set<string>();
    const result: ReasoningOption[] = [];
    for (const option of options) {
      const candidate = option.value.trim();
      if (!candidate || seen.has(candidate)) continue;
      seen.add(candidate);
      result.push({ ...option, value: candidate });
    }
    return result;
  }, [options]);

  const handleSelect = (next: string) => {
    onChange(next);
    setOpen(false);
  };

  const triggerLabel = trimmedValue
    ? labelFor(trimmedValue)
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
      <PopoverContent align="start" className="w-(--anchor-width) min-w-64 p-0">
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
              {knownOptions.map(option => (
                <CommandItem
                  key={option.value}
                  value={option.value}
                  onSelect={() => handleSelect(option.value)}
                  data-checked={trimmedValue === option.value ? "true" : "false"}
                  data-testid={`reasoning-command-item-${option.value}`}
                  data-source={option.source}
                >
                  <div className="flex min-w-0 flex-1 items-center gap-2">
                    <span className="truncate text-sm text-foreground">
                      {option.label || labelFor(option.value)}
                    </span>
                    <span className="ml-auto font-mono text-badge uppercase tracking-mono text-muted-foreground">
                      {option.value}
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

function labelFor(value: string): string {
  if (isReasoningEffort(value)) {
    return REASONING_LABELS[value];
  }
  return value;
}

function isReasoningEffort(value: string): value is ReasoningEffort {
  return (REASONING_EFFORTS as readonly string[]).includes(value);
}
