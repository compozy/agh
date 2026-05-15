import { Boxes } from "lucide-react";
import { useMemo, useState, type KeyboardEvent } from "react";

import {
  CommandEmpty,
  CommandItem,
  CommandList,
  CommandSelect,
  CommandSelectGroup,
  CommandSelectShell,
  CommandSelectTrigger,
  Eyebrow,
  Pill,
} from "@agh/ui";

import type { ModelSelectOption } from "../types";

export interface ModelCommandSelectProps {
  options: ModelSelectOption[];
  value: string;
  defaultModel?: string | null;
  onChange: (next: string) => void;
  placeholder?: string;
  disabled?: boolean;
  loading?: boolean;
  triggerId?: string;
  triggerTestId?: string;
  className?: string;
  testIdPrefix?: string;
}

export function ModelCommandSelect({
  options,
  value,
  defaultModel = null,
  onChange,
  placeholder = "Use provider default",
  disabled,
  loading,
  triggerId,
  triggerTestId,
  className,
  testIdPrefix = "model-command",
}: ModelCommandSelectProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const trimmedValue = value.trim();
  const trimmedDefault = defaultModel?.trim() ?? "";
  const knownOptions = useMemo(() => {
    const seen = new Set<string>();
    const result: ModelSelectOption[] = [];
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
    : loading
      ? "Loading models..."
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
        icon={<Boxes aria-hidden="true" className="size-3" />}
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
          "data-testid": `${testIdPrefix}-input`,
        }}
      >
        <CommandList>
          <CommandEmpty data-testid={`${testIdPrefix}-empty`}>
            {trimmedQuery === ""
              ? "No models listed for this provider."
              : "Press Enter to use this name."}
          </CommandEmpty>
          <CommandSelectGroup
            heading="Provider default"
            data-testid={`${testIdPrefix}-default-group`}
          >
            <CommandItem
              value="provider-default"
              onSelect={handleClear}
              data-checked={trimmedValue === "" ? "true" : "false"}
              data-testid={`${testIdPrefix}-item-default`}
            >
              <div className="flex min-w-0 flex-1 items-center gap-2">
                <span className="truncate text-sm text-fg">Use provider default</span>
                {trimmedDefault ? (
                  <Eyebrow className="ml-auto truncate text-muted">{trimmedDefault}</Eyebrow>
                ) : null}
              </div>
            </CommandItem>
          </CommandSelectGroup>
          {knownOptions.length > 0 ? (
            <CommandSelectGroup
              heading="Available models"
              data-testid={`${testIdPrefix}-available-group`}
            >
              {knownOptions.map(option => (
                <CommandItem
                  key={option.id}
                  value={option.id}
                  onSelect={() => handleSelect(option.id)}
                  data-checked={trimmedValue === option.id ? "true" : "false"}
                  data-testid={`${testIdPrefix}-item-${option.id}`}
                  data-availability={option.availability?.state}
                >
                  <div className="flex min-w-0 flex-1 items-center gap-2">
                    <span className="truncate text-sm text-fg">{option.label}</span>
                    {option.availability ? (
                      <Pill
                        mono
                        tone={option.availability.tone}
                        className="ml-auto"
                        data-testid={`${testIdPrefix}-item-${option.id}-availability`}
                      >
                        {option.availability.label}
                      </Pill>
                    ) : null}
                  </div>
                </CommandItem>
              ))}
            </CommandSelectGroup>
          ) : null}
          {showCustomItem ? (
            <CommandSelectGroup heading="Custom model" data-testid={`${testIdPrefix}-custom-group`}>
              <CommandItem
                value={`custom:${trimmedQuery}`}
                onSelect={() => handleSelect(trimmedQuery)}
                data-testid={`${testIdPrefix}-item-custom`}
              >
                <span className="truncate text-sm text-fg">Use "{trimmedQuery}"</span>
              </CommandItem>
            </CommandSelectGroup>
          ) : null}
        </CommandList>
      </CommandSelectShell>
    </CommandSelect>
  );
}
