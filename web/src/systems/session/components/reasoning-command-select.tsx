import { Gauge } from "lucide-react";
import { useMemo, useState } from "react";

import {
  CommandEmpty,
  CommandItem,
  CommandList,
  CommandSelect,
  CommandSelectGroup,
  CommandSelectShell,
  CommandSelectTrigger,
  Eyebrow,
} from "@agh/ui";

import type { ReasoningOption } from "@/systems/model-catalog";

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

  return (
    <CommandSelect open={open} onOpenChange={next => setOpen(next)}>
      <CommandSelectTrigger
        id={triggerId}
        aria-haspopup="listbox"
        aria-expanded={open}
        data-testid={triggerTestId}
        disabled={disabled}
        className={className}
        icon={<Gauge aria-hidden="true" className="size-3" />}
        label={triggerLabel}
        selected={Boolean(trimmedValue)}
        title={disabled ? disabledHint : undefined}
      />
      <CommandSelectShell
        inputProps={{
          placeholder: "Filter effort levels...",
          "data-testid": "reasoning-command-input",
        }}
      >
        <CommandList>
          <CommandEmpty data-testid="reasoning-command-empty">
            No matching effort levels.
          </CommandEmpty>
          <CommandSelectGroup heading="Reasoning effort">
            <CommandItem
              value="provider-default"
              onSelect={() => handleSelect("")}
              data-checked={trimmedValue === "" ? "true" : "false"}
              data-testid="reasoning-command-item-default"
            >
              <span className="truncate text-sm text-fg">Use provider default</span>
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
                  <span className="truncate text-sm text-fg">
                    {option.label || labelFor(option.value)}
                  </span>
                  <Eyebrow className="text-muted ml-auto">{option.value}</Eyebrow>
                </div>
              </CommandItem>
            ))}
          </CommandSelectGroup>
        </CommandList>
      </CommandSelectShell>
    </CommandSelect>
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
