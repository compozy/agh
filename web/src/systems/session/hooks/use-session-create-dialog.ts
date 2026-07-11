import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import type { AgentPayload } from "@/systems/agent";
import { isReasoningEffort, type ReasoningEffort } from "@/lib/api-contract";
import {
  deriveActiveSessionOptions,
  useRuntimeModelCatalog,
  type ProviderModelPayload,
  type RuntimeCatalogProvider,
} from "@/systems/model-catalog";
import type {
  RuntimeModelOption,
  RuntimeProviderOption,
  RuntimeSelectorValue,
} from "@/systems/runtime";
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
  reasoningEffort: ReasoningEffort | "";
}

export interface SessionCreateDialogState {
  open: boolean;
  agents: AgentPayload[];
  workspace: WorkspacePayload | undefined;
  providersLoading: boolean;
  providersError: string | null;
  hasProviderOptions: boolean;
  selectedAgentName: string;
  runtimeValue: RuntimeSelectorValue;
  runtimeProviders: RuntimeProviderOption[];
  runtimeModels: RuntimeModelOption[];
  catalogStale: boolean;
  catalogLoading: boolean;
  catalogLoaded: boolean;
  catalogError: string | null;
  catalogRefreshing: boolean;
  catalogRefreshError: string | null;
  isSubmitting: boolean;
  submitError: string | null;
  pendingAgentName: string | null;
  pendingWorkspaceId: string | null;
}

export interface SessionCreateDialogApi extends SessionCreateDialogState {
  openForAgent: (agentName: string) => void;
  setOpen: (open: boolean) => void;
  onAgentChange: (agentName: string) => void;
  onRuntimeChange: (next: RuntimeSelectorValue) => void;
  refreshCatalog: () => void;
  openProviderSettings: () => void;
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

function normalizeEffort(effort: string): ReasoningEffort | "" {
  return effort === "" ? "" : isReasoningEffort(effort) ? effort : "";
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

  const runtimeProviders = useMemo<RuntimeProviderOption[]>(
    () =>
      providerOptions.map(option => ({
        id: option.name,
        name: option.display_name?.trim() || option.name,
        ...(option.harness?.trim() ? { harness: option.harness.trim() } : {}),
        runtime_provider: option.runtime_provider?.trim() || option.name,
      })),
    [providerOptions]
  );

  // Browse/search span every provider available to the workspace via the single
  // aggregate catalog query, filtered to those providers; the selected provider's
  // raw payloads still drive reasoning-support derivation and the stale indicator.
  const catalogProviders = useMemo<RuntimeCatalogProvider[]>(
    () => runtimeProviders.map(entry => ({ id: entry.id })),
    [runtimeProviders]
  );
  const catalog = useRuntimeModelCatalog(catalogProviders, { enabled: open });
  const runtimeModels = catalog.models;
  const catalogModels = useMemo<ProviderModelPayload[]>(
    () => catalog.payloadsByProvider[selectedProvider] ?? [],
    [catalog.payloadsByProvider, selectedProvider]
  );

  const trimmedSelectedModel = useMemo(() => draft.modelOverride.trim(), [draft.modelOverride]);
  const trimmedAgentProvider = selectedAgent?.provider.trim() ?? "";
  const trimmedAgentModel = useMemo(() => {
    if (trimmedAgentProvider !== selectedProvider) {
      return "";
    }
    return selectedAgent?.model?.trim() ?? "";
  }, [selectedAgent?.model, selectedProvider, trimmedAgentProvider]);
  const effectiveSelectedModel = trimmedSelectedModel || trimmedAgentModel;

  const reasoningSupported = useMemo(
    () =>
      deriveActiveSessionOptions({
        catalog: catalogModels,
        selectedModel: effectiveSelectedModel.length > 0 ? effectiveSelectedModel : null,
      }).reasoningSupported,
    [catalogModels, effectiveSelectedModel]
  );

  const selectedReasoning = reasoningSupported ? draft.reasoningEffort : "";

  // Render the EFFECTIVE model (explicit override or the inherited agent default)
  // so the selector shows the inherited model and can offer its reasoning. The
  // POST still omits `model` while it remains inherited (see `submit`).
  const runtimeValue = useMemo<RuntimeSelectorValue>(
    () => ({
      provider: selectedProvider,
      model: effectiveSelectedModel,
      reasoning_effort: selectedReasoning,
    }),
    [selectedProvider, effectiveSelectedModel, selectedReasoning]
  );

  const catalogStale = catalog.stale;
  const catalogLoading = catalog.loading;
  const catalogLoaded = catalog.loaded;
  const catalogError = catalog.error;
  const catalogRefreshError = catalog.refreshError;

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
    setDraft({ agentName, providerOverride: "", modelOverride: "", reasoningEffort: "" });
  }, []);

  const onRuntimeChange = useCallback((next: RuntimeSelectorValue) => {
    setDraft(current => ({
      ...current,
      providerOverride: next.provider,
      modelOverride: next.model,
      reasoningEffort: normalizeEffort(next.reasoning_effort),
    }));
  }, []);

  const refreshCatalog = catalog.refresh;

  const openProviderSettings = useCallback(() => {
    setOpenState(false);
    void navigate({ to: "/settings/providers" });
  }, [navigate]);

  const submit = useCallback(async () => {
    if (!activeWorkspace) return;
    const agentName = draft.agentName.trim();
    const provider = selectedProvider.trim();
    if (agentName.length === 0 || provider.length === 0) return;

    setSubmitError(null);
    setPendingAgentName(agentName);
    setPendingWorkspaceId(activeWorkspace.id);

    // Send `model` only when the effective model diverges from the agent default
    // for this provider; an inherited model stays omitted so the daemon resolves
    // the agent default — while reasoning_effort can still ship on its own.
    const modelDiffersFromDefault =
      effectiveSelectedModel.length > 0 && effectiveSelectedModel !== trimmedAgentModel;
    const reasoningEffort = selectedReasoning === "" ? undefined : selectedReasoning;

    try {
      const session = await createSession.mutateAsync({
        agent_name: agentName,
        workspace: activeWorkspace.id,
        provider,
        ...(modelDiffersFromDefault ? { model: effectiveSelectedModel } : {}),
        ...(reasoningEffort ? { reasoning_effort: reasoningEffort } : {}),
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
    effectiveSelectedModel,
    navigate,
    selectedProvider,
    selectedReasoning,
    trimmedAgentModel,
  ]);

  const providersError = workspaceDetailError ? describeWorkspaceError(workspaceDetailError) : null;

  return {
    open,
    agents: agentList,
    workspace: activeWorkspace,
    providersLoading: workspaceId.length > 0 && workspaceDetailLoading,
    providersError,
    hasProviderOptions: providerOptions.length > 0,
    selectedAgentName: draft.agentName,
    runtimeValue,
    runtimeProviders,
    runtimeModels,
    catalogStale,
    catalogLoading,
    catalogLoaded,
    catalogError,
    catalogRefreshing: catalog.refreshing,
    catalogRefreshError,
    isSubmitting: createSession.isPending,
    submitError,
    pendingAgentName,
    pendingWorkspaceId,
    openForAgent,
    setOpen,
    onAgentChange,
    onRuntimeChange,
    refreshCatalog,
    openProviderSettings,
    submit,
  };
}
