import type { Filter, FilterFieldsConfig } from "@agh/ui";

import type { SkillPayload } from "../types";
import { filterSkillsByQuery, skillSourceLabel } from "./skill-formatters";

export type SkillSourceFilter = "bundled" | "workspace" | "marketplace" | "user" | "additional";

export type SkillEnabledFilter = "true" | "false";

export type SkillFilterFieldKey = "source" | "enabled";

export interface SkillFilterState {
  source: SkillSourceFilter | null;
  enabled: SkillEnabledFilter | null;
}

export interface SkillFilterHandlers {
  onSourceChange: (next: SkillSourceFilter | null) => void;
  onEnabledChange: (next: SkillEnabledFilter | null) => void;
}

const SOURCE_OPTIONS: SkillSourceFilter[] = [
  "bundled",
  "workspace",
  "marketplace",
  "user",
  "additional",
];

const ENABLED_OPTIONS: { value: SkillEnabledFilter; label: string }[] = [
  { value: "true", label: "Enabled" },
  { value: "false", label: "Disabled" },
];

export function buildSkillFilterFields(): FilterFieldsConfig<string> {
  return [
    {
      key: "source",
      label: "Source",
      type: "select",
      options: SOURCE_OPTIONS.map(value => ({
        value,
        label: skillSourceLabel(value),
      })),
    },
    {
      key: "enabled",
      label: "Status",
      type: "select",
      options: ENABLED_OPTIONS,
    },
  ];
}

export function skillFiltersToChips(state: SkillFilterState): Filter<string>[] {
  const chips: Filter<string>[] = [];
  if (state.source) {
    chips.push({
      id: "skill-filter-source",
      field: "source",
      operator: "is",
      values: [state.source],
    });
  }
  if (state.enabled) {
    chips.push({
      id: "skill-filter-enabled",
      field: "enabled",
      operator: "is",
      values: [state.enabled],
    });
  }
  return chips;
}

export function applySkillFilterChips(
  chips: Filter<string>[],
  handlers: SkillFilterHandlers
): void {
  const lookup = new Map<string, string | undefined>();
  for (const chip of chips) {
    lookup.set(chip.field, chip.values[0]);
  }
  handlers.onSourceChange(asSkillSource(lookup.get("source")));
  handlers.onEnabledChange(asSkillEnabled(lookup.get("enabled")));
}

export function filterInstalledSkills(
  skills: SkillPayload[],
  query: string,
  filters: SkillFilterState
): SkillPayload[] {
  let next = filterSkillsByQuery(skills, query);
  if (filters.source) {
    next = next.filter(skill => skill.source === filters.source);
  }
  if (filters.enabled) {
    const wantEnabled = filters.enabled === "true";
    next = next.filter(skill => skill.enabled === wantEnabled);
  }
  return [...next].sort((left, right) => left.name.localeCompare(right.name));
}

export function parseSkillSourceFilter(value: unknown): SkillSourceFilter | undefined {
  if (typeof value !== "string") return undefined;
  return asSkillSource(value) ?? undefined;
}

export function parseSkillEnabledFilter(value: unknown): SkillEnabledFilter | undefined {
  if (typeof value !== "string") return undefined;
  return asSkillEnabled(value) ?? undefined;
}

function asSkillSource(value: string | undefined): SkillSourceFilter | null {
  if (!value) return null;
  return (SOURCE_OPTIONS as readonly string[]).includes(value)
    ? (value as SkillSourceFilter)
    : null;
}

function asSkillEnabled(value: string | undefined): SkillEnabledFilter | null {
  if (value === "true" || value === "false") return value;
  return null;
}
