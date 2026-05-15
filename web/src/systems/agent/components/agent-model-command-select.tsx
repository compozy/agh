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
} from "@agh/ui";

export interface AgentModelCommandSelectProps {
  options: string[];
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  disabled?: boolean;
  loading?: boolean;
  triggerId?: string;
  triggerTestId?: string;
  className?: string;
}

export function AgentModelCommandSelect({
  options,
  value,
  onChange,
  placeholder = "Provider default",
  disabled,
  loading,
  triggerId,
  triggerTestId,
  className,
}: AgentModelCommandSelectProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const trimmedValue = value.trim();
  const knownOptions = useMemo(() => {
    const seen = new Set<string>();
    const result: string[] = [];
    for (const option of options) {
      const id = option.trim();
      if (!id || seen.has(id)) continue;
      seen.add(id);
      result.push(id);
    }
    return result;
  }, [options]);
  const trimmedQuery = query.trim();
  const showCustomItem = trimmedQuery !== "" && !knownOptions.includes(trimmedQuery);

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
        label={trimmedValue || (loading ? "Loading models..." : placeholder)}
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
          "data-testid": "agent-create-model-input",
        }}
      >
        <CommandList>
          <CommandEmpty data-testid="agent-create-model-empty">
            {trimmedQuery === ""
              ? "No models listed for this provider."
              : "Press Enter to use this name."}
          </CommandEmpty>
          <CommandSelectGroup
            heading="Provider default"
            data-testid="agent-create-model-default-group"
          >
            <CommandItem
              value="provider-default"
              onSelect={handleClear}
              data-checked={trimmedValue === "" ? "true" : "false"}
              data-testid="agent-create-model-item-default"
            >
              <span className="truncate text-sm text-fg">Use provider default</span>
            </CommandItem>
          </CommandSelectGroup>
          {knownOptions.length > 0 ? (
            <CommandSelectGroup
              heading="Available models"
              data-testid="agent-create-model-available-group"
            >
              {knownOptions.map(option => (
                <CommandItem
                  key={option}
                  value={option}
                  onSelect={() => handleSelect(option)}
                  data-checked={trimmedValue === option ? "true" : "false"}
                  data-testid={`agent-create-model-item-${option}`}
                >
                  <span className="truncate text-sm text-fg">{option}</span>
                </CommandItem>
              ))}
            </CommandSelectGroup>
          ) : null}
          {showCustomItem ? (
            <CommandSelectGroup
              heading="Custom model"
              data-testid="agent-create-model-custom-group"
            >
              <CommandItem
                value={`custom:${trimmedQuery}`}
                onSelect={() => handleSelect(trimmedQuery)}
                data-testid="agent-create-model-item-custom"
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
