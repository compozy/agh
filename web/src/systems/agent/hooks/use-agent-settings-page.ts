import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import {
  providerNeedsAuth,
  useRuntimeModelCatalog,
  type RuntimeCatalogProvider,
} from "@/systems/model-catalog";
import type { RuntimeModelOption, RuntimeProviderOption } from "@/systems/runtime";
import { useSettingsProviders, type SettingsProviderEntry } from "@/systems/settings";
import type { SessionProviderOption } from "@/systems/workspace";
import { useActiveWorkspace, useWorkspace } from "@/systems/workspace";

import { isAgentDigestConflict } from "../adapters/agent-api";
import {
  buildSettingsDraftFromAgent,
  buildUpdateAgentParams,
  isAgentSettingsDraftDirty,
  validateAgentSettingsDraft,
  type AgentSettingsDraft,
} from "../lib/agent-settings-draft";
import type { AgentSettingsSection } from "../lib/agent-settings-search";
import type { AgentPayload } from "../types";
import { useAgent, useUpdateAgent } from "./use-agents";
import { useAgentDeleteFlow } from "./use-agent-delete-flow";
import { useUnsavedGuard } from "./use-unsaved-guard";

function describeError(fallback: string, error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return fallback;
}

function settingsProviderToOption(provider: SettingsProviderEntry): RuntimeProviderOption {
  const displayName = provider.settings.display_name?.trim();
  const harness = provider.settings.harness?.trim();
  const runtimeProvider = provider.settings.runtime_provider?.trim();
  return {
    id: provider.name,
    name: displayName || provider.name,
    ...(harness ? { harness } : {}),
    ...(runtimeProvider ? { runtime_provider: runtimeProvider } : {}),
    needs_auth: providerNeedsAuth(provider.auth_status?.state),
  };
}

function workspaceProviderToOption(provider: SessionProviderOption): RuntimeProviderOption {
  const displayName = provider.display_name?.trim();
  const harness = provider.harness?.trim();
  const runtimeProvider = provider.runtime_provider?.trim();
  return {
    id: provider.name,
    name: displayName || provider.name,
    ...(harness ? { harness } : {}),
    runtime_provider: runtimeProvider || provider.name,
  };
}

export interface UseAgentSettingsPageOptions {
  name: string;
  section: AgentSettingsSection;
}

export function useAgentSettingsPage({ name, section }: UseAgentSettingsPageOptions) {
  const navigate = useNavigate();
  const { activeWorkspaceId, activeWorkspace } = useActiveWorkspace();
  const agentQuery = useAgent(name, activeWorkspaceId);
  const updateAgent = useUpdateAgent();
  const settingsProviders = useSettingsProviders();
  const workspaceDetail = useWorkspace(activeWorkspaceId ?? "", {
    enabled: activeWorkspaceId !== null,
  });

  const agent = agentQuery.data;
  const [draft, setDraftState] = useState<AgentSettingsDraft | null>(null);
  const [baselineAgent, setBaselineAgent] = useState<AgentPayload | null>(null);
  const [conflictBanner, setConflictBanner] = useState<string | null>(null);
  const [mutationDenied, setMutationDenied] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  useEffect(() => {
    if (!agent) return;
    setBaselineAgent(current =>
      current?.definition_digest === agent.definition_digest ? current : agent
    );
    setDraftState(current => {
      if (current === null) return buildSettingsDraftFromAgent(agent);
      if (current.definitionDigest === agent.definition_digest) return current;
      return buildSettingsDraftFromAgent(agent);
    });
  }, [agent]);

  const draftSafe = draft;
  const dirty = Boolean(
    baselineAgent && draftSafe && isAgentSettingsDraftDirty(draftSafe, baselineAgent)
  );
  const validation = useMemo(
    () => (draftSafe ? validateAgentSettingsDraft(draftSafe) : null),
    [draftSafe]
  );
  const canSave = Boolean(validation?.canSave);
  const saveBlocked = dirty && (!canSave || mutationDenied);
  const fieldErrorCount = validation ? Object.values(validation.fields).filter(Boolean).length : 0;

  const guard = useUnsavedGuard({ dirty, entityName: name });
  const deleteFlow = useAgentDeleteFlow({ agent, workspaceId: activeWorkspaceId });

  const useWorkspaceProviders = agent?.origin === "workspace";
  const globalProviders = useMemo(
    () => settingsProviders.data?.providers.map(settingsProviderToOption) ?? [],
    [settingsProviders.data?.providers]
  );
  const workspaceProviders = useMemo(
    () => (workspaceDetail.data?.providers ?? []).map(workspaceProviderToOption),
    [workspaceDetail.data?.providers]
  );
  const providerOptions = useWorkspaceProviders ? workspaceProviders : globalProviders;
  const providersLoading = useWorkspaceProviders
    ? activeWorkspaceId !== null && workspaceDetail.isLoading
    : settingsProviders.isLoading || settingsProviders.isFetching;

  const catalogProviders = useMemo<RuntimeCatalogProvider[]>(
    () => providerOptions.map(option => ({ id: option.id, needsAuth: option.needs_auth })),
    [providerOptions]
  );
  const catalog = useRuntimeModelCatalog(catalogProviders, { enabled: Boolean(agent) });

  const setDraft = useCallback((next: AgentSettingsDraft) => {
    setDraftState(next);
    setSaveError(null);
    setConflictBanner(null);
  }, []);

  const patchDraft = useCallback((patch: Partial<AgentSettingsDraft>) => {
    setDraftState(current => (current ? { ...current, ...patch } : current));
    setSaveError(null);
    setConflictBanner(null);
  }, []);

  const onDiscard = useCallback(() => {
    if (!baselineAgent) return;
    setDraftState(buildSettingsDraftFromAgent(baselineAgent));
    setSaveError(null);
    setConflictBanner(null);
  }, [baselineAgent]);

  const onReloadAndRetry = useCallback(async () => {
    setConflictBanner(null);
    setSaveError(null);
    const result = await agentQuery.refetch();
    if (result.data) {
      setBaselineAgent(result.data);
      setDraftState(buildSettingsDraftFromAgent(result.data));
    }
  }, [agentQuery]);

  const onSave = useCallback(() => {
    if (!draftSafe || !agent || saveBlocked || updateAgent.isPending) return;
    const params = buildUpdateAgentParams(draftSafe, activeWorkspaceId);
    if (!params) return;

    setSaveError(null);
    setConflictBanner(null);
    setMutationDenied(false);

    updateAgent.mutate(
      { name: agent.name, params },
      {
        onSuccess: updated => {
          setBaselineAgent(updated);
          setDraftState(buildSettingsDraftFromAgent(updated));
          toast.success("Changes saved");
        },
        onError: error => {
          if (isAgentDigestConflict(error)) {
            setConflictBanner(
              "This agent changed elsewhere. Reload the latest definition, then retry your edits."
            );
            return;
          }
          const status =
            error && typeof error === "object" && "status" in error
              ? Number((error as { status?: number }).status)
              : NaN;
          if (status === 403) {
            setMutationDenied(true);
            return;
          }
          setSaveError(describeError("Couldn't save agent", error));
        },
      }
    );
  }, [activeWorkspaceId, agent, draftSafe, saveBlocked, updateAgent]);

  const setSection = useCallback(
    (next: AgentSettingsSection) => {
      void navigate({
        to: "/agents/$name/settings",
        params: { name },
        search: { section: next },
        replace: true,
      });
    },
    [name, navigate]
  );

  const onBackToDetail = useCallback(() => {
    void navigate({
      to: "/agents/$name",
      params: { name },
    });
  }, [name, navigate]);

  const onOpenProviderSettings = useCallback(() => {
    void navigate({ to: "/settings/providers" });
  }, [navigate]);

  const saveBlockedCaption = mutationDenied
    ? "Editing is not permitted for this agent."
    : saveBlocked && fieldErrorCount > 0
      ? `Fix ${fieldErrorCount} field${fieldErrorCount === 1 ? "" : "s"} before saving`
      : undefined;

  return {
    agent,
    agentLoading: agentQuery.isLoading,
    agentError: (agentQuery.error as Error | null) ?? null,
    draft: draftSafe,
    setDraft,
    patchDraft,
    dirty,
    validation,
    canSave,
    saveBlocked,
    saveBlockedCaption,
    section,
    setSection,
    onSave,
    onDiscard,
    onReloadAndRetry,
    onBackToDetail,
    onOpenProviderSettings,
    isSaving: updateAgent.isPending,
    saveError,
    conflictBanner,
    mutationDenied,
    fieldsDisabled: updateAgent.isPending,
    fieldsReadOnly: mutationDenied,
    providerOptions,
    providersLoading,
    runtimeModels: catalog.models as RuntimeModelOption[],
    modelCatalogLoading: catalog.loading,
    modelCatalogLoaded: catalog.loaded,
    modelCatalogRefreshing: catalog.refreshing,
    modelCatalogError: catalog.error,
    onRefreshCatalog: catalog.refresh,
    workspaceName: activeWorkspace?.name ?? null,
    deleteFlow,
    unsavedGuardDialog: guard.confirmDialog,
  };
}
