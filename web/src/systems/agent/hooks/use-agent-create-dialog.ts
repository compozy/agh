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

import { useCreateAgent } from "./use-agents";
import {
  buildCreateAgentParams,
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
}

export interface AgentCreateDialogApi extends AgentCreateDialogState {
  openDialog: () => void;
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
  const settingsProviders = useSettingsProviders();
  const [open, setOpenState] = useState(false);
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

  // Browse/search span every provider available to the current scope via one
  // aggregate catalog query filtered to those providers.
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

  const openDialog = useCallback(() => {
    setDraft(createDefaultAgentCreateDraft(Boolean(activeWorkspace)));
    setSubmitError(null);
    setOpenState(true);
  }, [activeWorkspace]);

  const onOpenChange = useCallback(
    (next: boolean) => {
      setOpenState(next);
      if (!next) {
        setSubmitError(null);
        setDraft(createDefaultAgentCreateDraft(Boolean(activeWorkspace)));
      }
    },
    [activeWorkspace]
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
    const request = buildCreateAgentParams(draft, activeWorkspace?.id, validationContext);
    if (!request) {
      const validation = validateAgentCreateDraft(draft, validationContext);
      const message =
        Object.values(validation.fields).find(field => field && field.length > 0) ??
        "Fix the highlighted fields before creating an agent.";
      setSubmitError(message);
      return;
    }

    setSubmitError(null);
    try {
      const agent: AgentPayload = await createAgent.mutateAsync(request);
      setOpenState(false);
      setDraft(createDefaultAgentCreateDraft(Boolean(activeWorkspace)));
      await navigate({
        to: "/agents/$name",
        params: { name: agent.name },
      });
    } catch (error) {
      const message = describeError("Failed to create agent.", error);
      setSubmitError(message);
      toast.error(message);
    }
  }, [activeWorkspace, createAgent, draft, navigate, validationContext]);

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
    isSubmitting: createAgent.isPending,
    hasActiveWorkspace: Boolean(activeWorkspace),
    workspaceName: activeWorkspace?.name ?? null,
    openDialog,
    onDraftChange,
    onOpenChange,
    onRefreshCatalog,
    onOpenProviderSettings,
    onSubmit,
  };
}
