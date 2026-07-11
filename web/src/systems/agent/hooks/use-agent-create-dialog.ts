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
import type { SessionProviderOption, WorkspacePayload } from "@/systems/workspace";

import { useCreateAgent, useDuplicateAgent } from "./use-agents";
import {
  buildCreateAgentParams,
  buildDraftFromAgentPayload,
  buildDuplicateAgentParams,
  createDefaultAgentCreateDraft,
  updateAgentCreateScope,
  validateAgentCreateDraft,
  type AgentCreateDialogDraft,
} from "../lib/agent-create-draft";
import type { AgentPayload } from "../types";

interface AgentCreateDialogContext {
  activeWorkspace: WorkspacePayload | undefined;
  workspaceProviders: SessionProviderOption[];
  workspaceProvidersError: string | null;
  workspaceProvidersLoading: boolean;
}

export interface AgentCreateDialogState {
  open: boolean;
  draft: AgentCreateDialogDraft;
  providerOptions: RuntimeProviderOption[];
  providersLoading: boolean;
  providersError: string | null;
  runtimeModels: RuntimeModelOption[];
  modelCatalogLoading: boolean;
  modelCatalogLoaded: boolean;
  modelCatalogRefreshing: boolean;
  modelCatalogError: string | null;
  submitError: string | null;
  isSubmitting: boolean;
  hasActiveWorkspace: boolean;
  workspaceName: string | null;
  mode: "create" | "duplicate";
  duplicateSourceName: string | null;
}

export interface AgentCreateDialogApi extends AgentCreateDialogState {
  openDialog: () => void;
  openForDuplicate: (agent: AgentPayload) => void;
  onDraftChange: (draft: AgentCreateDialogDraft) => void;
  onOpenChange: (open: boolean) => void;
  onRefreshCatalog: () => void;
  onOpenProviderSettings: () => void;
  onSubmit: () => Promise<void>;
}

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

export function useAgentCreateDialog({
  activeWorkspace,
  workspaceProviders,
  workspaceProvidersError,
  workspaceProvidersLoading,
}: AgentCreateDialogContext): AgentCreateDialogApi {
  const navigate = useNavigate();
  const createAgent = useCreateAgent();
  const duplicateAgent = useDuplicateAgent();
  const settingsProviders = useSettingsProviders();
  const [open, setOpenState] = useState(false);
  const [mode, setMode] = useState<"create" | "duplicate">("create");
  const [duplicateSource, setDuplicateSource] = useState<AgentPayload | null>(null);
  const [draft, setDraft] = useState<AgentCreateDialogDraft>(() =>
    createDefaultAgentCreateDraft(Boolean(activeWorkspace))
  );
  const [submitError, setSubmitError] = useState<string | null>(null);

  const globalProviders = useMemo<RuntimeProviderOption[]>(
    () => settingsProviders.data?.providers.map(settingsProviderToOption) ?? [],
    [settingsProviders.data?.providers]
  );

  const workspaceProviderOptions = useMemo<RuntimeProviderOption[]>(
    () => workspaceProviders.map(workspaceProviderToOption),
    [workspaceProviders]
  );

  const providerOptions = useMemo<RuntimeProviderOption[]>(
    () => (draft.scope === "workspace" ? workspaceProviderOptions : globalProviders),
    [draft.scope, globalProviders, workspaceProviderOptions]
  );

  const providersLoading =
    draft.scope === "workspace"
      ? workspaceProvidersLoading
      : settingsProviders.isLoading || settingsProviders.isFetching;
  const providersError =
    draft.scope === "workspace"
      ? workspaceProvidersError
      : settingsProviders.error
        ? describeError("Unable to load global provider settings.", settingsProviders.error)
        : null;

  useEffect(() => {
    setDraft(current => {
      if (current.scope !== "workspace" || activeWorkspace) return current;
      return updateAgentCreateScope(current, "global");
    });
  }, [activeWorkspace]);

  useEffect(() => {
    setDraft(current => {
      if (current.provider.length === 0) return current;
      if (providerOptions.some(option => option.id === current.provider)) return current;
      return { ...current, provider: "", model: "", reasoningEffort: "" };
    });
  }, [providerOptions]);

  const catalogProviders = useMemo<RuntimeCatalogProvider[]>(
    () => providerOptions.map(option => ({ id: option.id, needsAuth: option.needs_auth })),
    [providerOptions]
  );
  const catalog = useRuntimeModelCatalog(catalogProviders, { enabled: open });
  const runtimeModels = catalog.models;

  const validationContext = useMemo(
    () => ({
      hasActiveWorkspace: Boolean(activeWorkspace),
      providerOptions,
      providersError,
      providersLoading,
    }),
    [activeWorkspace, providerOptions, providersError, providersLoading]
  );

  const resetCreateState = useCallback(() => {
    setMode("create");
    setDuplicateSource(null);
    setDraft(createDefaultAgentCreateDraft(Boolean(activeWorkspace)));
    setSubmitError(null);
  }, [activeWorkspace]);

  const openDialog = useCallback(() => {
    resetCreateState();
    setOpenState(true);
  }, [resetCreateState]);

  const openForDuplicate = useCallback(
    (agent: AgentPayload) => {
      setMode("duplicate");
      setDuplicateSource(agent);
      setDraft(buildDraftFromAgentPayload(agent, Boolean(activeWorkspace)));
      setSubmitError(null);
      setOpenState(true);
    },
    [activeWorkspace]
  );

  const onOpenChange = useCallback(
    (next: boolean) => {
      setOpenState(next);
      if (!next) {
        resetCreateState();
      }
    },
    [resetCreateState]
  );

  const onDraftChange = useCallback((nextDraft: AgentCreateDialogDraft) => {
    setDraft(nextDraft);
    setSubmitError(null);
  }, []);

  const onRefreshCatalog = catalog.refresh;

  const onOpenProviderSettings = useCallback(() => {
    setOpenState(false);
    void navigate({ to: "/settings/providers" });
  }, [navigate]);

  const onSubmit = useCallback(async () => {
    setSubmitError(null);
    try {
      let agent: AgentPayload;
      if (mode === "duplicate") {
        if (!duplicateSource) {
          setSubmitError("Missing duplicate source agent.");
          return;
        }
        const request = buildDuplicateAgentParams(
          duplicateSource,
          draft,
          activeWorkspace?.id,
          validationContext
        );
        if (!request) {
          const validation = validateAgentCreateDraft(draft, validationContext);
          const message =
            Object.values(validation.fields).find(field => field && field.length > 0) ??
            "Fix the highlighted fields before duplicating an agent.";
          setSubmitError(message);
          return;
        }
        agent = await duplicateAgent.mutateAsync({
          sourceName: duplicateSource.name,
          params: request,
        });
      } else {
        const request = buildCreateAgentParams(draft, activeWorkspace?.id, validationContext);
        if (!request) {
          const validation = validateAgentCreateDraft(draft, validationContext);
          const message =
            Object.values(validation.fields).find(field => field && field.length > 0) ??
            "Fix the highlighted fields before creating an agent.";
          setSubmitError(message);
          return;
        }
        agent = await createAgent.mutateAsync(request);
      }
      setOpenState(false);
      resetCreateState();
      await navigate({
        to: "/agents/$name",
        params: { name: agent.name },
      });
    } catch (error) {
      const message = describeError(
        mode === "duplicate" ? "Failed to duplicate agent." : "Failed to create agent.",
        error
      );
      setSubmitError(message);
      toast.error(message);
    }
  }, [
    activeWorkspace,
    createAgent,
    draft,
    duplicateAgent,
    duplicateSource,
    mode,
    navigate,
    resetCreateState,
    validationContext,
  ]);

  return {
    open,
    draft,
    providerOptions,
    providersLoading,
    providersError,
    runtimeModels,
    modelCatalogLoading: catalog.loading,
    modelCatalogLoaded: catalog.loaded,
    modelCatalogRefreshing: catalog.refreshing,
    modelCatalogError: catalog.error,
    submitError,
    isSubmitting: createAgent.isPending || duplicateAgent.isPending,
    hasActiveWorkspace: Boolean(activeWorkspace),
    workspaceName: activeWorkspace?.name ?? null,
    mode,
    duplicateSourceName: duplicateSource?.name ?? null,
    openDialog,
    openForDuplicate,
    onDraftChange,
    onOpenChange,
    onRefreshCatalog,
    onOpenProviderSettings,
    onSubmit,
  };
}
