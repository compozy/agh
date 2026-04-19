import { Check, Loader2, Users } from "lucide-react";
import type { FormEvent } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Empty,
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  MonoBadge,
  Section,
} from "@agh/ui";
import { cn } from "@/lib/utils";
import { AgentIcon, type AgentPayload } from "@/systems/agent";

import type { NetworkCreateChannelDraft } from "../types";

interface NetworkCreateChannelDialogProps {
  agents: AgentPayload[];
  canSubmit: boolean;
  draft: NetworkCreateChannelDraft;
  isSubmitting: boolean;
  onChannelNameChange: (value: string) => void;
  onOpenChange: (open: boolean) => void;
  onSubmit: () => void;
  onToggleAgent: (agentName: string) => void;
  open: boolean;
  workspaceName?: string | null;
}

export function NetworkCreateChannelDialog({
  agents,
  canSubmit,
  draft,
  isSubmitting,
  onChannelNameChange,
  onOpenChange,
  onSubmit,
  onToggleAgent,
  open,
  workspaceName,
}: NetworkCreateChannelDialogProps) {
  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!canSubmit || isSubmitting) return;
    onSubmit();
  };

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 text-[color:var(--color-text-primary)] sm:max-w-[30rem]"
        data-testid="network-create-channel-dialog"
      >
        <DialogHeader className="border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle>Create channel</DialogTitle>
          <DialogDescription>
            {workspaceName
              ? `Spawn one new session per selected agent inside ${workspaceName}.`
              : "Create a materialized network channel by spawning one new session per selected agent."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-5 px-5 py-5">
            <Field>
              <FieldLabel htmlFor="network-channel-name">Channel name</FieldLabel>
              <FieldDescription>
                Dot-notation encouraged — e.g. coord.core, ops.alerts.
              </FieldDescription>
              <Input
                className="h-10 font-mono"
                data-testid="network-channel-name-input"
                id="network-channel-name"
                onChange={event => onChannelNameChange(event.target.value)}
                placeholder="e.g. deployments"
                value={draft.channelName}
              />
            </Field>

            <Section
              label="Add agents"
              right={
                <MonoBadge data-testid="network-selected-agents-count">
                  {draft.selectedAgentNames.length} selected
                </MonoBadge>
              }
            >
              {agents.length === 0 ? (
                <Empty
                  icon={Users}
                  title="No agents available"
                  description="This workspace has no local agents to invite."
                />
              ) : (
                <div className="overflow-hidden rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)]">
                  {agents.map(agent => {
                    const isSelected = draft.selectedAgentNames.includes(agent.name);

                    return (
                      <button
                        aria-pressed={isSelected}
                        className={cn(
                          "flex w-full items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3 text-left transition-colors last:border-b-0",
                          "hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
                          isSelected && "bg-[color:var(--color-surface)]"
                        )}
                        data-testid={`network-agent-option-${agent.name}`}
                        key={agent.name}
                        onClick={() => onToggleAgent(agent.name)}
                        type="button"
                      >
                        <span
                          aria-hidden="true"
                          className={cn(
                            "flex size-4 shrink-0 items-center justify-center rounded border",
                            isSelected
                              ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]"
                              : "border-[color:var(--color-divider)] bg-transparent text-transparent"
                          )}
                        >
                          <Check className="size-3" />
                        </span>
                        <AgentIcon
                          className="size-4 text-[color:var(--color-text-tertiary)]"
                          provider={agent.provider}
                        />
                        <span className="min-w-0 flex-1 truncate text-[13px] text-[color:var(--color-text-primary)]">
                          {agent.name}
                        </span>
                        <MonoBadge tone={isSelected ? "accent" : "default"}>
                          {agent.provider}
                        </MonoBadge>
                      </button>
                    );
                  })}
                </div>
              )}

              {!workspaceName ? (
                <p className="mt-2 text-[12px] leading-relaxed text-[color:var(--color-warning)]">
                  Select an active workspace before creating a channel.
                </p>
              ) : null}
            </Section>
          </div>

          <div className="flex flex-wrap items-center justify-end gap-2 border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-5 py-3">
            <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
              Cancel
            </Button>
            <Button
              data-testid="network-create-channel-submit"
              disabled={!canSubmit || isSubmitting}
              type="submit"
            >
              {isSubmitting ? <Loader2 aria-hidden="true" className="size-4 animate-spin" /> : null}
              Create channel
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
