import { useMemo, useState, type KeyboardEvent } from "react";
import { Boxes, ChevronsUpDown } from "lucide-react";

import {
  cn,
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  Pill,
  type PillTone,
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@agh/ui";

import type { ModelOption } from "@/systems/model-catalog";

const TRIGGER_BASE =
  "flex h-9 w-full items-center justify-between gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-none outline-none transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-ring/50";

const AVAILABILITY_LABELS: Record<string, string> = {
  available_live: "live",
  available_stale: "stale",
  unavailable_live: "unavailable",
  unavailable_stale: "stale · unavailable",
  unknown: "unknown",
};

const AVAILABILITY_TONES: Record<string, PillTone> = {
  available_live: "success",
  available_stale: "warning",
  unavailable_live: "danger",
  unavailable_stale: "warning",
  unknown: "neutral",
};

export interface ModelCommandSelectProps {
  options: ModelOption[];
  defaultModel: string | null;
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  disabled?: boolean;
  triggerId?: string;
  triggerTestId?: string;
  className?: string;
}

export function ModelCommandSelect({
  options,
  defaultModel,
  value,
  onChange,
  placeholder = "Use provider default",
  disabled,
  triggerId,
  triggerTestId,
  className,
}: ModelCommandSelectProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const trimmedValue = value.trim();
  const trimmedDefault = defaultModel?.trim() ?? "";
  const knownOptions = useMemo(() => {
    const seen = new Set<string>();
    const result: ModelOption[] = [];
    for (const option of options) {
      const id = option.id.trim();
      if (!id || seen.has(id)) continue;
      seen.add(id);
      result.push({ ...option, id });
    }
    return result;
  }, [options]);
  const trimmedQuery = query.trim();
  const showCustomItem =
    trimmedQuery !== "" && !knownOptions.some(option => option.id === trimmedQuery);

  const handleSelect = (next: string) => {
    onChange(next);
    setOpen(false);
    setQuery("");
  };

  const handleClear = () => {
    onChange("");
    setOpen(false);
    setQuery("");
  };

  const handleInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" && trimmedQuery !== "") {
      event.preventDefault();
      handleSelect(trimmedQuery);
    }
  };

  const triggerLabel = trimmedValue
    ? trimmedValue
    : trimmedDefault
      ? `${placeholder} · ${trimmedDefault}`
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
      >
        <span className="flex min-w-0 flex-1 items-center gap-2 text-left">
          <Boxes aria-hidden="true" className="size-3.5 shrink-0 text-muted-foreground" />
          <span className={cn("truncate text-sm", triggerEmphasis)}>{triggerLabel}</span>
        </span>
        <ChevronsUpDown aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-(--anchor-width) min-w-72 p-0">
        <Command shouldFilter={true}>
          <CommandInput
            placeholder="Search or type a model..."
            value={query}
            onValueChange={setQuery}
            onKeyDown={handleInputKeyDown}
            data-testid="model-command-input"
          />
          <CommandList>
            <CommandEmpty data-testid="model-command-empty">
              {trimmedQuery === ""
                ? "No models listed for this provider."
                : "Press Enter to use this name."}
            </CommandEmpty>
            <CommandGroup heading="Provider default" data-testid="model-command-default-group">
              <CommandItem
                value="provider-default"
                onSelect={handleClear}
                data-checked={trimmedValue === "" ? "true" : "false"}
                data-testid="model-command-item-default"
              >
                <div className="flex min-w-0 flex-1 items-center gap-2">
                  <span className="truncate text-sm text-foreground">Use provider default</span>
                  {trimmedDefault ? (
                    <span className="ml-auto truncate font-mono text-xs uppercase tracking-wide text-muted-foreground">
                      {trimmedDefault}
                    </span>
                  ) : null}
                </div>
              </CommandItem>
            </CommandGroup>
            {knownOptions.length > 0 ? (
              <CommandGroup heading="Available models" data-testid="model-command-available-group">
                {knownOptions.map(option => (
                  <CommandItem
                    key={option.id}
                    value={option.id}
                    onSelect={() => handleSelect(option.id)}
                    data-checked={trimmedValue === option.id ? "true" : "false"}
                    data-testid={`model-command-item-${option.id}`}
                    data-availability={option.availabilityState}
                  >
                    <div className="flex min-w-0 flex-1 items-center gap-2">
                      <span className="truncate text-sm text-foreground">{option.displayName}</span>
                      <Pill
                        mono
                        tone={AVAILABILITY_TONES[option.availabilityState] ?? "neutral"}
                        className="ml-auto"
                        data-testid={`model-command-item-${option.id}-availability`}
                      >
                        {AVAILABILITY_LABELS[option.availabilityState] ?? option.availabilityState}
                      </Pill>
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            ) : null}
            {showCustomItem ? (
              <CommandGroup heading="Custom model" data-testid="model-command-custom-group">
                <CommandItem
                  value={`custom:${trimmedQuery}`}
                  onSelect={() => handleSelect(trimmedQuery)}
                  data-testid="model-command-item-custom"
                >
                  <span className="truncate text-sm text-foreground">Use "{trimmedQuery}"</span>
                </CommandItem>
              </CommandGroup>
            ) : null}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
