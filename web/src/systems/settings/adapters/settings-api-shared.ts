import type {
  SettingsMCPServerDeleteFilter,
  SettingsMCPServerListFilter,
  SettingsMCPServerPutFilter,
  SettingsNotificationPresetFilter,
  SettingsApplyRecordsFilter,
  SettingsSkillsFilter,
  SettingsUpdateSkillsFilter,
} from "../types";

export class SettingsApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "SettingsApiError";
  }
}

export function normalizeOptionalText(value?: string | null): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function normalizeMCPListFilter(filter: SettingsMCPServerListFilter = {}) {
  return {
    scope: filter.scope,
    workspace_id: normalizeOptionalText(filter.workspace_id),
  };
}

export function normalizeMCPMutationFilter(
  filter: SettingsMCPServerPutFilter | SettingsMCPServerDeleteFilter = {}
) {
  return {
    scope: filter.scope,
    workspace_id: normalizeOptionalText(filter.workspace_id),
    target: filter.target,
  };
}

export function normalizeNotificationPresetFilter(filter: SettingsNotificationPresetFilter = {}) {
  return {
    enabled: filter.enabled,
    built_in: filter.built_in,
    name: normalizeOptionalText(filter.name),
    limit: filter.limit,
  };
}

export function normalizeSettingsSkillsFilter(
  filter: SettingsSkillsFilter | SettingsUpdateSkillsFilter = {}
) {
  return {
    scope: filter.scope,
    workspace_id: normalizeOptionalText(filter.workspace_id),
    agent_name: normalizeOptionalText(filter.agent_name),
  };
}

export function normalizeApplyRecordsFilter(filter: SettingsApplyRecordsFilter = {}) {
  return {
    status: filter.status,
    actor: normalizeOptionalText(filter.actor),
    limit: filter.limit,
  };
}
