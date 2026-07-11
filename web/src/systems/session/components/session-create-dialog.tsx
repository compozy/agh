import { type FormEvent } from "react";

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
  FieldTitle,
  Spinner,
} from "@agh/ui";

import { AgentCommandSelect, AgentIcon, type AgentPayload } from "@/systems/agent";
import {
  RuntimeSelector,
  type RuntimeModelOption,
  type RuntimeProviderOption,
  type RuntimeSelectorValue,
} from "@/systems/runtime";
import type { WorkspacePayload } from "@/systems/workspace";

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
  onAgentChange: (agentName: string) => void;
  onRuntimeChange: (next: RuntimeSelectorValue) => void;
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
  onAgentChange,
  onRuntimeChange,
  onCatalogRefresh,
  onOpenProviderSettings,
  onSubmit,
  isSubmitting,
  submitError,
}: SessionCreateDialogProps) {
  const trimmedSelectedAgentName = selectedAgentName.trim();
  const workspaceSelected = workspace !== undefined;
  const activeAgent = workspaceSelected
    ? agents.find(agent => agent.name === trimmedSelectedAgentName)
    : undefined;
  const activeAgentProvider = activeAgent?.provider.trim() ?? "";
  const hasAgents = agents.length > 0;
  const hasSelectedAgent = agents.some(agent => agent.name === trimmedSelectedAgentName);
  const hasSelectedProvider = runtimeProviders.some(option => option.id === runtimeValue.provider);
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

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        className="text-fg sm:max-w-(--width-modal-sm)"
        data-testid="session-create-dialog"
        showCloseButton={!isSubmitting}
        unframed
      >
        <DialogHeader variant="ruled">
          <DialogTitle>Start a new session</DialogTitle>
          <DialogDescription>
            {workspaceSelected
              ? `Pick the agent and runtime for this session in ${workspace.name}.`
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
                  className="mt-1 flex items-center gap-1.5 text-form-hint text-subtle"
                  data-testid="session-create-agent-default"
                >
                  <AgentIcon className="size-3 text-subtle" provider={activeAgentProvider} />
                  <span>Agent default provider: {activeAgentProvider}</span>
                </div>
              ) : null}
            </Field>

            <Field>
              <FieldTitle id="session-create-runtime-label">Runtime</FieldTitle>
              <FieldDescription>
                Provider, model, and reasoning effort for this session. The agent default is
                preselected; override any axis for this session only.
              </FieldDescription>
              <RuntimeSelector
                value={runtimeValue}
                onChange={onRuntimeChange}
                providers={runtimeProviders}
                models={runtimeModels}
                loading={catalogLoading}
                catalogLoaded={catalogLoaded}
                refreshing={catalogRefreshing}
                onRefreshCatalog={onCatalogRefresh}
                onOpenProviderSettings={onOpenProviderSettings}
                ariaLabelledby="session-create-runtime-label"
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
              {providersError ? (
                <p
                  className="mt-1 text-form-hint text-danger"
                  data-testid="session-create-providers-error"
                  role="alert"
                >
                  {providersError}
                </p>
              ) : null}
              {workspaceSelected && !providersLoading && !providersError && !hasProviderOptions ? (
                <p
                  className="mt-1 text-form-hint text-warning"
                  data-testid="session-create-providers-empty"
                >
                  No providers are configured for this workspace.
                </p>
              ) : null}
            </Field>

            {submitError ? (
              <p
                className="text-form-hint text-danger"
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
      <span className="text-danger" data-testid="session-create-catalog-refresh-error" role="alert">
        {refreshError}
      </span>
    );
  }
  if (error) {
    return (
      <span className="text-danger" data-testid="session-create-catalog-error" role="alert">
        {error}. Type a model ID to continue.
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
        No catalog models — type an exact model ID to continue.
      </span>
    );
  }
  return null;
}

export { SessionCreateDialog };
