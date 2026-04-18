import type {
  SettingsMCPServerListFilter,
  SettingsSectionName,
  SettingsSectionSlug,
} from "../types";

function normalizeText(value?: string | null): string {
  if (value == null) {
    return "";
  }

  const trimmed = value.trim();
  return trimmed;
}

export type SettingsSectionKey = SettingsSectionName | SettingsSectionSlug;

export const settingsKeys = {
  all: ["settings"] as const,

  sections: () => [...settingsKeys.all, "section"] as const,
  section: (section: SettingsSectionKey) => [...settingsKeys.sections(), section] as const,

  collections: () => [...settingsKeys.all, "collection"] as const,

  providersRoot: () => [...settingsKeys.collections(), "providers"] as const,
  providersList: () => [...settingsKeys.providersRoot(), "list"] as const,
  providerDetail: (name: string) => [...settingsKeys.providersRoot(), "detail", name] as const,

  environmentsRoot: () => [...settingsKeys.collections(), "environments"] as const,
  environmentsList: () => [...settingsKeys.environmentsRoot(), "list"] as const,
  environmentDetail: (name: string) =>
    [...settingsKeys.environmentsRoot(), "detail", name] as const,

  hooksRoot: () => [...settingsKeys.collections(), "hooks"] as const,
  hooksList: () => [...settingsKeys.hooksRoot(), "list"] as const,

  mcpRoot: () => [...settingsKeys.collections(), "mcp-servers"] as const,
  mcpLists: () => [...settingsKeys.mcpRoot(), "list"] as const,
  mcpList: (filter: SettingsMCPServerListFilter = {}) =>
    [...settingsKeys.mcpLists(), filter.scope ?? "", normalizeText(filter.workspace_id)] as const,

  extensionsRoot: () => [...settingsKeys.all, "extensions"] as const,
  extensionsList: () => [...settingsKeys.extensionsRoot(), "list"] as const,

  restartRoot: () => [...settingsKeys.all, "restart"] as const,
  restartStatus: (operationId: string) => [...settingsKeys.restartRoot(), operationId] as const,
};
