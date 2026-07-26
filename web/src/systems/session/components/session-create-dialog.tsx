import { useRef, type FormEvent } from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@agh/ui";

import {
  AgentCommandSelect,
  AgentIcon,
  resolveAgentRuntimeValue,
  type AgentPayload,
} from "@/systems/agent";
import {
  NetworkParticipationFields,
  isNetworkParticipationDraftValid,
  type NetworkParticipationDraft,
} from "@/systems/network";
import {
  RuntimeSelector,
  type RuntimeModelOption,
  type RuntimeProviderOption,
  type RuntimeSelectorValue,
} from "@/systems/runtime";
import type { WorkspacePayload } from "@/systems/workspace";

import { validateSessionModelSelection } from "../lib/session-model-selection";
import { SessionCreatePromptComposer } from "./session-create-prompt-composer";

export interface SessionCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agents: AgentPayload[];
  workspace: WorkspacePayload | undefined;
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
  providersLoading: boolean;
  providersError: string | null;
  hasProviderOptions: boolean;
  networkParticipation: NetworkParticipationDraft;
  promptValue: string;
  onPromptChange: (next: string) => void;
  onAgentChange: (agentName: string) => void;
  onRuntimeChange: (next: RuntimeSelectorValue) => void;
  onNetworkParticipationChange: (next: NetworkParticipationDraft) => void;
  onCatalogRefresh: () => void;
  onOpenProviderSettings: () => void;
  onSubmit: () => void;
  isSubmitting: boolean;
  submitError: string | null;
}

function SessionCreateDialog({
  open,
  onOpenChange,
  agents,
  workspace,
  selectedAgentName,
  runtimeValue,
  runtimeProviders,
  runtimeModels,
  catalogStale,
  catalogLoading,
  catalogLoaded,
  catalogError,
  catalogRefreshing,
  catalogRefreshError,
  providersLoading,
  providersError,
  hasProviderOptions,
  networkParticipation,
  promptValue,
  onPromptChange,
  onAgentChange,
  onRuntimeChange,
  onNetworkParticipationChange,
  onCatalogRefresh,
  onOpenProviderSettings,
  onSubmit,
  isSubmitting,
  submitError,
}: SessionCreateDialogProps) {
  // Base UI otherwise focuses the first selectable control.
  const promptRef = useRef<HTMLTextAreaElement>(null);
  const trimmedSelectedAgentName = selectedAgentName.trim();
  const workspaceSelected = workspace !== undefined;
  const activeAgent = workspaceSelected
    ? agents.find(agent => agent.name === trimmedSelectedAgentName)
    : undefined;
  const activeAgentProvider = resolveAgentRuntimeValue(activeAgent).provider;
  const hasAgents = agents.length > 0;
  const hasSelectedAgent = agents.some(agent => agent.name === trimmedSelectedAgentName);
  const hasSelectedProvider = runtimeProviders.some(option => option.id === runtimeValue.provider);
  const modelSelection = validateSessionModelSelection({
    provider: runtimeValue.provider,
    model: runtimeValue.model,
    models: runtimeModels,
    catalogLoading,
    catalogLoaded,
    catalogError,
  });
  const agentPlaceholder = !workspaceSelected
    ? "Select a workspace first"
    : hasAgents
      ? "Select an agent"
      : "No agents available";
  const canSubmit =
    !isSubmitting &&
    !providersLoading &&
    workspaceSelected &&
    hasAgents &&
    hasSelectedAgent &&
    hasProviderOptions &&
    hasSelectedProvider &&
    modelSelection.valid &&
    promptValue.trim().length > 0 &&
    isNetworkParticipationDraftValid(networkParticipation, ["named"]);

  const submitIfAllowed = () => {
    if (!canSubmit) return;
    onSubmit();
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    submitIfAllowed();
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (isSubmitting && !nextOpen) {
      return;
    }
    onOpenChange(nextOpen);
  };

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        // Keep Live participation and inline errors reachable on short viewports.
        className="max-h-[calc(100dvh-4rem)] text-fg sm:max-w-(--width-modal-sm)"
        data-testid="session-create-dialog"
        initialFocus={promptRef}
        showCloseButton={!isSubmitting}
        unframed
      >
        <DialogHeader variant="ruled">
          <DialogTitle>Start a new session</DialogTitle>
          <DialogDescription>
            {workspaceSelected
              ? `Write the first message for a new session in ${workspace.name}. AGH sends it as soon as the runtime starts.`
              : "Choose an active workspace before starting a session."}
          </DialogDescription>
        </DialogHeader>

        <form className="min-h-0 overflow-y-auto" onSubmit={handleSubmit}>
          <div className="space-y-5 p-5">
            <Field>
              <FieldLabel htmlFor="session-create-agent">Agent</FieldLabel>
              <FieldDescription>
                The agent owns the instructions, tools, and provider for this session.
              </FieldDescription>
              <AgentCommandSelect
                agents={agents}
                value={workspaceSelected ? trimmedSelectedAgentName || null : null}
                onChange={next => onAgentChange(next ?? "")}
                disabled={!workspaceSelected || !hasAgents || isSubmitting}
                triggerId="session-create-agent"
                triggerTestId="session-create-agent-select"
                placeholder={agentPlaceholder}
              />
              {activeAgent && activeAgentProvider.length > 0 ? (
                <div
                  className="mt-1 flex items-center gap-1.5 text-form-hint text-subtle"
                  data-testid="session-create-agent-default"
                >
                  <AgentIcon className="size-3 text-subtle" provider={activeAgentProvider} />
                  <span>Effective provider: {activeAgentProvider}</span>
                </div>
              ) : null}
            </Field>

            <NetworkParticipationFields
              allowedStrategies={["named"]}
              disabled={isSubmitting}
              onChange={onNetworkParticipationChange}
              testIdPrefix="session-create-participation"
              value={networkParticipation}
            />

            <SessionCreatePromptComposer
              canSubmit={canSubmit}
              disabled={!workspaceSelected || !hasAgents}
              errorMessageId={submitError ? "session-create-submit-error" : undefined}
              inputRef={promptRef}
              isSubmitting={isSubmitting}
              onChange={onPromptChange}
              onSubmitDraft={submitIfAllowed}
              runtimeControl={
                <RuntimeSelector
                  value={runtimeValue}
                  onChange={onRuntimeChange}
                  providers={runtimeProviders}
                  models={runtimeModels}
                  variant="composer"
                  loading={catalogLoading}
                  catalogLoaded={catalogLoaded}
                  refreshing={catalogRefreshing}
                  onRefreshCatalog={onCatalogRefresh}
                  onOpenProviderSettings={onOpenProviderSettings}
                  catalogStatus={
                    <CatalogStatusLine
                      loading={catalogLoading}
                      refreshing={catalogRefreshing}
                      stale={catalogStale}
                      error={catalogError}
                      refreshError={catalogRefreshError}
                      optionCount={runtimeModels.length}
                    />
                  }
                  disabled={
                    !workspaceSelected || providersLoading || !hasProviderOptions || isSubmitting
                  }
                  triggerId="session-create-runtime"
                  triggerTestId="session-create-runtime-select"
                />
              }
              value={promptValue}
            />

            {providersError ? (
              <p
                className="text-form-hint text-danger"
                data-testid="session-create-providers-error"
                role="alert"
              >
                {providersError}
              </p>
            ) : null}
            {workspaceSelected && !providersLoading && !providersError && !hasProviderOptions ? (
              <p
                className="text-form-hint text-warning"
                data-testid="session-create-providers-empty"
              >
                No providers are configured for this workspace.
              </p>
            ) : null}
            {modelSelection.error ? (
              <FieldError
                className="text-form-hint text-danger"
                data-testid="session-create-model-error"
              >
                {modelSelection.error}
              </FieldError>
            ) : null}

            {isSubmitting ? (
              <p
                aria-live="polite"
                className="text-form-hint text-subtle"
                data-testid="session-create-pending-status"
                role="status"
              >
                Starting the session. It opens as soon as AGH accepts it; the first message is sent
                when the runtime starts.
              </p>
            ) : null}

            {submitError ? (
              <p
                className="text-form-hint text-danger"
                data-testid="session-create-submit-error"
                id="session-create-submit-error"
                role="alert"
              >
                {submitError}
              </p>
            ) : null}
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface CatalogStatusLineProps {
  loading: boolean;
  refreshing: boolean;
  stale: boolean;
  error: string | null;
  refreshError: string | null;
  optionCount: number;
}

function CatalogStatusLine({
  loading,
  refreshing,
  stale,
  error,
  refreshError,
  optionCount,
}: CatalogStatusLineProps) {
  if (refreshError) {
    return (
      <span className="text-danger" data-testid="session-create-catalog-refresh-error" role="alert">
        {refreshError}
      </span>
    );
  }
  if (error) {
    return (
      <span className="text-danger" data-testid="session-create-catalog-error" role="alert">
        {error}. Refresh the catalog or leave Model blank to use the provider default.
      </span>
    );
  }
  if (refreshing) {
    return (
      <span className="text-subtle" data-testid="session-create-catalog-refreshing">
        Refreshing model catalog…
      </span>
    );
  }
  if (loading) {
    return (
      <span className="text-subtle" data-testid="session-create-catalog-loading">
        Loading provider models…
      </span>
    );
  }
  if (stale) {
    return (
      <span className="text-warning" data-testid="session-create-catalog-stale">
        Some models are stale — refresh to confirm availability.
      </span>
    );
  }
  if (optionCount === 0) {
    return (
      <span className="text-subtle" data-testid="session-create-catalog-empty">
        No catalog models. Leave Model blank to use the provider default.
      </span>
    );
  }
  return null;
}

export { SessionCreateDialog };
