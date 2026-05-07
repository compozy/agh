import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import type { AgentPayload } from "@/systems/agent";
import {
  deriveActiveSessionOptions,
  useProviderModels,
  useRefreshProviderModels,
  type ModelOption,
  type ProviderModelPayload,
  type ReasoningOption,
} from "@/systems/model-catalog";
import type { SessionProviderOption, WorkspacePayload } from "@/systems/workspace";
import { useWorkspace } from "@/systems/workspace";

import { useCreateSession } from "./use-session-actions";

interface SessionCreateDialogContext {
  agents: AgentPayload[] | undefined;
  activeWorkspace: WorkspacePayload | undefined;
}

export interface SessionCreateDialogDraft {
  agentName: string;
  providerOverride: string;
  modelOverride: string;
  reasoningEffort: string;
}

export interface SessionCreateDialogState {
  open: boolean;
  agents: AgentPayload[];
  workspace: WorkspacePayload | undefined;
  providerOptions: SessionProviderOption[];
  providersLoading: boolean;
  providersError: string | null;
  selectedAgentName: string;
  selectedProvider: string;
  selectedProviderOption: SessionProviderOption | undefined;
  selectedModel: string;
  selectedReasoning: string;
  modelOptions: ModelOption[];
  reasoningOptions: ReasoningOption[];
  reasoningSupported: boolean;
  catalogStale: boolean;
  catalogLoading: boolean;
  catalogError: string | null;
  catalogRefreshing: boolean;
  catalogRefreshError: string | null;
  defaultReasoning: string | null;
  isSubmitting: boolean;
  submitError: string | null;
  pendingAgentName: string | null;
  pendingWorkspaceId: string | null;
}

export interface SessionCreateDialogApi extends SessionCreateDialogState {
  openForAgent: (agentName: string) => void;
  setOpen: (open: boolean) => void;
  onAgentChange: (agentName: string) => void;
  onProviderChange: (provider: string) => void;
  onModelChange: (model: string) => void;
  onReasoningChange: (effort: string) => void;
  refreshCatalog: () => void;
  submit: () => Promise<void>;
}

function pickDefaultProvider(
  agent: AgentPayload | undefined,
  options: SessionProviderOption[]
): string {
  if (options.length === 0) {
    return "";
  }
  if (agent && options.some(option => option.name === agent.provider)) {
    return agent.provider;
  }
  return options[0]?.name ?? "";
}

function resolveSelectedProvider(
  agentName: string,
  providerOverride: string,
  agent: AgentPayload | undefined,
  options: SessionProviderOption[]
): string {
  if (providerOverride.length > 0 && options.some(option => option.name === providerOverride)) {
    return providerOverride;
  }
  if (agentName.trim().length === 0) {
    return "";
  }
  return pickDefaultProvider(agent, options);
}

function describeWorkspaceError(error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return "Unable to load provider options for this workspace.";
}

function describeError(fallback: string, error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return fallback;
}

export function useSessionCreateDialog({
  agents,
  activeWorkspace,
}: SessionCreateDialogContext): SessionCreateDialogApi {
  const navigate = useNavigate();
  const createSession = useCreateSession();
  const workspaceId = activeWorkspace?.id ?? "";
  const {
    data: workspaceDetail,
    isLoading: workspaceDetailLoading,
    error: workspaceDetailError,
  } = useWorkspace(workspaceId, { enabled: workspaceId.length > 0 });

  const providerOptions = useMemo<SessionProviderOption[]>(
    () => workspaceDetail?.providers ?? [],
    [workspaceDetail?.providers]
  );

  const [open, setOpenState] = useState(false);
  const [draft, setDraft] = useState<SessionCreateDialogDraft>({
    agentName: "",
    providerOverride: "",
    modelOverride: "",
    reasoningEffort: "",
  });
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [pendingAgentName, setPendingAgentName] = useState<string | null>(null);
  const [pendingWorkspaceId, setPendingWorkspaceId] = useState<string | null>(null);

  const agentList = useMemo(() => agents ?? [], [agents]);
  const selectedAgent = useMemo(
    () => agentList.find(agent => agent.name === draft.agentName),
    [agentList, draft.agentName]
  );
  const selectedProvider = useMemo(
    () =>
      resolveSelectedProvider(
        draft.agentName,
        draft.providerOverride,
        selectedAgent,
        providerOptions
      ),
    [draft.agentName, draft.providerOverride, providerOptions, selectedAgent]
  );

  const selectedProviderOption = useMemo(
    () => providerOptions.find(option => option.name === selectedProvider),
    [providerOptions, selectedProvider]
  );

  const catalogQuery = useProviderModels({
    providerId: selectedProvider,
    includeStale: true,
    enabled: open && selectedProvider.length > 0,
  });

  const refreshMutation = useRefreshProviderModels();

  const catalogModels = useMemo<ProviderModelPayload[]>(
    () => catalogQuery.data?.models ?? [],
    [catalogQuery.data?.models]
  );

  const trimmedSelectedModel = useMemo(() => draft.modelOverride.trim(), [draft.modelOverride]);

  const derived = useMemo(
    () =>
      deriveActiveSessionOptions({
        catalog: catalogModels,
        selectedModel: trimmedSelectedModel.length > 0 ? trimmedSelectedModel : null,
      }),
    [catalogModels, trimmedSelectedModel]
  );

  const catalogStale = useMemo(() => catalogModels.some(model => model.stale), [catalogModels]);
  const catalogLoading = catalogQuery.isLoading || catalogQuery.isFetching;
  const catalogError = catalogQuery.error
    ? describeError("Failed to load provider models.", catalogQuery.error)
    : null;
  const catalogRefreshError = refreshMutation.error
    ? describeError("Failed to refresh provider models.", refreshMutation.error)
    : null;

  const reasoningSupported = derived.reasoningSupported;

  const selectedReasoning = useMemo(() => {
    if (!reasoningSupported) return "";
    return draft.reasoningEffort.trim();
  }, [draft.reasoningEffort, reasoningSupported]);

  const openForAgent = useCallback(
    (agentName: string) => {
      if (!activeWorkspace) {
        toast.error("Select an active workspace before starting a session.");
        return;
      }

      const matched = agentList.find(agent => agent.name === agentName) ?? agentList[0];
      const nextAgentName = matched?.name ?? agentName;

      setDraft({
        agentName: nextAgentName,
        providerOverride: "",
        modelOverride: "",
        reasoningEffort: "",
      });
      setSubmitError(null);
      setOpenState(true);
    },
    [activeWorkspace, agentList]
  );

  const setOpen = useCallback((next: boolean) => {
    setOpenState(next);
    if (!next) {
      setSubmitError(null);
    }
  }, []);

  const onAgentChange = useCallback((agentName: string) => {
    setDraft({
      agentName,
      providerOverride: "",
      modelOverride: "",
      reasoningEffort: "",
    });
  }, []);

  const onProviderChange = useCallback((provider: string) => {
    setDraft(current => ({
      ...current,
      providerOverride: provider,
      modelOverride: "",
      reasoningEffort: "",
    }));
  }, []);

  const onModelChange = useCallback((model: string) => {
    setDraft(current => ({ ...current, modelOverride: model, reasoningEffort: "" }));
  }, []);

  const onReasoningChange = useCallback((effort: string) => {
    setDraft(current => ({ ...current, reasoningEffort: effort }));
  }, []);

  const refreshCatalog = useCallback(() => {
    if (selectedProvider.length === 0) {
      return;
    }
    refreshMutation.mutate({ providerId: selectedProvider, force: true });
  }, [refreshMutation, selectedProvider]);

  const submit = useCallback(async () => {
    if (!activeWorkspace) return;
    const agentName = draft.agentName.trim();
    const provider = selectedProvider.trim();
    if (agentName.length === 0 || provider.length === 0) return;

    setSubmitError(null);
    setPendingAgentName(agentName);
    setPendingWorkspaceId(activeWorkspace.id);

    const trimmedModel = trimmedSelectedModel;
    const trimmedReasoning = selectedReasoning.trim();

    try {
      const session = await createSession.mutateAsync({
        agent_name: agentName,
        workspace: activeWorkspace.id,
        provider,
        ...(trimmedModel.length > 0 ? { model: trimmedModel } : {}),
        ...(trimmedReasoning.length > 0 ? { reasoning_effort: trimmedReasoning } : {}),
      });
      setOpenState(false);
      await navigate({
        to: "/agents/$name/sessions/$id",
        params: { name: session.agent_name, id: session.id },
      });
    } catch (error) {
      const message = describeError("Failed to create session.", error);
      setSubmitError(message);
      toast.error(message);
    } finally {
      setPendingAgentName(null);
      setPendingWorkspaceId(null);
    }
  }, [
    activeWorkspace,
    createSession,
    draft.agentName,
    navigate,
    selectedProvider,
    selectedReasoning,
    trimmedSelectedModel,
  ]);

  const providersError = workspaceDetailError ? describeWorkspaceError(workspaceDetailError) : null;

  return {
    open,
    agents: agentList,
    workspace: activeWorkspace,
    providerOptions,
    providersLoading: workspaceId.length > 0 && workspaceDetailLoading,
    providersError,
    selectedAgentName: draft.agentName,
    selectedProvider,
    selectedProviderOption,
    selectedModel: trimmedSelectedModel,
    selectedReasoning,
    modelOptions: derived.modelOptions,
    reasoningOptions: derived.reasoningOptions,
    reasoningSupported,
    catalogStale,
    catalogLoading,
    catalogError,
    catalogRefreshing: refreshMutation.isPending,
    catalogRefreshError,
    defaultReasoning: derived.defaultReasoning,
    isSubmitting: createSession.isPending,
    submitError,
    pendingAgentName,
    pendingWorkspaceId,
    openForAgent,
    setOpen,
    onAgentChange,
    onProviderChange,
    onModelChange,
    onReasoningChange,
    refreshCatalog,
    submit,
  };
}
