import { RefreshCw } from "lucide-react";
import { useMemo, type FormEvent } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldDescription,
  FieldLabel,
  Spinner,
} from "@agh/ui";

import { AgentCommandSelect, AgentIcon, type AgentPayload } from "@/systems/agent";
import {
  modelAvailabilityLabel,
  modelAvailabilityTone,
  type ModelOption,
  type ReasoningOption,
} from "@/systems/model-catalog";
import {
  ModelCommandSelect,
  ProviderCommandSelect,
  ReasoningCommandSelect,
  type ModelSelectOption,
  type ReasoningSelectOption,
} from "@/systems/runtime";
import type { SessionProviderOption, WorkspacePayload } from "@/systems/workspace";

export interface SessionCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agents: AgentPayload[];
  workspace: WorkspacePayload | undefined;
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
  providerOptions: SessionProviderOption[];
  providersLoading: boolean;
  providersError: string | null;
  onAgentChange: (agentName: string) => void;
  onProviderChange: (provider: string) => void;
  onModelChange: (model: string) => void;
  onReasoningChange: (effort: string) => void;
  onCatalogRefresh: () => void;
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
  selectedProvider,
  selectedProviderOption,
  selectedModel,
  selectedReasoning,
  modelOptions,
  reasoningOptions,
  reasoningSupported,
  catalogStale,
  catalogLoading,
  catalogError,
  catalogRefreshing,
  catalogRefreshError,
  defaultReasoning,
  providerOptions,
  providersLoading,
  providersError,
  onAgentChange,
  onProviderChange,
  onModelChange,
  onReasoningChange,
  onCatalogRefresh,
  onSubmit,
  isSubmitting,
  submitError,
}: SessionCreateDialogProps) {
  const trimmedSelectedAgentName = selectedAgentName.trim();
  const trimmedSelectedProvider = selectedProvider.trim();
  const workspaceSelected = workspace !== undefined;
  const activeAgent = workspaceSelected
    ? agents.find(agent => agent.name === trimmedSelectedAgentName)
    : undefined;
  const activeAgentProvider = activeAgent?.provider.trim() ?? "";
  const activeAgentModel =
    activeAgentProvider === trimmedSelectedProvider ? (activeAgent?.model?.trim() ?? "") : "";
  const hasAgents = agents.length > 0;
  const hasProviderOptions = providerOptions.length > 0;
  const hasSelectedAgent = agents.some(agent => agent.name === trimmedSelectedAgentName);
  const hasSelectedProvider = providerOptions.some(
    option => option.name === trimmedSelectedProvider
  );
  const activeProvider = selectedProviderOption ?? undefined;
  const agentPlaceholder = !workspaceSelected
    ? "Select a workspace first"
    : hasAgents
      ? "Select an agent"
      : "No agents available";
  const providerPlaceholder = !workspaceSelected
    ? "Select a workspace first"
    : providersLoading
      ? "Loading providers…"
      : hasProviderOptions
        ? "Select a provider"
        : "No providers available";
  const canSubmit =
    !isSubmitting &&
    !providersLoading &&
    workspaceSelected &&
    hasAgents &&
    hasSelectedAgent &&
    hasProviderOptions &&
    hasSelectedProvider;

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!canSubmit) return;
    onSubmit();
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (isSubmitting && !nextOpen) {
      return;
    }
    onOpenChange(nextOpen);
  };

  const refreshDisabled =
    !hasSelectedProvider || isSubmitting || catalogRefreshing || catalogLoading;

  const modelSelectOptions = useMemo<ModelSelectOption[]>(
    () =>
      modelOptions.map(option => ({
        id: option.id,
        label: option.displayName,
        availability: {
          label: modelAvailabilityLabel(option.availabilityState),
          tone: modelAvailabilityTone(option.availabilityState),
          state: option.availabilityState,
        },
      })),
    [modelOptions]
  );
  const reasoningSelectOptions = useMemo<ReasoningSelectOption[]>(
    () =>
      reasoningOptions.map(option => ({
        value: option.value,
        label: option.label,
        source: option.source,
      })),
    [reasoningOptions]
  );

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        className="text-fg sm:max-w-xl"
        data-testid="session-create-dialog"
        showCloseButton={!isSubmitting}
        unframed
      >
        <DialogHeader variant="ruled">
          <DialogTitle>Start a new session</DialogTitle>
          <DialogDescription>
            {workspaceSelected
              ? `Pick the agent and provider runtime for this session in ${workspace.name}.`
              : "Choose an active workspace before starting a session."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-5 p-5">
            <Field>
              <FieldLabel htmlFor="session-create-agent">Agent</FieldLabel>
              <FieldDescription>
                The agent owns the default prompt, tools, and provider for this session.
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
                  className="mt-1 flex items-center gap-1.5 text-xs text-subtle"
                  data-testid="session-create-agent-default"
                >
                  <AgentIcon className="size-3 text-subtle" provider={activeAgentProvider} />
                  <span>Agent default provider: {activeAgentProvider}</span>
                </div>
              ) : null}
            </Field>

            <Field>
              <FieldLabel htmlFor="session-create-provider">Provider</FieldLabel>
              <FieldDescription>
                Override the runtime for this session only. The agent default is preselected when it
                matches a provider visible in this workspace.
              </FieldDescription>
              <ProviderCommandSelect
                options={providerOptions}
                value={workspaceSelected ? trimmedSelectedProvider || null : null}
                onChange={next => onProviderChange(next ?? "")}
                disabled={
                  !workspaceSelected || providersLoading || !hasProviderOptions || isSubmitting
                }
                triggerId="session-create-provider"
                triggerTestId="session-create-provider-select"
                placeholder={providerPlaceholder}
              />
              {activeProvider ? (
                <div
                  className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-eyebrow text-subtle"
                  data-testid="session-create-provider-runtime"
                >
                  <span>{activeProvider.harness ?? "acp"}</span>
                  {activeProvider.runtime_provider ? (
                    <span>{activeProvider.runtime_provider}</span>
                  ) : null}
                </div>
              ) : null}
              {providersError ? (
                <p
                  className="mt-1 text-xs text-danger"
                  data-testid="session-create-providers-error"
                  role="alert"
                >
                  {providersError}
                </p>
              ) : null}
              {workspaceSelected && !providersLoading && !providersError && !hasProviderOptions ? (
                <p
                  className="mt-1 text-xs text-warning"
                  data-testid="session-create-providers-empty"
                >
                  No providers are configured for this workspace.
                </p>
              ) : null}
            </Field>

            <div className="grid gap-5 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="session-create-model">Model</FieldLabel>
                <FieldDescription>
                  Override the model for this session, or inherit the provider default.
                </FieldDescription>
                <ModelCommandSelect
                  options={modelSelectOptions}
                  defaultModel={activeAgentModel || null}
                  value={selectedModel}
                  onChange={onModelChange}
                  disabled={!workspaceSelected || !hasSelectedProvider || isSubmitting}
                  triggerId="session-create-model"
                  triggerTestId="session-create-model-select"
                />
                <CatalogStatusLine
                  loading={catalogLoading}
                  refreshing={catalogRefreshing}
                  stale={catalogStale}
                  error={catalogError}
                  refreshError={catalogRefreshError}
                  optionCount={modelOptions.length}
                />
                {hasSelectedProvider ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={onCatalogRefresh}
                    disabled={refreshDisabled}
                    data-testid="session-create-catalog-refresh"
                    aria-label="Refresh provider model catalog"
                    className="mt-1 w-fit"
                  >
                    <RefreshCw
                      aria-hidden="true"
                      className={catalogRefreshing ? "size-3 animate-spin" : "size-3"}
                    />
                    Refresh catalog
                  </Button>
                ) : null}
              </Field>

              <Field>
                <FieldLabel htmlFor="session-create-reasoning">Reasoning effort</FieldLabel>
                <FieldDescription>
                  Hint reasoning depth when the selected provider supports it.
                </FieldDescription>
                <ReasoningCommandSelect
                  options={reasoningSelectOptions}
                  value={selectedReasoning}
                  onChange={onReasoningChange}
                  disabled={
                    !workspaceSelected ||
                    !hasSelectedProvider ||
                    !reasoningSupported ||
                    isSubmitting
                  }
                  disabledHint={
                    hasSelectedProvider && !reasoningSupported
                      ? "Selected model does not advertise reasoning effort"
                      : undefined
                  }
                  triggerId="session-create-reasoning"
                  triggerTestId="session-create-reasoning-select"
                />
                {defaultReasoning ? (
                  <p
                    className="mt-1 text-xs text-subtle"
                    data-testid="session-create-reasoning-default"
                  >
                    Default reasoning: {defaultReasoning}
                  </p>
                ) : null}
              </Field>
            </div>

            {submitError ? (
              <p
                className="text-xs text-danger"
                data-testid="session-create-submit-error"
                role="alert"
              >
                {submitError}
              </p>
            ) : null}
          </div>

          <DialogFooter variant="ruled">
            <Button
              data-testid="session-create-dialog-cancel"
              disabled={isSubmitting}
              onClick={() => handleOpenChange(false)}
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
            <Button data-testid="session-create-dialog-submit" disabled={!canSubmit} type="submit">
              {isSubmitting ? <Spinner aria-hidden="true" /> : null}
              Start session
            </Button>
          </DialogFooter>
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
      <p
        className="mt-1 text-xs text-danger"
        data-testid="session-create-catalog-refresh-error"
        role="alert"
      >
        {refreshError}
      </p>
    );
  }
  if (error) {
    return (
      <p
        className="mt-1 text-xs text-danger"
        data-testid="session-create-catalog-error"
        role="alert"
      >
        {error}. Type a model name to continue.
      </p>
    );
  }
  if (refreshing) {
    return (
      <p className="mt-1 text-xs text-subtle" data-testid="session-create-catalog-refreshing">
        Refreshing model catalog…
      </p>
    );
  }
  if (loading) {
    return (
      <p className="mt-1 text-xs text-subtle" data-testid="session-create-catalog-loading">
        Loading provider models…
      </p>
    );
  }
  if (stale) {
    return (
      <p className="mt-1 text-xs text-warning" data-testid="session-create-catalog-stale">
        Some models are stale , refresh to confirm availability.
      </p>
    );
  }
  if (optionCount === 0) {
    return (
      <p className="mt-1 text-xs text-subtle" data-testid="session-create-catalog-empty">
        No catalog models , type a model name to continue.
      </p>
    );
  }
  return null;
}

export { SessionCreateDialog };
