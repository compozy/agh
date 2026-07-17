import { useEffect, useRef, useState } from "react";
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
import {
  networkParticipationDraftFromValues,
  networkParticipationValidationMessage,
  serializeNetworkParticipation,
  type NetworkParticipationDraft,
  type NetworkParticipationStrategy,
} from "@/systems/network";

import {
  MODEL_CATALOG_PENDING,
  validateSessionModelSelection,
} from "../lib/session-model-selection";
import type { SessionPayload } from "../types";
import { useCreateSession } from "./use-session-actions";

interface SessionCreateDialogContext {
  agents: AgentPayload[] | undefined;
  activeWorkspace: WorkspacePayload | undefined;
}

interface SessionNavigationTarget {
  agentName: string;
  sessionId: string;
}

export interface SessionCreateDialogDraft {
  agentName: string;
  providerOverride: string;
  modelOverride: string;
  reasoningEffort: ReasoningEffort | "";
  networkParticipationMode: "local" | "live";
  networkChannelId: string;
  networkChannelStrategy: NetworkParticipationStrategy | "";
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
  onOpenChange: (open: boolean) => void;
  onAgentChange: (agentName: string) => void;
  onRuntimeChange: (next: RuntimeSelectorValue) => void;
  onNetworkParticipationChange: (next: NetworkParticipationDraft) => void;
  networkParticipation: NetworkParticipationDraft;
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

  const providerOptions: SessionProviderOption[] = workspaceDetail?.providers ?? [];

  const [open, setOpenState] = useState(false);
  const [draft, setDraft] = useState<SessionCreateDialogDraft>({
    agentName: "",
    providerOverride: "",
    modelOverride: "",
    reasoningEffort: "",
    networkParticipationMode: "local",
    networkChannelId: "",
    networkChannelStrategy: "",
  });
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [pendingAgentName, setPendingAgentName] = useState<string | null>(null);
  const [pendingWorkspaceId, setPendingWorkspaceId] = useState<string | null>(null);
  const [navigationTarget, setNavigationTarget] = useState<SessionNavigationTarget | null>(null);
  const navigatedTarget = useRef<SessionNavigationTarget | null>(null);

  useEffect(() => {
    if (!navigationTarget || navigatedTarget.current === navigationTarget) return;
    navigatedTarget.current = navigationTarget;
    void navigate({
      to: "/agents/$name/sessions/$id",
      params: { name: navigationTarget.agentName, id: navigationTarget.sessionId },
    });
  }, [navigate, navigationTarget]);

  const agentList = agents ?? [];
  const selectedAgent = agentList.find(agent => agent.name === draft.agentName);
  const selectedProvider = resolveSelectedProvider(
    draft.agentName,
    draft.providerOverride,
    selectedAgent,
    providerOptions
  );

  const runtimeProviders: RuntimeProviderOption[] = providerOptions.map(option => ({
    id: option.name,
    name: option.display_name?.trim() || option.name,
    ...(option.harness?.trim() ? { harness: option.harness.trim() } : {}),
    runtime_provider: option.runtime_provider?.trim() || option.name,
  }));

  // Browse/search span every provider available to the workspace via the single
  // aggregate catalog query, filtered to those providers; the selected provider's
  // raw payloads still drive reasoning-support derivation and the stale indicator.
  const catalogProviders: RuntimeCatalogProvider[] = runtimeProviders.map(entry => ({
    id: entry.id,
  }));
  const catalog = useRuntimeModelCatalog(catalogProviders, { enabled: open });
  const runtimeModels = catalog.models;
  const catalogModels: ProviderModelPayload[] = catalog.payloadsByProvider[selectedProvider] ?? [];

  const trimmedSelectedModel = draft.modelOverride.trim();
  const trimmedAgentProvider = selectedAgent?.provider.trim() ?? "";
  const trimmedAgentModel =
    trimmedAgentProvider === selectedProvider ? (selectedAgent?.model?.trim() ?? "") : "";
  const effectiveSelectedModel = trimmedSelectedModel || trimmedAgentModel;

  const reasoningSupported = deriveActiveSessionOptions({
    catalog: catalogModels,
    selectedModel: effectiveSelectedModel.length > 0 ? effectiveSelectedModel : null,
  }).reasoningSupported;

  const selectedReasoning = reasoningSupported ? draft.reasoningEffort : "";

  // Render the EFFECTIVE model (explicit override or the inherited agent default)
  // so the selector shows the inherited model and can offer its reasoning. The
  // POST still omits `model` while it remains inherited (see `submit`).
  const runtimeValue: RuntimeSelectorValue = {
    provider: selectedProvider,
    model: effectiveSelectedModel,
    reasoning_effort: selectedReasoning,
  };

  const modelSelection = validateSessionModelSelection({
    provider: selectedProvider,
    model: effectiveSelectedModel,
    models: runtimeModels,
    catalogLoading: catalog.loading,
    catalogLoaded: catalog.loaded,
    catalogError: catalog.error,
  });

  const catalogStale = catalog.stale;
  const catalogLoading = catalog.loading;
  const catalogLoaded = catalog.loaded;
  const catalogError = catalog.error;
  const catalogRefreshError = catalog.refreshError;

  const openForAgent = (agentName: string) => {
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
      networkParticipationMode: "local",
      networkChannelId: "",
      networkChannelStrategy: "",
    });
    setSubmitError(null);
    setOpenState(true);
  };

  const handleOpenChange = (next: boolean) => {
    setOpenState(next);
    if (!next) {
      setSubmitError(null);
    }
  };

  const onAgentChange = (agentName: string) => {
    setSubmitError(null);
    setDraft({
      agentName,
      providerOverride: "",
      modelOverride: "",
      reasoningEffort: "",
      networkParticipationMode: "local",
      networkChannelId: "",
      networkChannelStrategy: "",
    });
  };

  const onNetworkParticipationChange = (next: NetworkParticipationDraft) => {
    setSubmitError(null);
    setDraft(current => ({
      ...current,
      networkParticipationMode: next.mode,
      networkChannelId: next.channelId,
      networkChannelStrategy: next.channelStrategy,
    }));
  };

  const onRuntimeChange = (next: RuntimeSelectorValue) => {
    setSubmitError(null);
    setDraft(current => ({
      ...current,
      providerOverride: next.provider,
      modelOverride: next.model,
      reasoningEffort: normalizeEffort(next.reasoning_effort),
    }));
  };

  const refreshCatalog = catalog.refresh;

  const openProviderSettings = () => {
    setOpenState(false);
    void navigate({ to: "/settings/providers" });
  };

  const submit = async () => {
    if (!activeWorkspace) return;
    const agentName = draft.agentName.trim();
    const provider = selectedProvider.trim();
    if (agentName.length === 0 || provider.length === 0) return;
    if (!modelSelection.valid) {
      setSubmitError(modelSelection.error ?? MODEL_CATALOG_PENDING);
      return;
    }
    const networkParticipation = networkParticipationDraftFromValues(
      draft.networkParticipationMode,
      draft.networkChannelId,
      draft.networkChannelStrategy
    );
    const participationError = networkParticipationValidationMessage(networkParticipation, [
      "named",
    ]);
    if (participationError) {
      setSubmitError(participationError);
      return;
    }

    setSubmitError(null);
    setPendingAgentName(agentName);
    setPendingWorkspaceId(activeWorkspace.id);

    // Send `model` only when the effective model diverges from the agent default
    // for this provider; an inherited model stays omitted so the daemon resolves
    // the agent default — while reasoning_effort can still ship on its own.
    const modelDiffersFromDefault =
      effectiveSelectedModel.length > 0 && effectiveSelectedModel !== trimmedAgentModel;
    const reasoningEffort = selectedReasoning === "" ? undefined : selectedReasoning;

    let session: SessionPayload;
    try {
      session = await createSession.mutateAsync({
        agent_name: agentName,
        workspace: activeWorkspace.id,
        provider,
        ...(modelDiffersFromDefault ? { model: effectiveSelectedModel } : {}),
        ...(reasoningEffort ? { reasoning_effort: reasoningEffort } : {}),
        network_participation: serializeNetworkParticipation(networkParticipation),
      });
    } catch (error) {
      const message = describeError("Failed to create session.", error);
      setSubmitError(message);
      toast.error(message);
      setPendingAgentName(null);
      setPendingWorkspaceId(null);
      return;
    }

    setOpenState(false);
    setPendingAgentName(null);
    setPendingWorkspaceId(null);
    setNavigationTarget({
      agentName: session.agent_name,
      sessionId: session.id,
    });
  };

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
    onOpenChange: handleOpenChange,
    onAgentChange,
    onRuntimeChange,
    onNetworkParticipationChange,
    networkParticipation: networkParticipationDraftFromValues(
      draft.networkParticipationMode,
      draft.networkChannelId,
      draft.networkChannelStrategy
    ),
    refreshCatalog,
    openProviderSettings,
    submit,
  };
}
