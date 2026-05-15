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
  KindIcon,
} from "@agh/ui";

import type { ProviderSelectOption } from "../types";

const FALLBACK_GROUP_KEY = "general";
const FALLBACK_GROUP_LABEL = "Providers";

interface ProviderGroupBucket {
  key: string;
  heading: string;
  options: ProviderSelectOption[];
}

function harnessGroupKey(harness: string | undefined | null): string {
  const trimmed = (harness ?? "").trim();
  return trimmed === "" ? FALLBACK_GROUP_KEY : trimmed.toLowerCase();
}

function harnessGroupHeading(key: string): string {
  if (key === FALLBACK_GROUP_KEY) return FALLBACK_GROUP_LABEL;
  return key.toUpperCase();
}

function bucketByHarness(options: ProviderSelectOption[]): ProviderGroupBucket[] {
  const buckets = new Map<string, ProviderGroupBucket>();
  for (const option of options) {
    const key = harnessGroupKey(option.harness);
    let bucket = buckets.get(key);
    if (!bucket) {
      bucket = { key, heading: harnessGroupHeading(key), options: [] };
      buckets.set(key, bucket);
    }
    bucket.options.push(option);
  }
  const order = Array.from(buckets.values());
  order.sort((a, b) => {
    if (a.key === FALLBACK_GROUP_KEY) return 1;
    if (b.key === FALLBACK_GROUP_KEY) return -1;
    return a.heading.localeCompare(b.heading);
  });
  return order;
}

function providerSearchKey(option: ProviderSelectOption): string {
  const segments = [option.name];
  if (option.display_name) segments.push(option.display_name);
  if (option.harness) segments.push(option.harness);
  if (option.runtime_provider) segments.push(option.runtime_provider);
  return segments.join(" ");
}

function providerDisplayName(option: ProviderSelectOption): string {
  return option.display_name?.trim() || option.name;
}

export interface ProviderCommandSelectProps {
  options: ProviderSelectOption[];
  value: string | null;
  onChange: (next: string | null) => void;
  placeholder?: string;
  disabled?: boolean;
  triggerId?: string;
  triggerTestId?: string;
  className?: string;
  testIdPrefix?: string;
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
  testIdPrefix = "provider-command",
}: ProviderCommandSelectProps) {
  const [open, setOpen] = useState(false);
  const trimmedValue = value?.trim() ?? "";
  const selected = useMemo(
    () => options.find(option => option.name === trimmedValue) ?? null,
    [options, trimmedValue]
  );
  const groups = useMemo(() => bucketByHarness(options), [options]);

  const handleSelect = (option: ProviderSelectOption) => {
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
            <span className="truncate text-small-body text-fg">
              {providerDisplayName(selected)}
            </span>
            <Eyebrow className="ml-auto text-muted">{selected.harness ?? "acp"}</Eyebrow>
          </span>
        ) : (
          <span className="truncate text-subtle">{placeholder}</span>
        )}
      </CommandSelectTrigger>
      <CommandSelectShell
        className="min-w-72"
        inputPlaceholder="Search providers..."
        inputProps={{ "data-testid": `${testIdPrefix}-input` }}
      >
        <CommandList>
          <CommandEmpty data-testid={`${testIdPrefix}-empty`}>
            No providers match your search.
          </CommandEmpty>
          {groups.map(group => (
            <CommandSelectGroup
              key={group.key}
              heading={group.heading}
              data-testid={`${testIdPrefix}-group-${group.key}`}
            >
              {group.options.map(option => {
                const isActive = option.name === trimmedValue;
                return (
                  <CommandItem
                    key={option.name}
                    value={providerSearchKey(option)}
                    onSelect={() => handleSelect(option)}
                    data-checked={isActive ? "true" : "false"}
                    data-testid={`${testIdPrefix}-item-${option.name}`}
                  >
                    <div className="flex min-w-0 flex-1 items-center gap-2">
                      <KindIcon
                        className="shrink-0"
                        kind={option.runtime_provider ?? option.name}
                        size="xs"
                        tone="muted"
                      />
                      <span className="truncate text-sm text-fg">
                        {providerDisplayName(option)}
                      </span>
                      <Eyebrow className="ml-auto text-muted">{option.harness ?? "acp"}</Eyebrow>
                    </div>
                  </CommandItem>
                );
              })}
            </CommandSelectGroup>
          ))}
        </CommandList>
      </CommandSelectShell>
    </CommandSelect>
  );
}
