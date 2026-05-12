import { useMemo, type ReactNode } from "react";

import {
  CommandEmpty,
  CommandItem,
  CommandList,
  CommandSelectGroup,
  Eyebrow,
  KindIcon,
} from "@agh/ui";

import type { SessionProviderOption } from "../types";

const FALLBACK_GROUP_KEY = "general";
const FALLBACK_GROUP_LABEL = "Providers";

interface ProviderCommandGroupBucket {
  key: string;
  heading: string;
  options: SessionProviderOption[];
}

function harnessGroupKey(harness: string | undefined | null): string {
  const trimmed = (harness ?? "").trim();
  return trimmed === "" ? FALLBACK_GROUP_KEY : trimmed.toLowerCase();
}

function harnessGroupHeading(key: string): string {
  if (key === FALLBACK_GROUP_KEY) return FALLBACK_GROUP_LABEL;
  return key.toUpperCase();
}

function bucketByHarness(options: SessionProviderOption[]): ProviderCommandGroupBucket[] {
  const buckets = new Map<string, ProviderCommandGroupBucket>();
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

function providerSearchKey(option: SessionProviderOption): string {
  const segments = [option.name];
  if (option.display_name) segments.push(option.display_name);
  if (option.harness) segments.push(option.harness);
  if (option.runtime_provider) segments.push(option.runtime_provider);
  return segments.join(" ");
}

export interface ProviderCommandListProps {
  options: SessionProviderOption[];
  isSelected: (option: SessionProviderOption) => boolean;
  onSelect: (option: SessionProviderOption) => void;
  emptyState?: ReactNode;
  itemTestId?: (option: SessionProviderOption) => string;
}

export function ProviderCommandList({
  options,
  isSelected,
  onSelect,
  emptyState = "No providers match your search.",
  itemTestId,
}: ProviderCommandListProps) {
  const groups = useMemo(() => bucketByHarness(options), [options]);

  return (
    <CommandList>
      <CommandEmpty data-testid="provider-command-empty">{emptyState}</CommandEmpty>
      {groups.map(group => (
        <CommandSelectGroup
          key={group.key}
          heading={group.heading}
          data-testid={`provider-command-group-${group.key}`}
        >
          {group.options.map(option => {
            const selected = isSelected(option);
            return (
              <CommandItem
                key={option.name}
                value={providerSearchKey(option)}
                onSelect={() => onSelect(option)}
                data-checked={selected ? "true" : "false"}
                data-testid={
                  itemTestId ? itemTestId(option) : `provider-command-item-${option.name}`
                }
              >
                <div className="flex min-w-0 flex-1 items-center gap-2">
                  <KindIcon
                    className="shrink-0"
                    kind={option.runtime_provider ?? option.name}
                    size="xs"
                    tone="muted"
                  />
                  <span className="truncate text-sm text-fg">
                    {option.display_name?.trim() || option.name}
                  </span>
                  <Eyebrow className="text-muted">{option.harness ?? "acp"}</Eyebrow>
                </div>
              </CommandItem>
            );
          })}
        </CommandSelectGroup>
      ))}
    </CommandList>
  );
}
