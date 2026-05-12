import type { Filter, FilterFieldsConfig } from "@agh/ui/components/reui/filters";

import type { SettingsProviderEntry, SettingsSourceKind } from "../types";

import {
  deriveProviderStateLabel,
  providerCredentialsConfigured,
  type ProviderStateLabel,
} from "./provider-state";

export type ProviderFilterFieldKey = "status" | "source" | "harness" | "auth_mode" | "default";

export type ProviderHarness = "acp" | "pi_acp";

export type ProviderAuthMode = "native_cli" | "bound_secret" | "none";

export type ProviderDefaultFilter = "default-only" | "non-default";

export interface ProviderFilterState {
  statusFilter: ProviderStateLabel | null;
  sourceFilter: SettingsSourceKind | null;
  harnessFilter: ProviderHarness | null;
  authModeFilter: ProviderAuthMode | null;
  defaultFilter: ProviderDefaultFilter | null;
  nameQuery: string;
}

export const DEFAULT_PROVIDER_FILTERS: ProviderFilterState = {
  statusFilter: null,
  sourceFilter: null,
  harnessFilter: null,
  authModeFilter: null,
  defaultFilter: null,
  nameQuery: "",
};

export interface ProviderFilterHandlers {
  onStatusChange: (next: ProviderStateLabel | null) => void;
  onSourceChange: (next: SettingsSourceKind | null) => void;
  onHarnessChange: (next: ProviderHarness | null) => void;
  onAuthModeChange: (next: ProviderAuthMode | null) => void;
  onDefaultChange: (next: ProviderDefaultFilter | null) => void;
}

const STATUS_OPTIONS: { value: ProviderStateLabel; label: string }[] = [
  { value: "installed", label: "Installed" },
  { value: "binary-missing", label: "Binary missing" },
  { value: "unconfigured", label: "Unconfigured" },
];

const SOURCE_OPTIONS: { value: SettingsSourceKind; label: string }[] = [
  { value: "builtin-provider", label: "Builtin" },
  { value: "global-config", label: "Global config" },
  { value: "workspace-config", label: "Workspace config" },
];

const HARNESS_OPTIONS: { value: ProviderHarness; label: string }[] = [
  { value: "acp", label: "ACP" },
  { value: "pi_acp", label: "PI-ACP" },
];

const AUTH_MODE_OPTIONS: { value: ProviderAuthMode; label: string }[] = [
  { value: "native_cli", label: "Native CLI" },
  { value: "bound_secret", label: "Bound secret" },
  { value: "none", label: "None" },
];

const DEFAULT_OPTIONS: { value: ProviderDefaultFilter; label: string }[] = [
  { value: "default-only", label: "Default provider" },
  { value: "non-default", label: "Not default" },
];

const STATUS_VALUES = new Set<ProviderStateLabel>(STATUS_OPTIONS.map(option => option.value));
const SOURCE_VALUES = new Set<SettingsSourceKind>(SOURCE_OPTIONS.map(option => option.value));
const HARNESS_VALUES = new Set<ProviderHarness>(HARNESS_OPTIONS.map(option => option.value));
const AUTH_MODE_VALUES = new Set<ProviderAuthMode>(AUTH_MODE_OPTIONS.map(option => option.value));
const DEFAULT_VALUES = new Set<ProviderDefaultFilter>(DEFAULT_OPTIONS.map(option => option.value));

export function buildProviderFilterFields(): FilterFieldsConfig<string> {
  return [
    { key: "status", label: "Status", type: "select", options: STATUS_OPTIONS },
    { key: "source", label: "Source", type: "select", options: SOURCE_OPTIONS },
    { key: "harness", label: "Harness", type: "select", options: HARNESS_OPTIONS },
    { key: "auth_mode", label: "Auth mode", type: "select", options: AUTH_MODE_OPTIONS },
    { key: "default", label: "Default", type: "select", options: DEFAULT_OPTIONS },
  ];
}

function buildChip(field: ProviderFilterFieldKey, value: string): Filter<string> {
  return {
    id: `provider-filter-${field}`,
    field,
    operator: "is",
    values: [value],
  };
}

export function providerFiltersToChips(state: ProviderFilterState): Filter<string>[] {
  const chips: Filter<string>[] = [];
  if (state.statusFilter) chips.push(buildChip("status", state.statusFilter));
  if (state.sourceFilter) chips.push(buildChip("source", state.sourceFilter));
  if (state.harnessFilter) chips.push(buildChip("harness", state.harnessFilter));
  if (state.authModeFilter) chips.push(buildChip("auth_mode", state.authModeFilter));
  if (state.defaultFilter) chips.push(buildChip("default", state.defaultFilter));
  return chips;
}

export function applyProviderFilterChips(
  chips: Filter<string>[],
  handlers: ProviderFilterHandlers
): void {
  const lookup = new Map<string, string | undefined>();
  for (const chip of chips) {
    lookup.set(chip.field, chip.values[0]);
  }
  handlers.onStatusChange(asMember(lookup.get("status"), STATUS_VALUES));
  handlers.onSourceChange(asMember(lookup.get("source"), SOURCE_VALUES));
  handlers.onHarnessChange(asMember(lookup.get("harness"), HARNESS_VALUES));
  handlers.onAuthModeChange(asMember(lookup.get("auth_mode"), AUTH_MODE_VALUES));
  handlers.onDefaultChange(asMember(lookup.get("default"), DEFAULT_VALUES));
}

export function applyProviderFilters(
  providers: SettingsProviderEntry[],
  state: ProviderFilterState
): SettingsProviderEntry[] {
  const query = state.nameQuery.trim().toLowerCase();
  return providers.filter(provider => {
    if (query.length > 0 && !providerMatchesQuery(provider, query)) return false;
    if (state.statusFilter && deriveProviderStateLabel(provider) !== state.statusFilter)
      return false;
    if (
      state.sourceFilter &&
      provider.source_metadata.effective_source.kind !== state.sourceFilter
    ) {
      return false;
    }
    if (state.harnessFilter && provider.settings.harness !== state.harnessFilter) {
      return false;
    }
    if (state.authModeFilter && provider.settings.auth_mode !== state.authModeFilter) {
      return false;
    }
    if (state.defaultFilter === "default-only" && !provider.default) return false;
    if (state.defaultFilter === "non-default" && provider.default) return false;
    return true;
  });
}

function providerMatchesQuery(provider: SettingsProviderEntry, query: string): boolean {
  if (provider.name.toLowerCase().includes(query)) return true;
  const display = provider.settings.display_name?.toLowerCase() ?? "";
  return display.includes(query);
}

function asMember<T extends string>(value: string | undefined, allowed: Set<T>): T | null {
  if (!value) return null;
  return allowed.has(value as T) ? (value as T) : null;
}

// Re-export here so callers don't have to import from two files for the filter API.
export { providerCredentialsConfigured };
