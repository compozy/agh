import type { FormEvent } from "react";
import { Loader2 } from "lucide-react";

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
  NativeSelect,
  NativeSelectOption,
} from "@agh/ui";

import { AgentCommandSelect, AgentIcon, type AgentPayload } from "@/systems/agent";
import type { SessionProviderOption, WorkspacePayload } from "@/systems/workspace";

export interface SessionCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agents: AgentPayload[];
  workspace: WorkspacePayload | undefined;
  selectedAgentName: string;
  selectedProvider: string;
  providerOptions: SessionProviderOption[];
  providersLoading: boolean;
  providersError: string | null;
  onAgentChange: (agentName: string) => void;
  onProviderChange: (provider: string) => void;
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
  providerOptions,
  providersLoading,
  providersError,
  onAgentChange,
  onProviderChange,
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
  const hasAgents = agents.length > 0;
  const hasProviderOptions = providerOptions.length > 0;
  const hasSelectedAgent = agents.some(agent => agent.name === trimmedSelectedAgentName);
  const hasSelectedProvider = providerOptions.some(
    option => option.name === trimmedSelectedProvider
  );
  const activeProvider = workspaceSelected
    ? providerOptions.find(option => option.name === trimmedSelectedProvider)
    : undefined;
  const agentPlaceholder = !workspaceSelected
    ? "Select a workspace first"
    : hasAgents
      ? "Select an agent"
      : "No agents available";
  const providerPlaceholder = !workspaceSelected
    ? "Select a workspace first"
    : providersLoading
      ? "Loading providers…"
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

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 text-[color:var(--color-text-primary)] sm:max-w-lg"
        data-testid="session-create-dialog"
        showCloseButton={!isSubmitting}
      >
        <DialogHeader className="border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle>Start a new session</DialogTitle>
          <DialogDescription>
            {workspaceSelected
              ? `Pick the agent and provider runtime for this session in ${workspace.name}.`
              : "Choose an active workspace before starting a session."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-5 px-5 py-5">
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
              {activeAgent ? (
                <div
                  className="mt-1 flex items-center gap-1.5 text-xs text-[color:var(--color-text-tertiary)]"
                  data-testid="session-create-agent-default"
                >
                  <AgentIcon
                    className="size-3.5 text-[color:var(--color-text-tertiary)]"
                    provider={activeAgent.provider}
                  />
                  <span>Agent default provider: {activeAgent.provider}</span>
                </div>
              ) : null}
            </Field>

            <Field>
              <FieldLabel htmlFor="session-create-provider">Provider</FieldLabel>
              <FieldDescription>
                Override the runtime for this session only. The agent default is preselected when it
                matches a provider visible in this workspace.
              </FieldDescription>
              <NativeSelect
                className="w-full"
                data-testid="session-create-provider-select"
                disabled={
                  !workspaceSelected || providersLoading || !hasProviderOptions || isSubmitting
                }
                id="session-create-provider"
                onChange={event => onProviderChange(event.target.value)}
                value={workspaceSelected ? selectedProvider : ""}
              >
                {workspaceSelected && hasProviderOptions ? null : (
                  <NativeSelectOption value="">{providerPlaceholder}</NativeSelectOption>
                )}
                {providerOptions.map(option => (
                  <NativeSelectOption key={option.name} value={option.name}>
                    {providerOptionLabel(option)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
              {activeProvider ? (
                <div
                  className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-[11px] text-[color:var(--color-text-tertiary)]"
                  data-testid="session-create-provider-runtime"
                >
                  <span>{activeProvider.harness ?? "acp"}</span>
                  {activeProvider.runtime_provider ? (
                    <span>{activeProvider.runtime_provider}</span>
                  ) : null}
                  {activeProvider.default_model ? (
                    <span>{activeProvider.default_model}</span>
                  ) : null}
                </div>
              ) : null}
              {providersError ? (
                <p
                  className="mt-1 text-xs text-[color:var(--color-danger)]"
                  data-testid="session-create-providers-error"
                  role="alert"
                >
                  {providersError}
                </p>
              ) : null}
              {workspaceSelected && !providersLoading && !providersError && !hasProviderOptions ? (
                <p
                  className="mt-1 text-xs text-[color:var(--color-warning)]"
                  data-testid="session-create-providers-empty"
                >
                  No providers are configured for this workspace.
                </p>
              ) : null}
            </Field>

            {submitError ? (
              <p
                className="text-xs text-[color:var(--color-danger)]"
                data-testid="session-create-submit-error"
                role="alert"
              >
                {submitError}
              </p>
            ) : null}
          </div>

          <DialogFooter className="flex flex-wrap items-center justify-end gap-2 border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-5 py-3">
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
              {isSubmitting ? <Loader2 aria-hidden="true" className="size-4 animate-spin" /> : null}
              Start session
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function providerOptionLabel(option: SessionProviderOption): string {
  const label = option.display_name?.trim() || option.name;
  if (!option.default_model?.trim()) {
    return label;
  }
  return `${label} · ${option.default_model}`;
}

export { SessionCreateDialog };
