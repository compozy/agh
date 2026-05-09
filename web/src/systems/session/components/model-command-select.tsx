import { useMemo, useState, type KeyboardEvent } from "react";
import { Boxes } from "lucide-react";

import {
  CommandSelect,
  CommandSelectGroup,
  CommandSelectShell,
  CommandSelectTrigger,
  CommandEmpty,
  CommandItem,
  CommandList,
  Pill,
} from "@agh/ui";

import {
  modelAvailabilityLabel,
  modelAvailabilityTone,
  type ModelOption,
} from "@/systems/model-catalog";

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

  return (
    <CommandSelect open={open} onOpenChange={next => setOpen(next)}>
      <CommandSelectTrigger
        id={triggerId}
        aria-haspopup="listbox"
        aria-expanded={open}
        data-testid={triggerTestId}
        disabled={disabled}
        className={className}
        icon={<Boxes aria-hidden="true" className="size-3.5" />}
        label={triggerLabel}
        selected={Boolean(trimmedValue)}
      />
      <CommandSelectShell
        className="min-w-72"
        commandProps={{ shouldFilter: true }}
        inputProps={{
          placeholder: "Search or type a model...",
          value: query,
          onValueChange: setQuery,
          onKeyDown: handleInputKeyDown,
          "data-testid": "model-command-input",
        }}
      >
        <CommandList>
          <CommandEmpty data-testid="model-command-empty">
            {trimmedQuery === ""
              ? "No models listed for this provider."
              : "Press Enter to use this name."}
          </CommandEmpty>
          <CommandSelectGroup heading="Provider default" data-testid="model-command-default-group">
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
          </CommandSelectGroup>
          {knownOptions.length > 0 ? (
            <CommandSelectGroup
              heading="Available models"
              data-testid="model-command-available-group"
            >
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
                      tone={modelAvailabilityTone(option.availabilityState)}
                      className="ml-auto"
                      data-testid={`model-command-item-${option.id}-availability`}
                    >
                      {modelAvailabilityLabel(option.availabilityState)}
                    </Pill>
                  </div>
                </CommandItem>
              ))}
            </CommandSelectGroup>
          ) : null}
          {showCustomItem ? (
            <CommandSelectGroup heading="Custom model" data-testid="model-command-custom-group">
              <CommandItem
                value={`custom:${trimmedQuery}`}
                onSelect={() => handleSelect(trimmedQuery)}
                data-testid="model-command-item-custom"
              >
                <span className="truncate text-sm text-foreground">Use "{trimmedQuery}"</span>
              </CommandItem>
            </CommandSelectGroup>
          ) : null}
        </CommandList>
      </CommandSelectShell>
    </CommandSelect>
  );
}
