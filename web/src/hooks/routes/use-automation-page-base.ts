import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";

import { AutomationApiError } from "@/systems/automation";
import { localInputToDate, toRfc3339 } from "@/systems/automation/lib/cron-engine";
import type {
  AutomationScopeFilter,
  AutomationSource,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "@/systems/automation";
import { useSettingsAutomation } from "@/systems/settings";
import type { SettingsAutomationSection } from "@/systems/settings";
import { toWorkspaceCommandSelectOptions, useActiveWorkspace } from "@/systems/workspace";

export type JobEditorState =
  | { draft: CreateAutomationJobRequest; mode: "create" }
  | { draft: CreateAutomationJobRequest; id: string; mode: "edit" };

export type TriggerEditorState =
  | { draft: CreateAutomationTriggerRequest; mode: "create" }
  | { draft: CreateAutomationTriggerRequest; id: string; mode: "edit" };

/** Pre-target seed for opening the create sheet aimed at one Loop (§9.14 CTAs). */
export interface AutomationCreateSeed {
  /** When set, the page opens the create sheet in Run-loop mode for this Loop. */
  loop?: string;
}

export interface AutomationRouteSearch {
  create?: "loop";
  enabled?: boolean;
  event?: string;
  loop?: string;
  q?: string;
  scope?: AutomationScopeFilter;
  source?: AutomationSource;
}

export function buildEmptyState({
  hasQuery,
  kind,
  onCreate,
}: {
  hasQuery: boolean;
  kind: "jobs" | "triggers";
  onCreate: () => void;
}) {
  if (hasQuery) {
    return {
      description: "Try a different search term or adjust the current scope filter.",
      icon: "search" as const,
      title: kind === "jobs" ? "No jobs found" : "No triggers found",
    };
  }

  if (kind === "jobs") {
    return {
      actionLabel: "Create Job",
      description:
        "Scheduled jobs dispatch prompts to agents on a time-based cadence. Create your first job to start automating.",
      icon: "jobs" as const,
      onAction: onCreate,
      title: "No jobs configured",
    };
  }

  return {
    actionLabel: "Create Trigger",
    description:
      "Event-driven triggers react to daemon events, webhooks, and extension signals. Create your first trigger to enable reactive automation.",
    icon: "triggers" as const,
    onAction: onCreate,
    title: "No triggers configured",
  };
}

export function resolveSelectedId<T extends { id: string }>(selectedId: string | null, items: T[]) {
  if (selectedId && items.some(item => item.id === selectedId)) {
    return selectedId;
  }

  return items[0]?.id ?? null;
}

export function automationUnavailableMessage(
  kind: "jobs" | "triggers",
  runtime: SettingsAutomationSection["runtime"] | null,
  error: Error | null
): string | null {
  if (runtime && !runtime.available) {
    const noun = kind === "jobs" ? "Jobs" : "Triggers";
    return `${noun} are unavailable because the automation runtime is disabled. Enable automation and restart the daemon before using this surface.`;
  }

  if (error instanceof AutomationApiError && error.status === 503) {
    const noun = kind === "jobs" ? "jobs" : "triggers";
    return `Automation runtime is unavailable, so ${noun} cannot be loaded.`;
  }

  return null;
}

export function normalizeAutomationSchedule(
  schedule: CreateAutomationJobRequest["schedule"]
): CreateAutomationJobRequest["schedule"] {
  if (schedule.mode !== "at") {
    return schedule;
  }

  return {
    ...schedule,
    time: toRfc3339(localInputToDate(schedule.time ?? "")),
  };
}

export function useAutomationPageBase(
  kind: "jobs" | "triggers",
  search: AutomationRouteSearch = {}
) {
  const navigate = useNavigate();
  const { activeWorkspace, activeWorkspaceId, workspaces } = useActiveWorkspace();
  const settingsQuery = useSettingsAutomation();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const scopeFilter = search.scope ?? "all";
  const searchQuery = search.q ?? "";

  const scopedWorkspaceId =
    scopeFilter === "workspace" ? (activeWorkspaceId ?? undefined) : undefined;

  const listFilters = {
    limit: 50,
    enabled: search.enabled,
    loop: search.loop,
    q: searchQuery,
    scope: scopeFilter === "all" ? undefined : scopeFilter,
    source: search.source,
    workspace_id: scopedWorkspaceId,
  };

  const updateSearch = (updates: Partial<AutomationRouteSearch>) => {
    void navigate({
      search: current => ({ ...(current as AutomationRouteSearch), ...updates }),
      to: kind === "jobs" ? "/jobs" : "/triggers",
    });
  };

  const setScopeFilter = (scope: AutomationScopeFilter) =>
    updateSearch({ scope: scope === "all" ? undefined : scope });
  const setSearchQuery = (q: string) => updateSearch({ q: q.trim() === "" ? undefined : q });

  return {
    activeWorkspace,
    activeWorkspaceId,
    automationRuntime: settingsQuery.data?.runtime ?? null,
    listFilters,
    scopeFilter,
    searchQuery,
    selectedId,
    setScopeFilter,
    setSearchQuery,
    setSelectedId,
    workspaces: toWorkspaceCommandSelectOptions(workspaces),
  };
}
