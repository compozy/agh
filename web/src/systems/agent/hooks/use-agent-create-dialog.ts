import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { useProviderModels } from "@/systems/model-catalog";
import type { ProviderSelectOption } from "@/systems/runtime";
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
  providerOptions: ProviderSelectOption[];
  providersLoading: boolean;
  providersError: string | null;
  modelOptions: string[];
  modelCatalogLoading: boolean;
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
  onSubmit: () => Promise<void>;
}

function describeError(fallback: string, error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return fallback;
}

function settingsProviderToOption(provider: SettingsProviderEntry): ProviderSelectOption {
  const displayName = provider.settings.display_name?.trim();
  const harness = provider.settings.harness?.trim();
  const runtimeProvider = provider.settings.runtime_provider?.trim();
  return {
    name: provider.name,
    ...(displayName ? { display_name: displayName } : {}),
    ...(harness ? { harness } : {}),
    ...(runtimeProvider ? { runtime_provider: runtimeProvider } : {}),
  };
}

function modelCatalogErrorMessage(error: unknown): string | null {
  if (!error) return null;
  return describeError("Unable to load provider models.", error);
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

  const globalProviders = useMemo<ProviderSelectOption[]>(
    () => settingsProviders.data?.providers.map(settingsProviderToOption) ?? [],
    [settingsProviders.data?.providers]
  );

  const providerOptions = useMemo<ProviderSelectOption[]>(
    () => (draft.scope === "workspace" ? workspaceProviders : globalProviders),
    [draft.scope, globalProviders, workspaceProviders]
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
      if (providerOptions.some(option => option.name === current.provider)) return current;
      return { ...current, provider: "", model: "" };
    });
  }, [providerOptions]);

  const modelCatalog = useProviderModels({
    providerId: draft.provider,
    includeStale: true,
    enabled: open && draft.provider.trim().length > 0,
  });

  const modelOptions = useMemo(
    () => modelCatalog.data?.models.map(model => model.model_id) ?? [],
    [modelCatalog.data?.models]
  );

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
    modelOptions,
    modelCatalogLoading: modelCatalog.isLoading || modelCatalog.isFetching,
    modelCatalogError: modelCatalogErrorMessage(modelCatalog.error),
    submitError,
    isSubmitting: createAgent.isPending,
    hasActiveWorkspace: Boolean(activeWorkspace),
    workspaceName: activeWorkspace?.name ?? null,
    openDialog,
    onDraftChange,
    onOpenChange,
    onSubmit,
  };
}
