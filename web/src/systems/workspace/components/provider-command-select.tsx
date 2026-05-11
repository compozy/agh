import { useMemo, useState } from "react";

import {
  CommandSelect,
  CommandSelectShell,
  CommandSelectTrigger,
  Eyebrow,
  KindIcon,
} from "@agh/ui";

import { ProviderCommandList } from "./provider-command-list";
import type { SessionProviderOption } from "../types";

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
    <CommandSelect open={open} onOpenChange={next => setOpen(next)}>
      <CommandSelectTrigger
        id={triggerId}
        aria-haspopup="listbox"
        aria-expanded={open}
        data-testid={triggerTestId}
        disabled={disabled}
        className={className}
        selected={Boolean(selected)}
      >
        {selected ? (
          <span className="flex min-w-0 flex-1 items-center gap-2 text-left">
            <KindIcon
              className="shrink-0"
              kind={selected.runtime_provider ?? selected.name}
              size="xs"
              tone="muted"
            />
            <span className="truncate text-sm text-fg">
              {selected.display_name?.trim() || selected.name}
            </span>
            <Eyebrow className="text-muted">{selected.harness ?? "acp"}</Eyebrow>
          </span>
        ) : (
          <span className="truncate text-muted">{placeholder}</span>
        )}
      </CommandSelectTrigger>
      <CommandSelectShell
        className="min-w-72"
        inputPlaceholder="Search providers..."
        inputProps={{ "data-testid": "provider-command-input" }}
      >
        <ProviderCommandList options={options} isSelected={isSelected} onSelect={handleSelect} />
      </CommandSelectShell>
    </CommandSelect>
  );
}
