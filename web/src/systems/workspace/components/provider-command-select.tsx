import { useMemo, useState } from "react";
import { ChevronsUpDown, Cpu } from "lucide-react";

import { cn, Popover, PopoverContent, PopoverTrigger } from "@agh/ui";

import { ProviderCommandList } from "./provider-command-list";
import type { SessionProviderOption } from "../types";

const TRIGGER_BASE =
  "flex h-9 w-full items-center justify-between gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-none outline-none transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-ring/50";

export interface ProviderCommandSelectProps {
  options: SessionProviderOption[];
  value: string | null;
  onChange: (next: string | null) => void;
  placeholder?: string;
  disabled?: boolean;
  triggerId?: string;
  triggerTestId?: string;
  className?: string;
}

export function ProviderCommandSelect({
  options,
  value,
  onChange,
  placeholder = "Select a provider",
  disabled,
  triggerId,
  triggerTestId,
  className,
}: ProviderCommandSelectProps) {
  const [open, setOpen] = useState(false);
  const selected = useMemo(
    () => options.find(option => option.name === value) ?? null,
    [options, value]
  );
  const isSelected = (option: SessionProviderOption) => option.name === value;
  const handleSelect = (option: SessionProviderOption) => {
    onChange(option.name);
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
        {selected ? (
          <span className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <Cpu aria-hidden="true" className="size-3.5 shrink-0 text-muted-foreground" />
            <span className="truncate text-sm text-foreground">
              {selected.display_name?.trim() || selected.name}
            </span>
            <span className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
              {selected.harness ?? "acp"}
            </span>
          </span>
        ) : (
          <span className="truncate text-muted-foreground">{placeholder}</span>
        )}
        <ChevronsUpDown aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[var(--anchor-width)] min-w-72 p-0">
        <ProviderCommandList options={options} isSelected={isSelected} onSelect={handleSelect} />
      </PopoverContent>
    </Popover>
  );
}
